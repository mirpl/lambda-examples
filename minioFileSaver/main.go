package main

import (
	"bytes"
	"context"
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
	log *zap.Logger
)

type fileSaverEvent struct {
	requestURL string `json:"requestUrl"`
}

type fileSaverResponse struct {
	inputURL string `json:"inputUrl"`
	s3Path   string `json:"s3Path"`
}

func getFileFromURL(requestURL string) (*os.File, *url.URL, error) {
	parsedURL, err := url.ParseRequestURI(requestURL)
	if err != nil {
		log.Error("parsing request URL failed", zap.Error(err))
		return nil, nil, err
	}

	response, err := http.Get(requestURL)
	if err != nil {
		log.Error("getting URL response failed", zap.Error(err))
		return nil, nil, err
	}
	defer response.Body.Close()

	file, err := os.Create(path.Base(parsedURL.Path))
	if err != nil {
		log.Error("creating file failed", zap.Error(err))
		return nil, nil, err
	}

	if _, err = io.Copy(file, response.Body); err != nil {
		log.Error("dumping response body to file failed", zap.Error(err))
		return nil, nil, err
	}
	return file, parsedURL, nil
}

func saveFileToS3(file *os.File, requestURL *url.URL) (string, error) {
	s, err := session.NewSession(&aws.Config{Region: aws.String(s3Region)})
	if err != nil {
		log.Error("creating new AWS session failed", zap.Error(err))
		return "", err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		log.Error("getting file info failed", zap.Error(err))
		return "", err

	}
	var size = fileInfo.Size()
	buffer := make([]byte, size)
	if _, err := file.Read(buffer); err != nil {
		log.Error("reading buffer failed", zap.Error(err))
		return "", err
	}

	s3KeyName := path.Base(requestURL.Path)
	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(s3Bucket),
		Key:                  aws.String(s3KeyName),
		ACL:                  aws.String("public"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	if err != nil {
		log.Error("saving file to S3 failed", zap.Error(err))
	}
	s3Path := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s3Bucket, s3Region, s3KeyName)
	return s3Path, err
}

func handler(_ context.Context, evt fileSaverEvent) (fileSaverResponse, error) {

	file, parsedURL, err := getFileFromURL(evt.requestURL)
	if err != nil {
		return fileSaverResponse{}, err
	}
	defer file.Close()

	s3Path, err := saveFileToS3(file, parsedURL)
	if err != nil {
		return fileSaverResponse{}, err
	}

	return fileSaverResponse{
		inputURL: evt.requestURL,
		s3Path:   s3Path,
	}, nil
}

func main() {
	lambda.Start(handler)
}
