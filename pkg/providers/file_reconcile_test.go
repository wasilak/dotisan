package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

func TestFileProvider_Reconcile_Addition(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	
	// Create resources subdirectory (files are resolved relative to resources/)
	resourcesDir := filepath.Join(dotfilesDir, "resources")
	os.MkdirAll(resourcesDir, 0755)
	
	// Create a source file in resources/
	sourceFile := filepath.Join(resourcesDir, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create provider
	ctx := config.NewTemplateContext()
	engine := diff.NewEngine()
	p := NewFileProvider(ctx, engine, dotfilesDir)

	// Create desired resource
	mf := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   resource.Metadata{Name: "test", Namespace: "default"},
		},
		Spec: resource.ManagedFileSpec{
			SourceFile:  "test.txt",
			Destination: filepath.Join(t.TempDir(), "dest.txt"), // Non-existent destination
			Template:    false,
		},
	}

	// Reconcile
	desired := []resource.Resource{mf}
	state := []provider.ResourceState{}
	plan := p.Reconcile(desired, state)

	// Should have one addition
	if len(plan.Additions) != 1 {
		t.Errorf("len(Additions) = %d, want 1", len(plan.Additions))
	}
}

func TestFileProvider_Reconcile_InSync(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()
	
	// Create resources subdirectory (files are resolved relative to resources/)
	resourcesDir := filepath.Join(dotfilesDir, "resources")
	os.MkdirAll(resourcesDir, 0755)
	
	// Create a source file in resources/
	sourceFile := filepath.Join(resourcesDir, "test.txt")
	content := "test content"
	if err := os.WriteFile(sourceFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination file with same content
	destFile := filepath.Join(destDir, "dest.txt")
	if err := os.WriteFile(destFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create dest file: %v", err)
	}

	// Create provider
	ctx := config.NewTemplateContext()
	engine := diff.NewEngine()
	p := NewFileProvider(ctx, engine, dotfilesDir)

	// Create desired resource
	mf := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   resource.Metadata{Name: "test", Namespace: "default"},
		},
		Spec: resource.ManagedFileSpec{
			SourceFile:  "test.txt",
			Destination: destFile,
			Template:    false,
		},
	}

	// Calculate checksum for state
	checksum := calculateChecksum(content)
	
	// Reconcile with matching state
	desired := []resource.Resource{mf}
	state := []provider.ResourceState{
		{
			ID:       "ManagedFile/test",
			Kind:     "ManagedFile",
			Name:     "test",
			DestHash: checksum,
		},
	}
	plan := p.Reconcile(desired, state)

	// Should be in sync
	if len(plan.InSync) != 1 {
		t.Errorf("len(InSync) = %d, want 1", len(plan.InSync))
	}
}

func TestFileProvider_Reconcile_Modification(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()
	
	// Create resources subdirectory (files are resolved relative to resources/)
	resourcesDir := filepath.Join(dotfilesDir, "resources")
	os.MkdirAll(resourcesDir, 0755)
	
	// Create a source file in resources/
	sourceFile := filepath.Join(resourcesDir, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination file with different content
	destFile := filepath.Join(destDir, "dest.txt")
	if err := os.WriteFile(destFile, []byte("old content"), 0644); err != nil {
		t.Fatalf("Failed to create dest file: %v", err)
	}

	// Create provider
	ctx := config.NewTemplateContext()
	engine := diff.NewEngine()
	p := NewFileProvider(ctx, engine, dotfilesDir)

	// Create desired resource
	mf := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   resource.Metadata{Name: "test", Namespace: "default"},
		},
		Spec: resource.ManagedFileSpec{
			SourceFile:  "test.txt",
			Destination: destFile,
			Template:    false,
		},
	}

	// Reconcile with old state
	desired := []resource.Resource{mf}
	state := []provider.ResourceState{
		{
			ID:       "ManagedFile/test",
			Kind:     "ManagedFile",
			Name:     "test",
			DestHash: calculateChecksum("old content"), // Old checksum
		},
	}
	plan := p.Reconcile(desired, state)

	// Should have one modification
	if len(plan.Modifications) != 1 {
		t.Errorf("len(Modifications) = %d, want 1", len(plan.Modifications))
	}
}

func TestFileProvider_resolveDestination(t *testing.T) {
	tests := []struct {
		name       string
		dest       string
		values     map[string]interface{}
		wantSuffix string
		wantErr    bool
	}{
		{
			name:       "plain path",
			dest:       "/tmp/test.txt",
			wantSuffix: "/tmp/test.txt",
			wantErr:    false,
		},
		{
			name:       "path with tilde",
			dest:       "~/test.txt",
			wantSuffix: "test.txt",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFileProvider(nil, nil, "")
			resolved, err := p.resolveDestination(tt.dest)

			if (err != nil) != tt.wantErr {
				t.Errorf("resolveDestination() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !containsStr(resolved, tt.wantSuffix) {
				t.Errorf("resolveDestination() = %q, should contain %q", resolved, tt.wantSuffix)
			}
		})
	}
}

func TestFileProvider_renderSource(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "template.txt")
	content := "Hello {{ .Env.USER }}!"
	if err := os.WriteFile(sourceFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	tests := []struct {
		name       string
		isTemplate bool
		wantTmpl   bool
	}{
		{
			name:       "static file",
			isTemplate: false,
			wantTmpl:   false,
		},
		{
			name:       "template file",
			isTemplate: true,
			wantTmpl:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := config.NewTemplateContext()
			p := NewFileProvider(ctx, nil, "")
			
			rendered, err := p.renderSource(sourceFile, tt.isTemplate)
			if err != nil {
				t.Errorf("renderSource() error = %v", err)
				return
			}

			if tt.isTemplate {
				// Should have {{ .Env.USER }} replaced
				if containsStr(rendered, "{{") {
					t.Error("template should have been rendered")
				}
			} else {
				// Should be unchanged
				if rendered != content {
					t.Errorf("static content changed: got %q, want %q", rendered, content)
				}
			}
		})
	}
}

func TestCalculateChecksum(t *testing.T) {
	content := "test content"
	checksum1 := calculateChecksum(content)
	checksum2 := calculateChecksum(content)

	if checksum1 != checksum2 {
		t.Error("same content should produce same checksum")
	}

	// Different content should produce different checksum
	checksum3 := calculateChecksum("different content")
	if checksum1 == checksum3 {
		t.Error("different content should produce different checksum")
	}

	// Should be 64 characters (SHA256 hex)
	if len(checksum1) != 64 {
		t.Errorf("checksum length = %d, want 64", len(checksum1))
	}
}

func TestCalculateChecksumFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")
	content := "test content"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	checksum := calculateChecksumFromFile(file)
	expected := calculateChecksum(content)

	if checksum != expected {
		t.Errorf("checksum = %q, want %q", checksum, expected)
	}
}

func TestCalculateChecksumFromFile_NonExistent(t *testing.T) {
	checksum := calculateChecksumFromFile("/nonexistent/file.txt")
	if checksum != "" {
		t.Error("checksum should be empty for non-existent file")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
