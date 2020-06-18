package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/minio/minio-go/v6"
	"go.uber.org/zap"
)

const (
	endpoint        = "s3.amazonaws.com"
	accessKeyID     = "AKIAWVAPK7HAZEHHTJL7"
	secretAccessKey = "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"
	useSSL          = true
	bucket          = "mvp-file-storage"
	region          = "us-east-1"
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

var (
	logger *zap.Logger
)

type MinioGatewayEvent struct {
	FunctionType string `json:"functionType"`
	Data         string `json:"data"`
}

type MinioGatewayResponse struct {
	Message string `json:"message"`
	Data    []byte `json:"data"`
}

func handler(evt MinioGatewayEvent) (MinioGatewayResponse, error) {
	s3Client, err := minioGatewayClient()
	if err != nil {
		return MinioGatewayResponse{}, err
	}
	if err = checkBucket(s3Client); err != nil {
		return MinioGatewayResponse{}, err
	}
	respMsg, respData, err := invokeFunction(evt.FunctionType, evt.Data, s3Client)
	if err != nil {
		return MinioGatewayResponse{}, err
	}
	return MinioGatewayResponse{
		Message: respMsg,
		Data:    respData,
	}, nil
}

func minioGatewayClient() (*minio.Client, error) {
	s3Client, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		logger.Error("MinIO S3 client initialization failed", zap.Error(err))
		return nil, err
	}
	return s3Client, nil
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

func invokeFunction(name, data string, client *minio.Client) (string, []byte, error) {
	f, ok := functions[toID[name]]
	if !ok {
		err := fmt.Errorf("function \"%s\" doesn't exist", name)
		logger.Error("invoking function failed", zap.Error(err))
		return "", nil, err
	}
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
	lambda.Start(handler)
}
