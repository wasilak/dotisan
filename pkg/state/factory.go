package state

import (
	"context"
	"fmt"

	"github.com/wasilak/dotisan/pkg/config"
)

// BackendType represents the type of state backend.
type BackendType string

const (
	BackendTypeLocal BackendType = "local"
	BackendTypeS3    BackendType = "s3"
)

// NewBackend creates a StateBackend based on the provided configuration.
// It returns either a LocalBackend or S3Backend depending on cfg.State.Backend.
func NewBackend(cfg *config.Config) (StateBackend, error) {
	switch BackendType(cfg.State.Backend) {
	case BackendTypeLocal, "": // Default to local
		path := cfg.State.Path
		if path == "" {
			var err error
			backend, err := NewLocalBackendWithDefaultPath()
			if err != nil {
				return nil, fmt.Errorf("failed to create local backend: %w", err)
			}
			return backend, nil
		}
		return NewLocalBackend(path), nil

	case BackendTypeS3:
		s3Config := S3Config{
			Endpoint:        cfg.State.S3.Endpoint,
			Bucket:          cfg.State.S3.Bucket,
			Key:             cfg.State.S3.Key,
			Region:          cfg.State.S3.Region,
			AccessKeyID:     cfg.State.S3.AccessKeyID,
			SecretAccessKey: cfg.State.S3.SecretAccessKey,
			UseSSL:          true, // Default to SSL
		}
		return NewS3Backend(s3Config)

	default:
		return nil, fmt.Errorf("unsupported state backend: %s", cfg.State.Backend)
	}
}

// BackendFromConfig is a convenience function that loads the dotisan config
// and creates the appropriate backend.
func BackendFromConfig() (StateBackend, error) {
	cfg, err := config.LoadConfigFromDefaultPath()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewBackend(cfg)
}

// BackendFromPath is a convenience function that creates a LocalBackend
// at the specified path.
func BackendFromPath(path string) StateBackend {
	return NewLocalBackend(path)
}

// BackendFromContext creates a backend using context-based configuration.
// This is useful when the backend needs to be reconfigured at runtime.
func BackendFromContext(ctx context.Context, cfg *config.Config) (StateBackend, error) {
	return NewBackend(cfg)
}
