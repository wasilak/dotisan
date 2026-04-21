package resource

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML dynamically unmarshals a YAML resource based on its kind.
// It first extracts the apiVersion and kind fields, then creates the appropriate
// resource type and unmarshals the full document into it.
func UnmarshalYAML(data []byte) (Resource, error) {
	// First, extract just the type information
	var typeInfo struct {
		APIVersion string `yaml:"apiVersion"`
		Kind       string `yaml:"kind"`
	}

	if err := yaml.Unmarshal(data, &typeInfo); err != nil {
		return nil, fmt.Errorf("failed to parse resource type info: %w", err)
	}

	// Validate API version
	if !IsSupportedAPIVersion(typeInfo.APIVersion) {
		return nil, fmt.Errorf("unsupported apiVersion: %s (supported: %s)", typeInfo.APIVersion, SupportedAPIVersion)
	}

	// Create the appropriate resource type based on kind
	var resource Resource
	switch typeInfo.Kind {
	case "BrewPackages":
		resource = &BrewPackages{}
	case "NpmPackages":
		resource = &NpmPackages{}
	case "GoPackages":
		resource = &GoPackages{}
	case "CargoPackages":
		resource = &CargoPackages{}
	case "ManagedFile":
		resource = &ManagedFile{}
	case "ManagedDirectory":
		resource = &ManagedDirectory{}
	default:
		return nil, fmt.Errorf("unknown resource kind: %s", typeInfo.Kind)
	}

	// Unmarshal the full document into the resource
	if err := yaml.Unmarshal(data, resource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s resource: %w", typeInfo.Kind, err)
	}

	return resource, nil
}

// ResourceKind represents all valid resource kinds.
type ResourceKind string

const (
	KindBrewPackages     ResourceKind = "BrewPackages"
	KindNpmPackages      ResourceKind = "NpmPackages"
	KindGoPackages       ResourceKind = "GoPackages"
	KindCargoPackages    ResourceKind = "CargoPackages"
	KindManagedFile      ResourceKind = "ManagedFile"
	KindManagedDirectory ResourceKind = "ManagedDirectory"
)

// ValidResourceKinds returns all valid resource kind strings.
func ValidResourceKinds() []string {
	return []string{
		"BrewPackages",
		"NpmPackages",
		"GoPackages",
		"CargoPackages",
		"ManagedFile",
		"ManagedDirectory",
	}
}
