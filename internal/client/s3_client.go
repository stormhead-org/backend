package client

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Client is a client for interacting with an S3-compatible object store.
type S3Client struct {
	s3Client *s3.Client
}

// NewS3Client creates a new S3Client.
func NewS3Client(ctx context.Context) (*S3Client, error) {
	// Load the AWS configuration from environment variables, shared config files, etc.
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS SDK config: %w", err)
	}

	// Create an S3 client
	s3Client := s3.NewFromConfig(cfg)

	return &S3Client{
		s3Client: s3Client,
	}, nil
}

// UploadFile uploads a file to S3.
func (c *S3Client) UploadFile(ctx context.Context, bucket, key string, data io.Reader) error {
	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}
	return nil
}
