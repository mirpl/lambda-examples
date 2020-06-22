package main

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

var (
	s3Region, s3Bucket string
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

func handler(evt FileSaverEvent) (*FileSaverResponse, error) {
	// Step 1 : Download file from evt.RequestURL
	parsedURL, err := getFileFromURL(evt.RequestURL)
	if err != nil {
		return nil, err
	}

	// Step 2 : Save file to S3
	s3Path, err := saveFileToS3(parsedURL)
	if err != nil {
		return nil, err
	}

	return &FileSaverResponse{
		InputURL: evt.RequestURL,
		S3Path:   s3Path,
	}, nil
}

func getFileFromURL(requestURL string) (*url.URL, error) {
	parsedURL, err := url.ParseRequestURI(requestURL)
	if err != nil {
		logger.Error("parsing request URL failed", zap.Error(err))
		return nil, err
	}

	response, err := http.Get(parsedURL.String())
	if err != nil {
		logger.Error("getting URL response failed", zap.Error(err))
		return nil, err
	}
	defer response.Body.Close()

	file, err := os.Create(path.Join("/tmp", path.Base(parsedURL.Path)))
	if err = file.Sync(); err != nil {
		logger.Error("file sync failed", zap.Error(err))
		return nil, err
	}
	defer file.Close()

	if _, err := io.Copy(file, response.Body); err != nil {
		logger.Error("dumping response body to file failed", zap.Error(err))
		return nil, err
	}
	return parsedURL, nil
}

func saveFileToS3(requestURL *url.URL) (string, error) {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(s3Region)}))
	uploader := s3manager.NewUploader(sess)

	file, err := os.Open(path.Join("/tmp", path.Base(requestURL.Path)))
	if err != nil {
		logger.Error("reading file to upload failed", zap.Error(err))
		return "", err
	}
	defer file.Close()

	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:   file,
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(path.Base(requestURL.Path)),
	})
	if err != nil {
		logger.Error("saving file to S3 failed", zap.Error(err))
	}
	return result.Location, err
}

func main() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
	if err := parseEnvVars(); err != nil {
		panic(err)
	}
	lambda.Start(handler)
}

func parseEnvVars() error {
	var err error

	s3Region = os.Getenv("S3_REGION")
	if len(s3Region) <= 0 {
		err = errors.New("S3_REGION not provided")
		logger.Error("environment variable is empty", zap.Error(err))
		return err
	}

	s3Bucket = os.Getenv("S3_BUCKET")
	if len(s3Bucket) <= 0 {
		err = errors.New("S3_BUCKET not provided")
		logger.Error("environment variable is empty", zap.Error(err))
		return err
	}
	return nil
}
