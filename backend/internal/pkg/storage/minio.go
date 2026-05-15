package storage

import (
	"context"
	"github.com/cbt-ai/enterprise-cbt/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func NewMinIO(ctx context.Context, cfg config.Config) (*minio.Client, error) {
	client, err := minio.New(cfg.S3Endpoint, &minio.Options{Creds: credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""), Secure: cfg.S3UseSSL, Region: cfg.S3Region})
	if err != nil {
		return nil, err
	}
	exists, err := client.BucketExists(ctx, cfg.S3Bucket)
	if err == nil && !exists {
		err = client.MakeBucket(ctx, cfg.S3Bucket, minio.MakeBucketOptions{Region: cfg.S3Region})
	}
	return client, err
}
