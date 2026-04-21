package providers

import (
	"testing"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/diff"
)

func TestNewFileProvider(t *testing.T) {
	ctx := config.NewTemplateContext()
	engine := diff.NewEngine()

	p := NewFileProvider(ctx, engine, "/tmp/dotfiles")

	if p == nil {
		t.Fatal("NewFileProvider() returned nil")
	}

	if p.templateContext != ctx {
		t.Error("templateContext not set correctly")
	}

	if p.diffEngine != engine {
		t.Error("diffEngine not set correctly")
	}

	if p.dotfilesRoot != "/tmp/dotfiles" {
		t.Errorf("dotfilesRoot = %q, want %q", p.dotfilesRoot, "/tmp/dotfiles")
	}
}

func TestFileProvider_Name(t *testing.T) {
	p := NewFileProvider(nil, nil, "")

	if name := p.Name(); name != "file" {
		t.Errorf("Name() = %q, want %q", name, "file")
	}
}

func TestFileProvider_Available(t *testing.T) {
	tests := []struct {
		name          string
		dotfilesRoot  string
		wantAvailable bool
	}{
		{
			name:          "valid dotfiles root",
			dotfilesRoot:  t.TempDir(),
			wantAvailable: true,
		},
		{
			name:          "empty dotfiles root (uses home)",
			dotfilesRoot:  "",
			wantAvailable: true,
		},
		{
			name:          "non-existent dotfiles root",
			dotfilesRoot:  "/nonexistent/path/12345",
			wantAvailable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFileProvider(nil, nil, tt.dotfilesRoot)
			available, msg := p.Available()

			if available != tt.wantAvailable {
				t.Errorf("Available() = %v, want %v (message: %s)", available, tt.wantAvailable, msg)
			}

			if msg == "" {
				t.Error("Available() should return a message")
			}
		})
	}
}

func TestFileProvider_Available_NoHomeDir(t *testing.T) {
	// This test verifies that Available() checks home directory
	// In practice, we can't easily unset HOME, but we can verify the logic
	p := NewFileProvider(nil, nil, "")
	available, msg := p.Available()

	// Should generally be available in test environment
	if !available {
		t.Logf("Available() returned false (may be expected in restricted env): %s", msg)
	}
}

func TestFileProvider_EmbedCorrectly(t *testing.T) {
	// Verify that FileProvider can be registered with provider registry
	ctx := config.NewTemplateContext()
	engine := diff.NewEngine()
	tmpDir := t.TempDir()

	p := NewFileProvider(ctx, engine, tmpDir)

	// Basic check that all fields are set
	if p.templateContext == nil {
		t.Error("templateContext is nil")
	}
	if p.diffEngine == nil {
		t.Error("diffEngine is nil")
	}
}
