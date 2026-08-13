// Package storage is a retirement half-state: uploads moved to the artifact
// service, so only Delete is still reachable. It exists for the pre-cutover
// `galgame_toolset_resource` rows whose type is still 's3' (81 in production as
// of 2026-08). Drop the package once that count reaches zero.
package storage

import (
	"context"
	"fmt"
	"log/slog"

	"kun-galgame-api/pkg/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client *s3.Client
	bucket string
}

func NewS3(cfg config.S3Config) *S3Client {
	if cfg.Endpoint == "" {
		slog.Warn("S3 未配置，图片上传功能不可用")
		return nil
	}

	resolver := aws.EndpointResolverWithOptionsFunc(
		func(service, region string, options ...any) (aws.Endpoint, error) {
			return aws.Endpoint{URL: cfg.Endpoint}, nil
		},
	)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithEndpointResolverWithOptions(resolver),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		panic(fmt.Sprintf("初始化 S3 失败: %v", err))
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	slog.Info("S3 客户端初始化成功")
	return &S3Client{client: client, bucket: cfg.Bucket}
}

func (s *S3Client) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	return err
}
