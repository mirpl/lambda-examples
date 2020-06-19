package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/minio/minio-go/v6"
	"go.uber.org/zap"
)

const (
	envVarEndpoint  = "MINIO_ENDPOINT"
	envVarAccessKey = "MINIO_ACCESSKEY"
	envVarSecretKey = "MINIO_SECRETKEY"
	envVarSSL       = "MINIO_USESSL"
	envVarBucket    = "MINIO_BUCKETNAME"
	envVarRegion    = "MINIO_LOCATION"
)

var (
	logger                                                 *zap.Logger
	s3Client                                               *minio.Client
	endpoint, accessKeyID, secretAccessKey, bucket, region string
	useSSL                                                 bool
)

type fName int

const (
	upload fName = iota
	download
)

var toID = map[string]fName{
	"upload":   upload,
	"download": download,
}

var functions = map[fName]interface{}{upload: minioUpload, download: minioDownload}

type MinioGatewayEvent struct {
	FunctionType string `json:"functionType"`
	Data         string `json:"data"`
}

type MinioGatewayResponse struct {
	Message string `json:"message"`
	Data    []byte `json:"data"`
}

func handler(evt MinioGatewayEvent) (*MinioGatewayResponse, error) {
	// Step 1: Invoke function type based on the name and with data provided by event
	respMsg, respData, err := invokeFunction(evt.FunctionType, evt.Data, s3Client)
	if err != nil {
		return nil, err
	}
	// Step 4: Return response with message (upload/download) and data (download only)
	return &MinioGatewayResponse{
		Message: respMsg,
		Data:    respData,
	}, nil
}

func invokeFunction(name, data string, client *minio.Client) (string, []byte, error) {
	// Step 2: Get function mapped by type names
	f, ok := functions[toID[name]]
	if !ok {
		err := fmt.Errorf("function \"%s\" doesn't exist", name)
		logger.Error("invoking function failed", zap.Error(err))
		return "", nil, err
	}
	// Step 3: Invoke appropriate function
	respMsg, respData, err := f.(func(string, *minio.Client) (string, []byte, error))(data, client)
	if err != nil {
		logger.Error("invoking function failed", zap.Error(err))
		return "", nil, err
	}
	return respMsg, respData, nil
}

func minioUpload(urlAddr string, client *minio.Client) (string, []byte, error) {
	parsedURL, err := url.ParseRequestURI(urlAddr)
	if err != nil {
		logger.Error("parsing request URL failed", zap.Error(err))
		return "", nil, err
	}
	resp, err := http.Get(parsedURL.String())
	if err != nil {
		logger.Error("getting URL response failed", zap.Error(err))
		return "", nil, err
	}
	defer resp.Body.Close()

	filename := path.Base(parsedURL.Path)
	n, err := client.PutObject(bucket, filename, resp.Body, resp.ContentLength, minio.PutObjectOptions{})
	if err != nil {
		logger.Error("putting object failed", zap.Error(err))
		return "", nil, err
	}
	return fmt.Sprintf("written bytes: %d", n), nil, nil
}

func minioDownload(key string, client *minio.Client) (string, []byte, error) {
	obj, err := client.GetObject(bucket, key, minio.GetObjectOptions{})
	if err != nil {
		logger.Error("getting object failed", zap.Error(err))
		return "", nil, err
	}
	defer obj.Close()

	objInfo, err := obj.Stat()
	if err != nil {
		logger.Error("getting object info failed", zap.Error(err))
		return "", nil, err
	}

	buffer := make([]byte, objInfo.Size)
	n, err := obj.Read(buffer)
	if err != nil && err != io.EOF {
		logger.Error("reading object failed", zap.Error(err))
		return "", nil, err
	}

	return fmt.Sprintf("read bytes: %d", n), buffer, nil
}

func main() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
	if err = parseEnvVars(); err != nil {
		panic(err)
	}
	s3Client, err = minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		panic(err)
	}
	if err = checkBucket(s3Client); err != nil {
		panic(err)
	}
	lambda.Start(handler)
}

func parseEnvVars() error {
	var err error
	loggerErrMsg := "parsing environment variable failed"
	errMsgFormat := "%s not provided"

	endpoint = os.Getenv(envVarEndpoint)
	if len(endpoint) <= 0 {
		err = fmt.Errorf(errMsgFormat, envVarEndpoint)
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	accessKeyID = os.Getenv(envVarAccessKey)
	if len(endpoint) <= 0 {
		err = fmt.Errorf(errMsgFormat, envVarAccessKey)
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	secretAccessKey = os.Getenv(envVarSecretKey)
	if len(endpoint) <= 0 {
		err = fmt.Errorf(errMsgFormat, envVarSecretKey)
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	useSSL = os.Getenv(envVarSSL)
	if ok, err := strconv.ParseBool(endpoint); !ok {
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	bucket = os.Getenv(envVarBucket)
	if len(endpoint) <= 0 {
		err = fmt.Errorf(errMsgFormat, envVarBucket)
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	region = os.Getenv(envVarRegion)
	if len(endpoint) <= 0 {
		err = fmt.Errorf(errMsgFormat, envVarRegion)
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}
	return nil
}

func checkBucket(client *minio.Client) error {
	exist, err := client.BucketExists(bucket)
	if err != nil {
		logger.Error("checking bucket existence failed", zap.Error(err))
		return err
	}
	if exist {
		return nil
	}
	if err = client.MakeBucket(bucket, region); err != nil {
		logger.Error("creating bucket failed", zap.Error(err))
	}
	return err
}
