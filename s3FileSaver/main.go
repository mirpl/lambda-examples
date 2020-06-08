package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.uber.org/zap"
)

const (
	s3Region = "us-east-1"
	s3Bucket = "mvp-file-storage"
)

var (
	logger *zap.Logger
)

type FileSaverEvent struct {
	RequestURL string `json:"requestUrl"`
}

type FileSaverResponse struct {
	InputURL string `json:"inputUrl"`
	S3Path   string `json:"s3Path"`
}

func getFileFromURL(requestURL string) (*os.File, *url.URL, error) {
	parsedURL, err := url.ParseRequestURI(requestURL)
	if err != nil {
		logger.Error("parsing request URL failed", zap.Error(err))
		return nil, nil, err
	}

	response, err := http.Get(requestURL)
	if err != nil {
		logger.Error("getting URL response failed", zap.Error(err))
		return nil, nil, err
	}
	defer response.Body.Close()

	file, err := os.Create(path.Join("/tmp", path.Base(parsedURL.Path)))
	if err != nil {
		logger.Error("creating file failed", zap.Error(err))
		return nil, nil, err
	}

	if _, err = io.Copy(file, response.Body); err != nil {
		logger.Error("dumping response body to file failed", zap.Error(err))
		return nil, nil, err
	}
	return file, parsedURL, nil
}

func saveFileToS3(file *os.File, requestURL *url.URL) (string, error) {
	s, err := session.NewSession(&aws.Config{Region: aws.String(s3Region)})
	if err != nil {
		logger.Error("creating new AWS session failed", zap.Error(err))
		return "", err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		logger.Error("getting file info failed", zap.Error(err))
		return "", err

	}
	var size = fileInfo.Size()
	buffer := make([]byte, size)
	if _, err := file.Read(buffer); err != nil && err != io.EOF {
		logger.Error("reading buffer failed", zap.Error(err))
		return "", err
	}

	s3KeyName := path.Base(requestURL.Path)
	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(s3Bucket),
		Key:                  aws.String(s3KeyName),
		ACL:                  aws.String("public-read"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	if err != nil {
		logger.Error("saving file to S3 failed", zap.Error(err))
	}
	s3Path := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s3Bucket, s3Region, s3KeyName)
	return s3Path, err
}

func handler(evt FileSaverEvent) (FileSaverResponse, error) {
	file, parsedURL, err := getFileFromURL(evt.RequestURL)
	if err != nil {
		return FileSaverResponse{}, err
	}
	defer file.Close()

	s3Path, err := saveFileToS3(file, parsedURL)
	if err != nil {
		return FileSaverResponse{}, err
	}

	return FileSaverResponse{
		InputURL: evt.RequestURL,
		S3Path:   s3Path,
	}, nil
}

func main() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
	lambda.Start(handler)
}
