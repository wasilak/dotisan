package config

import (
	"os"
	"runtime"
	"testing"
)

func TestNewTemplateContext(t *testing.T) {
	ctx := NewTemplateContext()

	if ctx == nil {
		t.Fatal("NewTemplateContext() returned nil")
	}

	// Check Values is initialized (empty)
	if ctx.Values == nil {
		t.Error("NewTemplateContext().Values is nil")
	}

	// Check Env is populated
	if ctx.Env == nil {
		t.Fatal("NewTemplateContext().Env is nil")
	}

	// Verify common environment variables are present
	if _, ok := ctx.Env["PATH"]; !ok {
		t.Error("PATH not found in Env")
	}

	if _, ok := ctx.Env["HOME"]; !ok {
		t.Error("HOME not found in Env")
	}

	// Check OS info
	if ctx.OS.GOOS != runtime.GOOS {
		t.Errorf("OS.GOOS = %q, want %q", ctx.OS.GOOS, runtime.GOOS)
	}

	if ctx.OS.GOARCH != runtime.GOARCH {
		t.Errorf("OS.GOARCH = %q, want %q", ctx.OS.GOARCH, runtime.GOARCH)
	}

	// Hostname may be empty in some environments, so we just check it exists
	// (it can be empty string if os.Hostname() fails)
}

func TestLoadEnv(t *testing.T) {
	// Set a test environment variable
	os.Setenv("DOTISAN_TEST_VAR", "test_value")
	defer os.Unsetenv("DOTISAN_TEST_VAR")

	env := loadEnv()

	if env == nil {
		t.Fatal("loadEnv() returned nil")
	}

	if env["DOTISAN_TEST_VAR"] != "test_value" {
		t.Errorf("env[\"DOTISAN_TEST_VAR\"] = %q, want %q", env["DOTISAN_TEST_VAR"], "test_value")
	}
}

func TestLoadEnv_EmptyValue(t *testing.T) {
	// Test variable with empty value
	os.Setenv("DOTISAN_EMPTY_VAR", "")
	defer os.Unsetenv("DOTISAN_EMPTY_VAR")

	env := loadEnv()

	// Empty value should still be present in map
	if _, ok := env["DOTISAN_EMPTY_VAR"]; !ok {
		t.Error("Empty value variable should still be in env map")
	}
}

func TestLoadEnv_EqualsInValue(t *testing.T) {
	// Test variable with = in value
	os.Setenv("DOTISAN_EQUALS_VAR", "value=with=equals")
	defer os.Unsetenv("DOTISAN_EQUALS_VAR")

	env := loadEnv()

	if env["DOTISAN_EQUALS_VAR"] != "value=with=equals" {
		t.Errorf("env[\"DOTISAN_EQUALS_VAR\"] = %q, want %q", env["DOTISAN_EQUALS_VAR"], "value=with=equals")
	}
}

func TestLoadOSInfo(t *testing.T) {
	osInfo := loadOSInfo()

	if osInfo.GOOS != runtime.GOOS {
		t.Errorf("GOOS = %q, want %q", osInfo.GOOS, runtime.GOOS)
	}

	if osInfo.GOARCH != runtime.GOARCH {
		t.Errorf("GOARCH = %q, want %q", osInfo.GOARCH, runtime.GOARCH)
	}

	// Hostname may be empty but should not panic
	// Just verifying the call doesn't error
}
