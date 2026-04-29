package engine

import (
    "testing"
)

func TestParseTargets(t *testing.T) {
    inputs := []string{"BrewPackages", "BrewPackages/core-tools", "BrewPackages/core-tools/ripgrep"}
    parsed := ParseTargets(inputs)
    if len(parsed) != 3 {
        t.Fatalf("expected 3 parsed targets, got %d", len(parsed))
    }
    if parsed[0].Kind != "BrewPackages" {
        t.Fatalf("expected kind BrewPackages, got %s", parsed[0].Kind)
    }
    if parsed[1].Group != "core-tools" {
        t.Fatalf("expected group core-tools, got %s", parsed[1].Group)
    }
    if parsed[2].Item != "ripgrep" {
        t.Fatalf("expected item ripgrep, got %s", parsed[2].Item)
    }
}

func TestTargetMatch_Matches(t *testing.T) {
    tm := TargetMatch{Kind: "BrewPackages", Group: "core-tools", Item: "ripgrep"}
    if !tm.Matches("BrewPackages", "core-tools", "ripgrep") {
        t.Fatalf("expected match to be true")
    }
    if tm.Matches("BrewPackages", "other", "ripgrep") {
        t.Fatalf("expected group mismatch to be false")
    }
}
