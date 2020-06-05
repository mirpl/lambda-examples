package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.uber.org/zap"
)

const (
	S3Region = "us-east-1"
	S3Bucket = "mvp-file-storage"
)

var (
	log *zap.Logger
)

type fileSaverEvent struct {
	requestURL string `json:"requestUrl"`
}

type fileSaverResponse struct {
	inputURL string `json:"inputUrl"`
	S3path   string `json:"s3path"`
}

func getFileFromURL(requestURL string) (*os.File, error) {
	if _, err := url.ParseRequestURI(requestURL); err != nil {
		log.Error("parsing request URL failed", zap.Error(err))
		return nil, err
	}

	response, err := http.Get(requestURL)
	if err != nil {
		log.Error("getting URL response failed", zap.Error(err))
		return nil, err
	}
	defer response.Body.Close()

	file, err := os.Create(filepath.Base(requestURL))
	if err != nil {
		log.Error("creating file failed", zap.Error(err))
		return nil, err
	}

	if _, err = io.Copy(file, response.Body); err != nil {
		log.Error("dumping response body to file failed", zap.Error(err))
		return nil, err
	}
	return file, nil
}

func saveFileToS3(file *os.File, requestURL string) (string, error) {
	s, err := session.NewSession(&aws.Config{Region: aws.String(S3Region)})
	if err != nil {
		log.Error("creating new S3 session failed", zap.Error(err))
		return "", err
	}

	fileInfo, _ := file.Stat()
	var size = fileInfo.Size()
	buffer := make([]byte, size)
	if _, err := file.Read(buffer); err != nil {
		log.Error("reading buffer failed", zap.Error(err))
		return "", err
	}

	s3path := fmt.Sprintf("s3.%s", requestURL)
	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(S3Bucket),
		Key:                  aws.String(s3path),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	if err != nil {
		log.Error("saving file to S3 failed", zap.Error(err))
	}
	return s3path, err
}

func handler(ctx context.Context, evt fileSaverEvent) (fileSaverResponse, error) {
	file, err := getFileFromURL(evt.requestURL)
	if err != nil {
		return fileSaverResponse{}, err
	}
	defer file.Close()

	s3path, err := saveFileToS3(file, evt.requestURL)
	if err != nil {
		return fileSaverResponse{}, err
	}

	return fileSaverResponse{
		inputURL: evt.requestURL,
		S3path:   s3path,
	}, nil
}

func main() {
	lambda.Start(handler)
}
