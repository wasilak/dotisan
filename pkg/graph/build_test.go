package graph

import (
	"strings"
	"testing"

	"github.com/wasilak/dotisan/pkg/resource"
)

// makeHB is a helper that constructs a minimal HomeBrewPackages resource.
func makeHB(name, namespace string, dependsOn ...string) *resource.HomeBrewPackages {
	return &resource.HomeBrewPackages{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewPackages",
			Metadata: resource.Metadata{
				Name:      name,
				Namespace: namespace,
				DependsOn: dependsOn,
			},
		},
	}
}

func TestBuild_NoDeps(t *testing.T) {
	resources := []resource.Resource{
		makeHB("tools", "default"),
		makeHB("fonts", "default"),
	}
	d, err := Build(resources, "default")
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if len(d.Nodes()) != 2 {
		t.Errorf("node count = %d, want 2", len(d.Nodes()))
	}
}

func TestBuild_WithDeps_TopologicalOrder(t *testing.T) {
	// fonts depends on tools
	resources := []resource.Resource{
		makeHB("tools", "default"),
		makeHB("fonts", "default", "HomeBrewPackages/tools"),
	}
	d, err := Build(resources, "default")
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder() error: %v", err)
	}
	if len(order) != 2 {
		t.Fatalf("order len = %d, want 2", len(order))
	}

	pos := make(map[NodeID]int, 2)
	for i, id := range order {
		pos[id] = i
	}
	toolsID := ResourceNodeID("default", "HomeBrewPackages", "tools")
	fontsID := ResourceNodeID("default", "HomeBrewPackages", "fonts")
	if pos[toolsID] >= pos[fontsID] {
		t.Errorf("tools must come before fonts: tools=%d fonts=%d", pos[toolsID], pos[fontsID])
	}
}

func TestBuild_CycleReturnsError(t *testing.T) {
	// A depends on B, B depends on A
	resources := []resource.Resource{
		makeHB("a", "default", "HomeBrewPackages/b"),
		makeHB("b", "default", "HomeBrewPackages/a"),
	}
	d, err := Build(resources, "default")
	if err != nil {
		t.Fatalf("Build() should not fail — cycle is detected at Validate: %v", err)
	}
	_, err = d.TopologicalOrder()
	if err == nil {
		t.Fatal("expected cycle error from TopologicalOrder, got nil")
	}
	if !strings.Contains(err.Error(), "→") {
		t.Errorf("cycle error should show path, got: %q", err.Error())
	}
}

func TestBuild_InvalidAddress(t *testing.T) {
	// "[bad]" is an invalid resource ID
	resources := []resource.Resource{
		makeHB("a", "default", "[bad]"),
	}
	if _, err := Build(resources, "default"); err == nil {
		t.Fatal("expected error for invalid dependency address")
	}
}

func TestBuild_Empty(t *testing.T) {
	d, err := Build(nil, "default")
	if err != nil {
		t.Fatalf("Build(nil) error: %v", err)
	}
	if len(d.Nodes()) != 0 {
		t.Errorf("expected empty DAG, got %d nodes", len(d.Nodes()))
	}
}

func TestBuild_ImplicitTapDeps_PackagesDependOnTaps(t *testing.T) {
	taps := &resource.HomeBrewTaps{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewTaps",
			Metadata:   resource.Metadata{Name: "extra-taps", Namespace: "default"},
		},
		Spec: resource.HomeBrewTapsSpec{},
	}
	packages := makeHB("tools", "default") // no explicit dependsOn

	d, err := Build([]resource.Resource{taps, packages}, "default")
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder() error: %v", err)
	}

	pos := make(map[NodeID]int, 2)
	for i, id := range order {
		pos[id] = i
	}
	tapID := ResourceNodeID("default", "HomeBrewTaps", "extra-taps")
	pkgID := ResourceNodeID("default", "HomeBrewPackages", "tools")
	if pos[tapID] >= pos[pkgID] {
		t.Errorf("taps must be applied before packages: tap=%d pkg=%d", pos[tapID], pos[pkgID])
	}
}

