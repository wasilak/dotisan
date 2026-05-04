package providers

import (
	"encoding/json"
	"testing"
)

func TestBrewInfoParsingFormulae(t *testing.T) {
	sample := `{"formulae":[{"name":"foo","versions":{"stable":"1.2.3"},"installed":[{"version":"1.2.3"}]},{"name":"bar","versions":{"stable":"2.0.0"},"installed":[]}],"casks":[]}`
	var out brewInfoOutput
	if err := json.Unmarshal([]byte(sample), &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(out.Formulae) != 2 {
		t.Fatalf("expected 2 formulae, got %d", len(out.Formulae))
	}
	if out.Formulae[0].InstalledVersion() != "1.2.3" {
		t.Fatalf("expected foo version 1.2.3, got %q", out.Formulae[0].InstalledVersion())
	}
	if out.Formulae[1].InstalledVersion() != "2.0.0" {
		t.Fatalf("expected bar stable version 2.0.0, got %q", out.Formulae[1].InstalledVersion())
	}
}

func TestBrewInfoParsingCasks(t *testing.T) {
	// Real brew info --json=v2: cask name is []string, installed is a version string or null.
	sample := `{"formulae":[],"casks":[{"token":"baz-token","name":["Baz App"],"installed":"3.4.5"},{"token":"no-name-token","name":[],"installed":null}]}`
	var out brewInfoOutput
	if err := json.Unmarshal([]byte(sample), &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(out.Casks) != 2 {
		t.Fatalf("expected 2 casks, got %d", len(out.Casks))
	}
	if out.Casks[0].InstalledVersion() != "3.4.5" {
		t.Fatalf("expected Baz App version 3.4.5, got %q", out.Casks[0].InstalledVersion())
	}
	if out.Casks[0].DisplayName() != "Baz App" {
		t.Fatalf("expected Baz App display name, got %q", out.Casks[0].DisplayName())
	}
	if out.Casks[0].Token != "baz-token" {
		t.Fatalf("expected baz-token token, got %q", out.Casks[0].Token)
	}
	if out.Casks[1].DisplayName() != "no-name-token" {
		t.Fatalf("expected token fallback display name, got %q", out.Casks[1].DisplayName())
	}
}
