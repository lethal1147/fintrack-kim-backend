package r2client

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Uploader is the interface ProfileService depends on, allowing test mocks.
type Uploader interface {
	Upload(ctx context.Context, key, contentType string, data io.Reader) (string, error)
}

// Client is a thin S3-compatible wrapper for Cloudflare R2.
type Client struct {
	s3c       *s3.Client
	bucket    string
	publicURL string
}

// New creates an R2 Client. Returns nil only if accountID is empty (R2 not configured).
func New(accountID, accessKey, secretKey, bucket, publicURL string) *Client {
	if accountID == "" {
		return nil
	}
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	s3c := s3.NewFromConfig(
		aws.Config{
			Region:      "auto",
			Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		},
		func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		},
	)
	return &Client{s3c: s3c, bucket: bucket, publicURL: publicURL}
}

// Upload stores data in R2 under key and returns the public URL.
func (c *Client) Upload(ctx context.Context, key, contentType string, data io.Reader) (string, error) {
	_, err := c.s3c.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Body:        data,
	})
	if err != nil {
		return "", fmt.Errorf("r2 upload: %w", err)
	}
	return c.publicURL + "/" + key, nil
}

// RandomKey generates a random hex key with the given extension (e.g. ".jpg").
func RandomKey(prefix, ext string) string {
	b := make([]byte, 16)
	rand.Read(b) //nolint:gosec — not used for cryptographic purpose
	return prefix + hex.EncodeToString(b) + ext
}