func TestBuild_ImplicitTapDeps_CasksAlsoDependOnTaps(t *testing.T) {
	taps := &resource.HomeBrewTaps{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewTaps",
			Metadata:   resource.Metadata{Name: "cask-fonts", Namespace: "default"},
		},
		Spec: resource.HomeBrewTapsSpec{},
	}
	casks := &resource.HomeBrewCasks{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewCasks",
			Metadata:   resource.Metadata{Name: "fonts", Namespace: "default"},
		},
		Spec: resource.HomeBrewCasksSpec{},
	}

	d, err := Build([]resource.Resource{taps, casks}, "default")
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder() error: %v", err)
	}

	pos := make(map[NodeID]int, 2)
	for i, id := range order {
		pos[id] = i
	}
	tapID := ResourceNodeID("default", "HomeBrewTaps", "cask-fonts")
	caskID := ResourceNodeID("default", "HomeBrewCasks", "fonts")
	if pos[tapID] >= pos[caskID] {
		t.Errorf("taps must be applied before casks: tap=%d cask=%d", pos[tapID], pos[caskID])
	}
}

func TestBuild_ImplicitTapDeps_NoDuplicateWhenExplicit(t *testing.T) {
	taps := &resource.HomeBrewTaps{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewTaps",
			Metadata:   resource.Metadata{Name: "extra-taps", Namespace: "default"},
		},
		Spec: resource.HomeBrewTapsSpec{},
	}
	// packages already declares the tap dep explicitly
	packages := makeHB("tools", "default", "HomeBrewTaps/extra-taps")

	d, err := Build([]resource.Resource{taps, packages}, "default")
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	tapID := ResourceNodeID("default", "HomeBrewTaps", "extra-taps")
	pkgID := ResourceNodeID("default", "HomeBrewPackages", "tools")
	deps := d.DependenciesOf(pkgID)
	count := 0
	for _, dep := range deps {
		if dep == tapID {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected tap dep exactly once, found %d times in %v", count, deps)
	}
}

func TestBuild_ImplicitTapDeps_CrossNamespaceIsolation(t *testing.T) {
	// Taps in "other" namespace should NOT be injected into packages in "default"
	tapsOther := &resource.HomeBrewTaps{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewTaps",
			Metadata:   resource.Metadata{Name: "other-taps", Namespace: "other"},
		},
		Spec: resource.HomeBrewTapsSpec{},
	}
	packages := makeHB("tools", "default") // default namespace

	d, err := Build([]resource.Resource{tapsOther, packages}, "default")
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	pkgID := ResourceNodeID("default", "HomeBrewPackages", "tools")
	deps := d.DependenciesOf(pkgID)
	if len(deps) != 0 {
		t.Errorf("packages in 'default' should have no implicit tap deps from 'other' ns, got %v", deps)
	}
}

func TestBuild_CrossNamespaceDependency(t *testing.T) {
	// "apps" (ns: apps) depends on "tools" (ns: default) using an explicit namespace prefix
	tools := makeHB("tools", "default")
	apps := makeHB("apps", "apps", "default/HomeBrewPackages/tools")

	d, err := Build([]resource.Resource{tools, apps}, "default")
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder() error: %v", err)
	}
	if len(order) != 2 {
		t.Fatalf("order len = %d, want 2", len(order))
	}

	pos := make(map[NodeID]int, 2)
	for i, id := range order {
		pos[id] = i
	}
	toolsID := ResourceNodeID("default", "HomeBrewPackages", "tools")
	appsID := ResourceNodeID("apps", "HomeBrewPackages", "apps")
	if pos[toolsID] >= pos[appsID] {
		t.Errorf("tools (default ns) must come before apps (apps ns): tools=%d apps=%d", pos[toolsID], pos[appsID])
	}
}

func TestBuild_MultipleKinds(t *testing.T) {
	// ManagedFile depends on HomeBrewPackages
	brew := makeHB("tools", "default")
	file := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata: resource.Metadata{
				Name:      "dotfiles",
				Namespace: "default",
				DependsOn: []string{"HomeBrewPackages/tools"},
			},
		},
		Spec: resource.ManagedFileSpec{
			Source:      "content",
			Destination: "/tmp/test",
		},
	}

	d, err := Build([]resource.Resource{brew, file}, "default")
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}
	if len(d.Nodes()) != 2 {
		t.Errorf("node count = %d, want 2", len(d.Nodes()))
	}

	order, err := d.TopologicalOrder()
	if err != nil {
		t.Fatalf("TopologicalOrder() error: %v", err)
	}
	pos := make(map[NodeID]int, 2)
	for i, id := range order {
		pos[id] = i
	}
	brewID := ResourceNodeID("default", "HomeBrewPackages", "tools")
	fileID := ResourceNodeID("default", "ManagedFile", "dotfiles")
	if pos[brewID] >= pos[fileID] {
		t.Errorf("brew tools must come before managed file: brew=%d file=%d", pos[brewID], pos[fileID])
	}
}
