// Package s3 implements the storage contract with AWS S3 (or a compatible API such as LocalStack/MinIO).
package s3

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/cockroachdb/errors"
	"github.com/tupicapp/go-modules/contract/storage"
)

// Config holds S3 connection settings. Services embed it in their config structs; the mapstructure tags live here so
// every service shares the same config schema.
type Config struct {
	Bucket       string `mapstructure:"bucket"`
	AwsRegion    string `mapstructure:"aws_region"`
	Key          string `mapstructure:"key"`
	Secret       string `mapstructure:"secret"`
	Endpoint     string `mapstructure:"endpoint"`
	UsePathStyle bool   `mapstructure:"use_path_style"`
	BaseURL      string `mapstructure:"base_url"`
}

// S3 is a per-region storage backend backed by AWS S3 (or a compatible API).
type S3 struct {
	bucket  string
	region  string
	baseURL string
	client  *awss3.Client
	presign *awss3.PresignClient
}

// New creates an S3 backend. Uses static credentials if Key/Secret are set; otherwise falls back to the default AWS
// credential chain.
func New(sc Config) (*S3, error) {
	if sc.Bucket == "" {
		return nil, errors.New("s3: bucket is required")
	}
	if sc.AwsRegion == "" {
		return nil, errors.New("s3: aws_region is required")
	}

	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(sc.AwsRegion),
	}
	if sc.Key != "" && sc.Secret != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(sc.Key, sc.Secret, ""),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, errors.Wrap(err, "s3: load aws config")
	}

	var clientOpts []func(*awss3.Options)
	if sc.Endpoint != "" {
		endpoint := sc.Endpoint
		clientOpts = append(clientOpts, func(o *awss3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
		})
	}
	if sc.UsePathStyle {
		clientOpts = append(clientOpts, func(o *awss3.Options) {
			o.UsePathStyle = true
		})
	}

	client := awss3.NewFromConfig(awsCfg, clientOpts...)

	return &S3{
		bucket:  sc.Bucket,
		region:  sc.AwsRegion,
		baseURL: sc.BaseURL,
		client:  client,
		presign: awss3.NewPresignClient(client),
	}, nil
}

// UploadURL returns a presigned PUT URL for direct client upload to S3.
func (s *S3) UploadURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	req, err := s.presign.PresignPutObject(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}, awss3.WithPresignExpires(expiry))
	if err != nil {
		return "", errors.Wrap(err, "s3: presign put object")
	}
	return req.URL, nil
}

// DownloadURL returns a presigned GET URL for direct client download from S3.
func (s *S3) DownloadURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	req, err := s.presign.PresignGetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	}, awss3.WithPresignExpires(expiry))
	if err != nil {
		return "", errors.Wrap(err, "s3: presign get object")
	}
	return req.URL, nil
}

// Delete removes an object from the bucket.
func (s *S3) Delete(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	return errors.WithStack(err)
}

// Move relocates an object within the bucket via server-side copy + delete.
func (s *S3) Move(ctx context.Context, src, dst string) error {
	copySource := url.PathEscape(s.bucket) + "/" + url.PathEscape(src)
	if _, err := s.client.CopyObject(ctx, &awss3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		CopySource: aws.String(copySource),
		Key:        aws.String(dst),
	}); err != nil {
		return errors.Wrap(err, "s3: copy object")
	}
	if _, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(dst),
	}); err != nil {
		if streamErr := s.copyObjectByStreaming(ctx, src, dst); streamErr != nil {
			return errors.Wrap(errors.CombineErrors(err, streamErr), "s3: verify copied object")
		}
	}
	if _, err := s.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(src),
	}); err != nil {
		return errors.Wrap(err, "s3: delete source after copy")
	}
	return nil
}

func (s *S3) copyObjectByStreaming(ctx context.Context, src, dst string) (err error) {
	get, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(src),
	})
	if err != nil {
		return errors.Wrap(err, "s3: get source object")
	}
	defer func() { err = errors.CombineErrors(err, get.Body.Close()) }()
	body, err := io.ReadAll(get.Body)
	if err != nil {
		return errors.Wrap(err, "s3: read source object")
	}

	if _, err = s.client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(dst),
		Body:   bytes.NewReader(body),
	}); err != nil {
		return errors.Wrap(err, "s3: put destination object")
	}

	return nil
}

var _ storage.Storage = (*S3)(nil)
