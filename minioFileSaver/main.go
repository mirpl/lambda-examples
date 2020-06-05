package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	//"log"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	S3_REGION = "us-east-1"
	S3_BUCKET = "mvp-file-storage"
)

type FileServerEvent struct {
	URL string `json:"url"`
}

func handler(ctx context.Context, evt FileServerEvent) (string, error) {
	if _, err := url.ParseRequestURI(evt.URL); err != nil {
		return "", err
	}

	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := session.NewSession(&aws.Config{Region: aws.String(S3_REGION)})
	if err != nil {
		log.Fatal(err)
	}

	// Upload
	if err = SaveFileToS3(s, evt.URL); err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("Hello %s!", evt.URL), nil
}

func SaveFileToS3(s *session.Session, url string) error {

	response, e := http.Get(url)
	if e != nil {
		log.Fatal(e)
	}
	defer response.Body.Close()

	file, err := os.Create("/tmp/farmer.jpg")
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err = io.Copy(file, response.Body); err != nil {
		log.Fatal(err)
	}

	fileInfo, _ := file.Stat()
	var size = fileInfo.Size()
	buffer := make([]byte, size)
	if _, err := file.Read(buffer); err != nil {
		return err
	}

	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(S3_BUCKET),
		Key:                  aws.String(url),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})
	return err
}

func main() {
	lambda.Start(handler)
}
