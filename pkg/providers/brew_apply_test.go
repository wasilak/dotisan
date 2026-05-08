package providers

import (
	"context"
	"strings"
	"testing"

	"github.com/wasilak/nim/pkg/cmdutil"
	"github.com/wasilak/nim/pkg/provider"
	"github.com/wasilak/nim/pkg/resource"
)

func TestApply_Additions_InvokesBrewCommands(t *testing.T) {
	p := NewBrewProvider()

	// Capture commands executed
	var calls []string
	orig := cmdutil.RunSimpleFn
	defer func() { cmdutil.RunSimpleFn = orig }()
	cmdutil.RunSimpleFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return "", "", nil
	}

	plan := provider.GroupPlan{
		Additions: []provider.GroupAddition{
			{
				Kind:  resource.KindHomeBrewTaps,
				Group: "my-taps",
				Items: []resource.ResourceItem{{Name: "homebrew/cask-fonts"}},
			},
			{
				Kind:  resource.KindHomeBrewPackages,
				Group: "core-tools",
				Items: []resource.ResourceItem{{Name: "ripgrep"}},
			},
			{
				Kind:  resource.KindHomeBrewCasks,
				Group: "apps",
				Items: []resource.ResourceItem{{Name: "wezterm"}},
			},
		},
	}

	if err := p.Apply(context.Background(), plan); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Expect tap, install, and cask install commands
	foundTap := false
	foundInstall := false
	foundCask := false
	for _, c := range calls {
		if strings.HasPrefix(c, "brew tap ") {
			foundTap = true
		}
		if strings.HasPrefix(c, "brew install ripgrep") {
			foundInstall = true
		}
		if strings.HasPrefix(c, "brew install --cask wezterm") {
			foundCask = true
		}
	}
	if !foundTap || !foundInstall || !foundCask {
		t.Fatalf("Expected brew tap/install/cask calls; got: %v", calls)
	}
}

func TestApply_Removals_InvokesBrewCommands(t *testing.T) {
	p := NewBrewProvider()

	var calls []string
	orig := cmdutil.RunSimpleFn
	defer func() { cmdutil.RunSimpleFn = orig }()
	cmdutil.RunSimpleFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return "", "", nil
	}

	plan := provider.GroupPlan{
		Removals: []provider.GroupRemoval{
			{
				Kind:  resource.KindHomeBrewTaps,
				Group: "my-taps",
				Items: []resource.ResourceItem{{Name: "homebrew/cask-fonts"}},
			},
			{
				Kind:  resource.KindHomeBrewPackages,
				Group: "core-tools",
				Items: []resource.ResourceItem{{Name: "ripgrep"}},
			},
			{
				Kind:  resource.KindHomeBrewCasks,
				Group: "apps",
				Items: []resource.ResourceItem{{Name: "wezterm"}},
			},
		},
	}

	if err := p.Apply(context.Background(), plan); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Expect untap, uninstall, and cask uninstall commands
	foundUntap := false
	foundUninstall := false
	foundCaskUninstall := false
	for _, c := range calls {
		if strings.HasPrefix(c, "brew untap ") {
			foundUntap = true
		}
		if strings.HasPrefix(c, "brew uninstall ripgrep") {
			foundUninstall = true
		}
		if strings.HasPrefix(c, "brew uninstall --cask wezterm") {
			foundCaskUninstall = true
		}
	}
	if !foundUntap || !foundUninstall || !foundCaskUninstall {
		t.Fatalf("Expected brew untap/uninstall/cask uninstall calls; got: %v", calls)
	}
}
