// Package config provides configuration loading and management for dotisan.
// It handles loading of ~/.dotisan/config.yaml which contains tool-level configuration
// such as state backend settings and dotfiles root path.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// S3Config holds S3-compatible backend configuration.
type S3Config struct {
	Endpoint        string `yaml:"endpoint"`
	Bucket          string `yaml:"bucket"`
	Key             string `yaml:"key"`
	Region          string `yaml:"region"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
}

// StateConfig holds state backend configuration.
type StateConfig struct {
	Backend string   `yaml:"backend"`        // "local" or "s3"
	Path    string   `yaml:"path,omitempty"` // for local backend
	S3      S3Config `yaml:"s3,omitempty"`   // for s3 backend
}

// UIConfig holds UI/display configuration.
type UIConfig struct {
	// Output determines the default output format (plain, tree, json)
	Output string `yaml:"output,omitempty"`
}

// Config holds the complete dotisan tool configuration from ~/.dotisan/config.yaml
type Config struct {
	// DotfilesRoot is the path to the dotfiles directory (default: ~/.config/dotisan)
	DotfilesRoot string `yaml:"dotfiles_root"`

	// State holds state backend configuration
	State StateConfig `yaml:"state"`

	// UI holds UI/display configuration
	UI UIConfig `yaml:"ui,omitempty"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		DotfilesRoot: filepath.Join(homeDir, ".config/dotisan"),
		State: StateConfig{
			Backend: "local",
			Path:    filepath.Join(homeDir, ".config", "dotisan", "state.json"),
		},
	}
}

// LoadConfig loads the dotisan configuration from the specified path.
// If the file doesn't exist, it returns a default configuration.
func LoadConfig(path string) (*Config, error) {
	// Start with defaults
	cfg := DefaultConfig()

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No config file, return defaults
		return cfg, nil
	}

	// Read config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Expand environment variables in paths
	cfg.DotfilesRoot = os.ExpandEnv(cfg.DotfilesRoot)
	cfg.State.Path = os.ExpandEnv(cfg.State.Path)

	return cfg, nil
}

// LoadConfigFromDefaultPath loads configuration from the default location (~/.dotisan/config.yaml).
func LoadConfigFromDefaultPath() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".dotisan", "config.yaml")
	return LoadConfig(configPath)
}

// LoadValues loads values from a YAML file (like ~/.config/dotisan/values.yaml).
// It returns the parsed values as a map[string]interface{}.
func LoadValues(path string) (map[string]interface{}, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// No values file, return empty map
		return make(map[string]interface{}), nil
	}

	// Read values file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read values file %s: %w", path, err)
	}

	// Parse YAML
	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("failed to parse values file %s: %w", path, err)
	}

	return values, nil
}

// ParseResources parses YAML content containing multiple resource definitions.
// It returns a slice of resource objects (as map[string]interface{}).
func ParseResources(data []byte) ([]interface{}, error) {
	// Split by document separator
	docs := splitYAMLDocuments(string(data))

	var resources []interface{}
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var resource map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &resource); err != nil {
			return nil, fmt.Errorf("failed to parse resource: %w", err)
		}

		if len(resource) > 0 {
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// MarshalResource serializes a resource object to YAML.
func MarshalResource(resource interface{}) ([]byte, error) {
	return yaml.Marshal(resource)
}

// splitYAMLDocuments splits a YAML file containing multiple documents (--- separated).
func splitYAMLDocuments(content string) []string {
	var docs []string
	parts := strings.Split(content, "---")
	for _, part := range parts {
		docs = append(docs, part)
	}
	return docs
}
