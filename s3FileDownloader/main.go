package main

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
	downloader                   *s3manager.Downloader
	endpoint, bucket, region     *string
	accessKeyID, secretAccessKey string
)

type FileDownloaderEvent struct {
	S3FileKey string `json:"s3FileKey"`
}

type FileDownloaderResponse struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Content  []byte `json:"content"`
}

func handler(evt FileDownloaderEvent) (*FileDownloaderResponse, error) {
	// Step 1: Create a file to write to
	file, err := os.Create(path.Join("/tmp", evt.S3FileKey))
	if err != nil {
		logger.Error("creating file failed", zap.Error(err))
		return nil, err
	}
	defer file.Close()

	// Step 2: Get object and write its data to the created file
	if _, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: bucket,
		Key:    aws.String(evt.S3FileKey),
	}); err != nil {
		logger.Error("download from S3 failed", zap.Error(err))
		return nil, err
	}
	// Step 3: Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		logger.Error("getting file stat failed", zap.Error(err))
		return nil, err
	}
	//Step 4: Create a buffer to read file into
	buffer := make([]byte, fileInfo.Size())
	if _, err = file.Read(buffer); err != nil {
		logger.Error("reading file failed", zap.Error(err))
		return nil, err
	}

	return &FileDownloaderResponse{
		Filename: file.Name(),
		Size:     fileInfo.Size(),
		Content:  buffer,
	}, err
}

func main() {
	var err error
	if logger, err = zap.NewProduction(); err != nil {
		panic(err)
	}
	if err = parseEnvVars(); err != nil {
		panic(err)

	}
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      region,
		Endpoint:    endpoint,
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	}))
	downloader = s3manager.NewDownloader(sess)
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
