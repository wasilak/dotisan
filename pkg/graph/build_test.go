package graph

import (
	"strings"
	"testing"

	"github.com/wasilak/dotisan/pkg/resource"
)

// makeHB constructs a minimal HomeBrewPackages resource.
func makeHB(name string, dependsOn ...string) *resource.HomeBrewPackages {
	return &resource.HomeBrewPackages{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewPackages",
			Metadata: resource.Metadata{
				Name:      name,
				DependsOn: dependsOn,
			},
		},
	}
}

func TestBuild_NoDeps(t *testing.T) {
	resources := []resource.Resource{
		makeHB("tools"),
		makeHB("fonts"),
	}
	d, err := Build(resources)
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
		makeHB("tools"),
		makeHB("fonts", "HomeBrewPackages/tools"),
	}
	d, err := Build(resources)
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
	toolsID := ResourceNodeID("HomeBrewPackages", "tools")
	fontsID := ResourceNodeID("HomeBrewPackages", "fonts")
	if pos[toolsID] >= pos[fontsID] {
		t.Errorf("tools must come before fonts: tools=%d fonts=%d", pos[toolsID], pos[fontsID])
	}
}

func TestBuild_CycleReturnsError(t *testing.T) {
	// A depends on B, B depends on A
	resources := []resource.Resource{
		makeHB("a", "HomeBrewPackages/b"),
		makeHB("b", "HomeBrewPackages/a"),
	}
	d, err := Build(resources)
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
		makeHB("a", "[bad]"),
	}
	if _, err := Build(resources); err == nil {
		t.Fatal("expected error for invalid dependency address")
	}
}

func TestBuild_Empty(t *testing.T) {
	d, err := Build(nil)
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
			Metadata:   resource.Metadata{Name: "extra-taps"},
		},
		Spec: resource.HomeBrewTapsSpec{},
	}
	packages := makeHB("tools") // no explicit dependsOn

	d, err := Build([]resource.Resource{taps, packages})
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
	tapID := ResourceNodeID("HomeBrewTaps", "extra-taps")
	pkgID := ResourceNodeID("HomeBrewPackages", "tools")
	if pos[tapID] >= pos[pkgID] {
		t.Errorf("taps must be applied before packages: tap=%d pkg=%d", pos[tapID], pos[pkgID])
	}
}

func TestBuild_ImplicitTapDeps_CasksAlsoDependOnTaps(t *testing.T) {
	taps := &resource.HomeBrewTaps{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewTaps",
			Metadata:   resource.Metadata{Name: "cask-fonts"},
		},
		Spec: resource.HomeBrewTapsSpec{},
	}
	casks := &resource.HomeBrewCasks{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewCasks",
			Metadata:   resource.Metadata{Name: "fonts"},
		},
		Spec: resource.HomeBrewCasksSpec{},
	}

	d, err := Build([]resource.Resource{taps, casks})
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
	tapID := ResourceNodeID("HomeBrewTaps", "cask-fonts")
	caskID := ResourceNodeID("HomeBrewCasks", "fonts")
	if pos[tapID] >= pos[caskID] {
		t.Errorf("taps must be applied before casks: tap=%d cask=%d", pos[tapID], pos[caskID])
	}
}

func TestBuild_ImplicitTapDeps_NoDuplicateWhenExplicit(t *testing.T) {
	taps := &resource.HomeBrewTaps{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewTaps",
			Metadata:   resource.Metadata{Name: "extra-taps"},
		},
		Spec: resource.HomeBrewTapsSpec{},
	}
	// packages already declares the tap dep explicitly
	packages := makeHB("tools", "HomeBrewTaps/extra-taps")

	d, err := Build([]resource.Resource{taps, packages})
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	tapID := ResourceNodeID("HomeBrewTaps", "extra-taps")
	pkgID := ResourceNodeID("HomeBrewPackages", "tools")
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

func TestBuild_ImplicitTapDeps_GlobalInjection(t *testing.T) {
	// Taps are injected into all packages regardless of what namespace was in YAML.
	taps := &resource.HomeBrewTaps{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "HomeBrewTaps",
			Metadata:   resource.Metadata{Name: "some-taps", Namespace: "other"},
		},
		Spec: resource.HomeBrewTapsSpec{},
	}
	packages := makeHB("tools")

	d, err := Build([]resource.Resource{taps, packages})
	if err != nil {
		t.Fatalf("Build() error: %v", err)
	}

	pkgID := ResourceNodeID("HomeBrewPackages", "tools")
	deps := d.DependenciesOf(pkgID)
	if len(deps) != 1 {
		t.Errorf("expected 1 implicit tap dep, got %d: %v", len(deps), deps)
	}
}

func TestBuild_NamespacePrefixInDependsOn_Stripped(t *testing.T) {
	// A DependsOn with a namespace prefix resolves to the same NodeID as one without.
	tools := makeHB("tools")
	// "default/HomeBrewPackages/tools" should resolve to "HomeBrewPackages/tools"
	fonts := makeHB("fonts", "default/HomeBrewPackages/tools")

	d, err := Build([]resource.Resource{tools, fonts})
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
	toolsID := ResourceNodeID("HomeBrewPackages", "tools")
	fontsID := ResourceNodeID("HomeBrewPackages", "fonts")
	if pos[toolsID] >= pos[fontsID] {
		t.Errorf("tools must come before fonts: tools=%d fonts=%d", pos[toolsID], pos[fontsID])
	}
}

func TestBuild_MultipleKinds(t *testing.T) {
	// ManagedFile depends on HomeBrewPackages
	brew := makeHB("tools")
	file := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata: resource.Metadata{
				Name:      "dotfiles",
				DependsOn: []string{"HomeBrewPackages/tools"},
			},
		},
		Spec: resource.ManagedFileSpec{
			Source:      "content",
			Destination: "/tmp/test",
		},
	}

	d, err := Build([]resource.Resource{brew, file})
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
	brewID := ResourceNodeID("HomeBrewPackages", "tools")
	fileID := ResourceNodeID("ManagedFile", "dotfiles")
	if pos[brewID] >= pos[fileID] {
		t.Errorf("brew tools must come before managed file: brew=%d file=%d", pos[brewID], pos[fileID])
	}
}
