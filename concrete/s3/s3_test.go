package s3_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/go-modules/concrete/s3"
)

type S3Suite struct{ suite.Suite }

func TestS3Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(S3Suite))
}

// backend builds an S3 backend pointed at a fake endpoint with static creds. Presigning is a local signing operation,
// so no network is touched.
func (s *S3Suite) backend() *s3.S3 {
	b, err := s3.New(s3.Config{
		Bucket:       "assets",
		AwsRegion:    "us-east-1",
		Key:          "AKIAEXAMPLE",
		Secret:       "secret",
		Endpoint:     "http://localhost:4566",
		UsePathStyle: true,
	})
	s.Require().NoError(err)
	return b
}

func (s *S3Suite) TestNew_MissingBucket_ReturnsError() {
	_, err := s3.New(s3.Config{AwsRegion: "us-east-1"})
	s.Require().Error(err)
	s.ErrorContains(err, "bucket")
}

func (s *S3Suite) TestNew_MissingRegion_ReturnsError() {
	_, err := s3.New(s3.Config{Bucket: "assets"})
	s.Require().Error(err)
	s.ErrorContains(err, "aws_region")
}

func (s *S3Suite) TestUploadURL_PresignsPutForPath() {
	url, err := s.backend().UploadURL(context.Background(), "uploads/file.png", 15*time.Minute)
	s.Require().NoError(err)

	s.True(strings.HasPrefix(url, "http"), "expected an absolute URL, got %q", url)
	s.Contains(url, "assets/uploads/file.png")
	s.Contains(url, "X-Amz-Signature")
}

func (s *S3Suite) TestDownloadURL_PresignsGetForPath() {
	url, err := s.backend().DownloadURL(context.Background(), "uploads/file.png", time.Hour)
	s.Require().NoError(err)

	s.True(strings.HasPrefix(url, "http"), "expected an absolute URL, got %q", url)
	s.Contains(url, "assets/uploads/file.png")
	s.Contains(url, "X-Amz-Signature")
}

func (s *S3Suite) TestUploadAndDownloadURLs_Differ() {
	b := s.backend()
	put, err := b.UploadURL(context.Background(), "f", time.Minute)
	s.Require().NoError(err)
	get, err := b.DownloadURL(context.Background(), "f", time.Minute)
	s.Require().NoError(err)

	// PUT and GET presigns sign different verbs and must not be identical.
	s.NotEqual(put, get)
}
