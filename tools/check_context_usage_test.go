package tools

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoForbiddenContextUsage ensures library code does not use context.Background()
// or reference rootCtx. This prevents accidental loss of cancellation propagation.
func TestNoForbiddenContextUsage(t *testing.T) {
	repoRoot := "./"
	var offenders []string

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Only inspect .go files under pkg/ (library code)
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Skip tests
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if !strings.HasPrefix(path, "pkg/") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		s := string(data)
		if strings.Contains(s, "context.Background()") {
			offenders = append(offenders, path+": context.Background()")
		}
		if strings.Contains(s, "rootCtx") {
			offenders = append(offenders, path+": rootCtx")
		}
		if strings.Contains(s, "appctx.") {
			offenders = append(offenders, path+": appctx usage")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to scan repository: %v", err)
	}
	if len(offenders) > 0 {
		t.Errorf("forbidden context usage found:\n%s", strings.Join(offenders, "\n"))
	}
}
