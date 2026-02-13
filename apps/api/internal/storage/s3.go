package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const maxAttachmentSize = 10 * 1024 * 1024 // 10MB

type S3Client struct {
	bucket     string
	cdnBaseURL string
	presign    *s3.PresignClient
	client     *s3.Client
	presignDur time.Duration
}

type S3Config struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	PresignExpiry   time.Duration
	CDNBaseURL      string
}

func NewS3Client(cfg S3Config) (*S3Client, error) {
	presignDur := cfg.PresignExpiry
	if presignDur <= 0 {
		presignDur = 15 * time.Minute
	}

	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		))
	}

	if cfg.Endpoint != "" {
		opts = append(opts, config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:               cfg.Endpoint,
					SigningRegion:     cfg.Region,
					HostnameImmutable: true,
				}, nil
			}),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)

	return &S3Client{
		bucket:     cfg.Bucket,
		cdnBaseURL: cfg.CDNBaseURL,
		presign:    s3.NewPresignClient(client, s3.WithPresignExpires(presignDur)),
		client:     client,
		presignDur: presignDur,
	}, nil
}

func (c *S3Client) GenerateUploadURL(ctx context.Context, key string, size int64) (string, error) {
	if size <= 0 || size > maxAttachmentSize {
		return "", fmt.Errorf("invalid size: must be 1-%d bytes", maxAttachmentSize)
	}

	presigned, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(key),
		ContentLength: aws.Int64(size),
		ContentType:   aws.String("application/octet-stream"),
	})
	if err != nil {
		return "", fmt.Errorf("presign put object: %w", err)
	}

	return presigned.URL, nil
}

func (c *S3Client) GetPublicURL(key string) string {
	base := strings.TrimSuffix(c.cdnBaseURL, "/")
	return base + "/" + key
}

func (c *S3Client) DeleteObjects(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	objects := make([]types.ObjectIdentifier, len(keys))
	for i, k := range keys {
		objects[i] = types.ObjectIdentifier{Key: aws.String(k)}
	}

	_, err := c.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(c.bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	})
	if err != nil {
		return fmt.Errorf("delete objects: %w", err)
	}

	return nil
}
