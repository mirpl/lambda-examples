package main

import (
	"os"
	"path"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

const (
	s3Region = "us-east-1"
	s3Bucket = "mvp-file-storage"
)

var (
	logger *zap.Logger
)

type FileDownloaderEvent struct {
	S3FileKey string `json:"s3FileKey"`
}

type FileDownloaderResponse struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Content  []byte `json:"content"`
}

func downloadFileFromS3(s3key string) (FileDownloaderResponse, error) {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(s3Region)}))
	downloader := s3manager.NewDownloader(sess)

	file, err := os.Create(path.Join("/tmp", s3key))
	if err != nil {
		logger.Error("creating file failed", zap.Error(err))
		return FileDownloaderResponse{}, err
	}
	defer file.Close()

	if _, err = downloader.Download(
		file, (&s3.GetObjectInput{}).
			SetBucket(s3Bucket).
			SetKey(s3key),
	); err != nil {
		logger.Error("download from S3 failed", zap.Error(err))
		return FileDownloaderResponse{}, err
	}
	fileInfo, err := file.Stat()
	if err != nil {
		logger.Error("getting file stat failed", zap.Error(err))
		return FileDownloaderResponse{}, err
	}
	buffer := make([]byte, fileInfo.Size())
	if _, err = file.Read(buffer); err != nil {
		logger.Error("reading file failed", zap.Error(err))
		return FileDownloaderResponse{}, err
	}

	return FileDownloaderResponse{
		Filename: file.Name(),
		Size:     fileInfo.Size(),
		Content:  buffer,
	}, err
}

func handler(evt FileDownloaderEvent) (FileDownloaderResponse, error) {
	return downloadFileFromS3(evt.S3FileKey)
}

func main() {
	var err error
	if logger, err = zap.NewProduction(); err != nil {
		panic(err)
	}
	lambda.Start(handler)
}
