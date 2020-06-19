package main

import (
	"errors"
	"os"
	"path"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

var (
	logger             *zap.Logger
	downloader         *s3manager.Downloader
	s3Region, s3Bucket string
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
		Bucket: aws.String(s3Bucket),
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
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(s3Region)}))
	downloader = s3manager.NewDownloader(sess)
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
