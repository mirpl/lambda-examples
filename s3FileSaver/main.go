package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

const (
	envVarEndpoint  = "S3_ENDPOINT"
	envVarAccessKey = "S3_ACCESS_KEY"
	envVarSecretKey = "S3_SECRET_KEY"
	envVarRegion    = "S3_REGION"
	envVarBucket    = "S3_BUCKET"
)

var (
	logger                       *zap.Logger
	uploader                     *s3manager.Uploader
	endpoint, bucket, region     *string
	accessKeyID, secretAccessKey string
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
	file, err := os.Open(path.Join("/tmp", path.Base(requestURL.Path)))
	if err != nil {
		logger.Error("reading file to upload failed", zap.Error(err))
		return "", err
	}
	defer file.Close()

	result, err := uploader.Upload(&s3manager.UploadInput{
		Body:   file,
		Bucket: bucket,
		Key:    aws.String(path.Base(requestURL.Path)),
	})
	if err != nil {
		logger.Error("saving file to S3 failed", zap.Error(err))
	}
	r := ""
	if result != nil {
		r = result.Location
	}
	return r, err
}

func main() {
	var err error
	if err != nil {
		panic(err)
	}
	if err := parseEnvVars(); err != nil {
		panic(err)
	}
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      region,
		Endpoint:    endpoint,
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	}))
	uploader = s3manager.NewUploader(sess)
	lambda.Start(handler)
}

func parseEnvVars() error {
	var err error
	loggerErrMsg := "parsing environment variable failed"
	errMsgFormat := "%s not provided"

	endpoint = aws.String(os.Getenv(envVarEndpoint))
	if *endpoint == "" {
		endpoint = nil
	}

	accessKeyID = os.Getenv(envVarAccessKey)
	if len(accessKeyID) <= 0 {
		err = errors.New(fmt.Sprintf(errMsgFormat, envVarAccessKey))
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	secretAccessKey = os.Getenv(envVarSecretKey)
	if len(secretAccessKey) <= 0 {
		err = errors.New(fmt.Sprintf(errMsgFormat, envVarSecretKey))
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	bucket = aws.String(os.Getenv(envVarBucket))
	if len(*bucket) <= 0 {
		err = errors.New(fmt.Sprintf(errMsgFormat, envVarBucket))
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}

	region = aws.String(os.Getenv(envVarRegion))
	if len(*region) <= 0 {
		err = errors.New(fmt.Sprintf(errMsgFormat, envVarRegion))
		logger.Error(loggerErrMsg, zap.Error(err))
		return err
	}
	return nil
}
