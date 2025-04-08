// internal/repository/s3_repository.go
package repository

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"image-upload-server/internal/config"
)

// S3Repository handles interactions with the S3 storage
type S3Repository struct {
	client *s3.Client
	cfg    config.S3Config
}

// NewS3Repository creates a new S3 repository
func NewS3Repository(cfg config.S3Config) (*S3Repository, error) {
	client, err := createS3Client(cfg)
	if err != nil {
		return nil, err
	}

	return &S3Repository{
		client: client,
		cfg:    cfg,
	}, nil
}

// UploadFile uploads a file to S3 and returns its URL
func (r *S3Repository) UploadFile(fileBytes []byte, fileName string, contentType string) (string, error) {
	ctx := context.Background()

	// Upload to S3
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.cfg.BucketName),
		Key:         aws.String(fileName),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", err
	}

	// Generate URL for the uploaded file
	var imageURL string
	if r.cfg.Endpoint != "" {
		// For custom S3 endpoint
		imageURL = fmt.Sprintf("%s/%s/%s", r.cfg.Endpoint, r.cfg.BucketName, fileName)
	} else {
		// For AWS S3
		imageURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", r.cfg.BucketName, r.cfg.Region, fileName)
	}

	return imageURL, nil
}

// GetFile checks if a file exists in S3
func (r *S3Repository) GetFile(fileName string) (bool, error) {
	ctx := context.Background()
	_, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.cfg.BucketName),
		Key:    aws.String(fileName),
	})

	if err != nil {
		return false, err
	}

	return true, nil
}

// ListFiles lists all files in the S3 bucket
func (r *S3Repository) ListFiles() ([]string, error) {
	ctx := context.Background()

	resp, err := r.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(r.cfg.BucketName),
	})

	if err != nil {
		return nil, err
	}

	var filenames []string
	for _, obj := range resp.Contents {
		filenames = append(filenames, *obj.Key)
	}

	return filenames, nil
}

// Helper function to create an S3 client
func createS3Client(cfg config.S3Config) (*s3.Client, error) {
	var awsCfg aws.Config
	var err error

	if cfg.Endpoint != "" {
		// Using custom endpoint (like MinIO or LocalStack)
		customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cfg.Endpoint,
				HostnameImmutable: true,
				SigningRegion:     cfg.Region,
			}, nil
		})

		awsCfg, err = awsconfig.LoadDefaultConfig(context.TODO(),
			awsconfig.WithRegion(cfg.Region),
			awsconfig.WithEndpointResolverWithOptions(customResolver),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			)),
		)
	} else {
		// Using standard AWS S3
		awsCfg, err = awsconfig.LoadDefaultConfig(context.TODO(),
			awsconfig.WithRegion(cfg.Region),
			awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			)),
		)
	}

	if err != nil {
		log.Printf("Failed to load AWS configuration: %v", err)
		return nil, err
	}

	return s3.NewFromConfig(awsCfg), nil
}
