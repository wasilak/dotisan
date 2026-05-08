package engine

import (
	"testing"

	"github.com/wasilak/nim/pkg/resource"
)

func TestParseTargets(t *testing.T) {
	inputs := []string{resource.KindHomeBrewPackages, resource.KindHomeBrewPackages + "/core-tools", resource.KindHomeBrewPackages + "/core-tools/ripgrep"}
	parsed := ParseTargets(inputs)
	if len(parsed) != 3 {
		t.Fatalf("expected 3 parsed targets, got %d", len(parsed))
	}
	if parsed[0].Kind != resource.KindHomeBrewPackages {
		t.Fatalf("expected kind %s, got %s", resource.KindHomeBrewPackages, parsed[0].Kind)
	}
	if parsed[1].Group != "core-tools" {
		t.Fatalf("expected group core-tools, got %s", parsed[1].Group)
	}
	if parsed[2].Item != "ripgrep" {
		t.Fatalf("expected item ripgrep, got %s", parsed[2].Item)
	}
}

func TestParseTargets_BracketedFormat(t *testing.T) {
	inputs := []string{resource.KindHomeBrewPackages, resource.KindHomeBrewPackages + "/homebrew-packages", resource.KindHomeBrewPackages + "/homebrew-packages[eza]"}
	parsed := ParseTargets(inputs)
	if len(parsed) != 3 {
		t.Fatalf("expected 3 parsed targets, got %d", len(parsed))
	}
	if parsed[0].Kind != resource.KindHomeBrewPackages || parsed[0].Group != "" || parsed[0].Item != "" {
		t.Fatalf("expected kind only")
	}
	if parsed[1].Kind != resource.KindHomeBrewPackages || parsed[1].Group != "homebrew-packages" || parsed[1].Item != "" {
		t.Fatalf("expected kind and group")
	}
	if parsed[2].Kind != resource.KindHomeBrewPackages || parsed[2].Group != "homebrew-packages" || parsed[2].Item != "eza" {
		t.Fatalf("expected kind, group, item from bracketed format")
	}
}

func TestTargetMatch_Matches(t *testing.T) {
	tm := TargetMatch{Kind: resource.KindHomeBrewPackages, Group: "core-tools", Item: "ripgrep"}
	if !tm.Matches(resource.KindHomeBrewPackages, "core-tools", "ripgrep") {
		t.Fatalf("expected match to be true")
	}
	if tm.Matches(resource.KindHomeBrewPackages, "other", "ripgrep") {
		t.Fatalf("expected group mismatch to be false")
	}
}
