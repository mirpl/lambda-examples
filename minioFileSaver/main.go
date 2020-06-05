package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-lambda-go/lambda"
)

type FileServerEvent struct {
	URL string `json:"url"`
}

func handler(ctx context.Context, evt FileServerEvent) (string, error) {
	_, err := url.ParseRequestURI(evt.URL)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Hello %s!", evt.URL), nil
}

func main() {
	lambda.Start(handler)
}
