// Package blob implements a thin wrapper around SeaweedFS' S3-compatible API.
//
// SeaweedFS exposes an S3 endpoint at port 8333 (in our compose). We use
// aws-sdk-go-v2 with path-style addressing and a static credentials provider
// pointed at the SeaweedFS endpoint.
package blob

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/cloudreport/api/internal/config"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/rs/zerolog/log"
)

type Store struct {
	client *s3.Client
	bucket string
}

func New(ctx context.Context, cfg *config.Config) (*Store, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws cfg: %w", err)
	}

	cli := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		o.UsePathStyle = true
	})

	bs := &Store{client: cli, bucket: cfg.S3Bucket}
	// Try to ensure bucket exists; SeaweedFS auto-creates on PutObject but be
	// explicit to fail early if creds are wrong.
	if err := bs.ensureBucket(ctx); err != nil {
		log.Warn().Err(err).Msg("ensureBucket (continuing, SeaweedFS may auto-create)")
	}
	return bs, nil
}

func (s *Store) ensureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err == nil {
		return nil
	}
	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)})
	var owned *types.BucketAlreadyOwnedByYou
	if errors.As(err, &owned) {
		return nil
	}
	return err
}

// Put uploads data and returns the object key (relative path inside the
// bucket). If key is empty, a key is generated automatically.
func (s *Store) Put(ctx context.Context, prefix, key string, body []byte, contentType string) (string, error) {
	if key == "" {
		key = fmt.Sprintf("%s/%s-%s", prefix, time.Now().UTC().Format("20060102"), randKey())
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (s *Store) Get(ctx context.Context, key string) ([]byte, string, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", err
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, "", err
	}
	ct := ""
	if out.ContentType != nil {
		ct = *out.ContentType
	}
	return data, ct, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// Sum256 returns the SHA-256 hex digest of the given bytes (used for asset
// integrity and de-duplication).
func Sum256(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func randKey() string {
	id, _ := gonanoid.New(16)
	return id
}
