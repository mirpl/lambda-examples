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

const (
	upload   = "upload"
	download = "download"
)

type MinioGatewayEvent struct {
	FunctionType string `json:"functionType"`
	Data         string `json:"data"`
}

type MinioGatewayResponse struct {
	Message string `json:"message"`
	Data    []byte `json:"data"`
}

func handler(evt MinioGatewayEvent) (*MinioGatewayResponse, error) {
	var (
		respMsg  string
		respData []byte
		err      error
	)
	// Step 1: Invoke function based on the type name and with data provided by event
	switch evt.FunctionType {
	case upload:
		// Download file from 'evt.Data' url and upload to bucket
		respMsg, respData, err = minioUpload(evt.Data, s3Client)
	case download:
		// Download file from 'evt.Data' object name and quit
		respMsg, respData, err = minioDownload(evt.Data, s3Client)
	default:
		err = fmt.Errorf("function type \"%s\" is invalid", evt.FunctionType)
		logger.Error("function type not supported", zap.Error(err))
		return nil, err
	}
	// Step 2: Return response with message (upload/download) and data (download only)
	return &MinioGatewayResponse{
		Message: respMsg,
		Data:    respData,
	}, nil
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
	if len(accessKeyID) <= 0 {
		err = fmt.Errorf(errMsgFormat, envVarAccessKey)
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	secretAccessKey = os.Getenv(envVarSecretKey)
	if len(secretAccessKey) <= 0 {
		err = fmt.Errorf(errMsgFormat, envVarSecretKey)
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	sslEnv := os.Getenv(envVarSSL)
	if useSSL, err = strconv.ParseBool(sslEnv); err != nil {
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	bucket = os.Getenv(envVarBucket)
	if len(bucket) <= 0 {
		err = fmt.Errorf(errMsgFormat, envVarBucket)
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	region = os.Getenv(envVarRegion)
	if len(region) < 0 { // empty string results setting default region value in client.MakeBucket function
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
