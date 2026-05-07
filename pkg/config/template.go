// Package config provides configuration loading and management for dotisan.
//
// This file contains the TemplateContext for Go template rendering,
// supporting two-pass templating where values.yaml is rendered first
// using Env and OS context, then parsed into .Values for subsequent renders.
package config

import (
	"os"
	"runtime"
)

// OSInfo holds operating system information available in templates.
type OSInfo struct {
	GOOS     string `yaml:"goos"`     // runtime.GOOS (darwin, linux, etc.)
	GOARCH   string `yaml:"goarch"`   // runtime.GOARCH (amd64, arm64, etc.)
	Hostname string `yaml:"hostname"` // Machine hostname
}

// TemplateContext provides the data available during Go template rendering.
// It supports Helm-style templating with values, environment variables, and OS info.
type TemplateContext struct {
	// Values holds the parsed values.yaml content (populated after first-pass render).
	// map[string]any is intentional: values.yaml is user-defined and schema-less at compile time.
	Values map[string]any `yaml:"values"`

	// Env holds environment variables from os.Environ()
	Env map[string]string `yaml:"env"`

	// OS holds operating system information
	OS OSInfo `yaml:"os"`
}

// NewTemplateContext creates a new TemplateContext with Env and OS pre-populated.
// This is used for the first-pass render of values.yaml.
func NewTemplateContext() *TemplateContext {
	return &TemplateContext{
		Values: make(map[string]any),
		Env:    loadEnv(),
		OS:     loadOSInfo(),
	}
}

// loadEnv converts os.Environ() into a map.
func loadEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		// Split on first '=' only
		for i := 0; i < len(e); i++ {
			if e[i] == '=' {
				key := e[:i]
				value := e[i+1:]
				env[key] = value
				break
			}
		}
	}
	return env
}

// loadOSInfo gathers OS information.
func loadOSInfo() OSInfo {
	hostname, _ := os.Hostname()
	return OSInfo{
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
		Hostname: hostname,
	}
}
