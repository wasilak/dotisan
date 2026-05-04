package resource

// Package represents a generic named package with optional version.
type Package struct {
	Name      string   `yaml:"name"`
	Version   string   `yaml:"version,omitempty"`
	DependsOn []string `yaml:"dependsOn,omitempty"`
}

// Tap represents a Homebrew tap entry.
type Tap struct {
	Name      string   `yaml:"name"`
	Version   string   `yaml:"version,omitempty"`
	DependsOn []string `yaml:"dependsOn,omitempty"`
}
