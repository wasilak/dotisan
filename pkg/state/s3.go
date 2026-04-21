package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/wasilak/dotisan/pkg/provider"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Backend implements StateBackend for S3-compatible storage.
type S3Backend struct {
	client   *minio.Client
	bucket   string
	key      string
	endpoint string
	useSSL   bool
}

// S3Config holds configuration for S3Backend.
type S3Config struct {
	// Endpoint is the S3 endpoint (e.g., "s3.amazonaws.com" or "play.minio.io")
	Endpoint string

	// Bucket is the S3 bucket name
	Bucket string

	// Key is the object key (path) for the state file
	Key string

	// Region is the AWS region (optional, for AWS S3)
	Region string

	// AccessKeyID is the AWS access key or S3 access key
	AccessKeyID string

	// SecretAccessKey is the AWS secret key or S3 secret key
	SecretAccessKey string

	// UseSSL indicates whether to use HTTPS
	UseSSL bool
}

// NewS3Backend creates a new S3Backend with the given configuration.
func NewS3Backend(cfg S3Config) (*S3Backend, error) {
	// Create minio client options
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	}

	if cfg.Region != "" {
		opts.Region = cfg.Region
	}

	// Create minio client
	client, err := minio.New(cfg.Endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &S3Backend{
		client:   client,
		bucket:   cfg.Bucket,
		key:      cfg.Key,
		endpoint: cfg.Endpoint,
		useSSL:   cfg.UseSSL,
	}, nil
}

// Load retrieves the state from S3 storage.
func (b *S3Backend) Load(ctx context.Context) (*State, error) {
	// Get object from S3
	obj, err := b.client.GetObject(ctx, b.bucket, b.key, minio.GetObjectOptions{})
	if err != nil {
		// Check if object doesn't exist
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return NewState(), nil
		}
		return nil, fmt.Errorf("failed to get state from S3: %w", err)
	}
	defer obj.Close()

	// Read object data
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(obj); err != nil {
		return nil, fmt.Errorf("failed to read state from S3: %w", err)
	}

	// If empty, return new state
	if buf.Len() == 0 {
		return NewState(), nil
	}

	// Parse JSON
	var state State
	if err := json.Unmarshal(buf.Bytes(), &state); err != nil {
		return nil, fmt.Errorf("failed to parse state from S3: %w", err)
	}

	// Initialize empty resources slice if nil
	if state.Resources == nil {
		state.Resources = []provider.ResourceState{}
	}

	return &state, nil
}

// Save persists the state to S3 storage.
func (b *S3Backend) Save(ctx context.Context, s *State) error {
	// Marshal to JSON
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Upload to S3
	_, err = b.client.PutObject(ctx, b.bucket, b.key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to save state to S3: %w", err)
	}

	return nil
}

// Bucket returns the S3 bucket name.
func (b *S3Backend) Bucket() string {
	return b.bucket
}

// Key returns the S3 object key.
func (b *S3Backend) Key() string {
	return b.key
}

// Endpoint returns the S3 endpoint.
func (b *S3Backend) Endpoint() string {
	return b.endpoint
}
