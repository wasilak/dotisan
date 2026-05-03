package providers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wasilak/dotisan/pkg/cmdutil"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

func TestApply_Removal_NotInstalled_NoError(t *testing.T) {
	p := NewBrewProvider()

	orig := cmdutil.RunSimpleFn
	defer func() { cmdutil.RunSimpleFn = orig }()

	// Simulate uninstall returning an error but stderr indicates not installed
	cmdutil.RunSimpleFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
		if len(args) >= 1 && args[0] == "uninstall" {
			return "", "is not installed", errors.New("exit status 1")
		}
		return "", "", nil
	}

	plan := provider.GroupPlan{
		Removals: []provider.GroupRemoval{{
			Kind:  resource.KindHomeBrewPackages,
			Group: "core-tools",
			Items: []resource.ResourceItem{{Name: "ripgrep"}},
		}},
	}

	if err := p.Apply(context.Background(), plan); err != nil {
		t.Fatalf("Apply should not error when package not installed; err=%v", err)
	}
}

func TestApply_Removal_RefuseShowsDependents(t *testing.T) {
	p := NewBrewProvider()

	orig := cmdutil.RunSimpleFn
	defer func() { cmdutil.RunSimpleFn = orig }()

	cmdutil.RunSimpleFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
		// First uninstall call returns Refusing to uninstall error
		if len(args) >= 1 && args[0] == "uninstall" {
			return "", "Refusing to uninstall", errors.New("exit status 1")
		}
		// The provider will call `brew uses --installed <name>` to gather dependents
		if len(args) >= 2 && args[0] == "uses" {
			// Return a list of installed dependents
			return "dep1\ndep2", "", nil
		}
		return "", "", nil
	}

	plan := provider.GroupPlan{
		Removals: []provider.GroupRemoval{{
			Kind:  resource.KindHomeBrewPackages,
			Group: "core-tools",
			Items: []resource.ResourceItem{{Name: "ripgrep"}},
		}},
	}

	err := p.Apply(context.Background(), plan)
	if err == nil {
		t.Fatalf("Apply should return error when uninstall is refused")
	}
	// Error message should contain Installed dependents hint
	if !strings.Contains(err.Error(), "Installed dependents") && !strings.Contains(err.Error(), "Installed dependents") {
		t.Fatalf("Expected error message to include dependents hint; got: %v", err)
	}
}

func TestApply_Addition_InstallFails_ReturnsError(t *testing.T) {
	p := NewBrewProvider()

	orig := cmdutil.RunSimpleFn
	defer func() { cmdutil.RunSimpleFn = orig }()

	cmdutil.RunSimpleFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
		if len(args) >= 1 && args[0] == "install" {
			return "", "permission denied", errors.New("exit status 1")
		}
		return "", "", nil
	}

	plan := provider.GroupPlan{
		Additions: []provider.GroupAddition{{
			Kind:  resource.KindHomeBrewPackages,
			Group: "core-tools",
			Items: []resource.ResourceItem{{Name: "ripgrep"}},
		}},
	}

	if err := p.Apply(context.Background(), plan); err == nil {
		t.Fatalf("Apply should return error when install fails")
	}
}

func TestApply_Untap_NoSuchTap_NoError(t *testing.T) {
	p := NewBrewProvider()

	orig := cmdutil.RunSimpleFn
	defer func() { cmdutil.RunSimpleFn = orig }()

	cmdutil.RunSimpleFn = func(ctx context.Context, name string, args ...string) (string, string, error) {
		if len(args) >= 1 && args[0] == "untap" {
			return "", "No such tap", errors.New("exit status 1")
		}
		return "", "", nil
	}

	plan := provider.GroupPlan{
		Removals: []provider.GroupRemoval{{
			Kind:  resource.KindHomeBrewTaps,
			Group: "my-taps",
			Items: []resource.ResourceItem{{Name: "homebrew/cask-fonts"}},
		}},
	}

	if err := p.Apply(context.Background(), plan); err != nil {
		t.Fatalf("Apply should not error when untap reports No such tap; err=%v", err)
	}
}
