// Package output defines output format types and utilities for dotisan commands.
package output

// Format represents the output format type
type Format string

const (
	// FormatPlain is the default table/text output
	FormatPlain Format = "plain"
	// FormatTree renders output as a 3-level tree
	FormatTree Format = "tree"
	// FormatJSON outputs clean JSON without extra human-readable text
	FormatJSON Format = "json"
)

// IsValid checks if the format is valid
func (f Format) IsValid() bool {
	switch f {
	case FormatPlain, FormatTree, FormatJSON:
		return true
	}
	return false
}

// String returns the string representation
func (f Format) String() string {
	return string(f)
}
