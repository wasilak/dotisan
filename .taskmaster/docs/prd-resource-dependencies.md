# PRD: Terraform-style Resource Dependencies for dotisan

## Overview

dotisan currently applies resources in arbitrary order (filesystem walk + map iteration). This makes it impossible to express that a Homebrew formula requires a tap to be registered first, or that a config file must be placed after the tool it configures is installed. This PRD specifies adding Terraform-style `depends_on` at both the resource and item level, backed by a DAG topological sort engine that determines apply order.

## Problem Statement

- Homebrew formulae from a third-party tap fail if the tap isn't installed first
- npm global packages may depend on a specific node version installed by another resource
- Dotfiles may reference binaries that must be installed before the file is placed
- No way to express "run group A completely before starting group B"

## Goals

1. Add `depends_on` to every resource (metadata level) — resource waits for named resources to complete
2. Add `depends_on` to every item type (Package, Tap, GoPackage, FileSpec) — item waits for named items to complete
3. Add `version` to `Tap` struct for consistency parity with other item types
4. Implement a DAG engine that topologically sorts both resources and items into a valid apply order
5. Detect dependency cycles at plan time and report them with the cycle path (never silently ignore)
6. Support `depends_on` in `GeneratorSpec` — all generated files inherit the dependency
7. Backwards-compatible: existing configs with no `depends_on` continue to work unchanged

## Non-Goals

- Parallel execution of independent items (sequential-only for this version)
- Automatic dependency inference from expression references (Terraform's implicit deps) — explicit only
- Remote/cross-repo dependency references
- `depends_on` within values.yaml or template rendering

## Address Format

Dependencies use the same format as existing CLI targets (from `ParseTargets()` in `pkg/engine/options.go`):

| Format | Meaning | Example |
|--------|---------|---------|
| `Kind/GroupName` | Entire resource group | `HomeBrewTaps/my-taps` |
| `Kind/GroupName[ItemName]` | Single item within a group | `HomeBrewTaps/my-taps[homebrew/cask-fonts]` |

Rules:
- References must resolve to existing resources/items; unknown references → plan error
- Self-references are forbidden (caught by cycle detection)
- Resource-level `depends_on` on another resource means: all items in this resource wait until ALL items in the target resource complete

## Schema Changes

### 1. `pkg/resource/types_common.go`

```go
// Package — add DependsOn
type Package struct {
    Name      string   `yaml:"name"`
    Version   string   `yaml:"version,omitempty"`
    DependsOn []string `yaml:"depends_on,omitempty"`
}

// Tap — add Version and DependsOn
type Tap struct {
    Name      string   `yaml:"name"`
    Version   string   `yaml:"version,omitempty"`
    DependsOn []string `yaml:"depends_on,omitempty"`
}
```

### 2. `pkg/resource/go.go`

```go
type GoPackage struct {
    Module    string   `yaml:"module"`
    Version   string   `yaml:"version"`
    DependsOn []string `yaml:"depends_on,omitempty"`
}
```

### 3. `pkg/resource/file.go` — `FileSpec`

```go
type FileSpec struct {
    Source      string   `yaml:"source,omitempty"`
    SourceFile  string   `yaml:"sourceFile,omitempty"`
    Destination string   `yaml:"destination"`
    Template    bool     `yaml:"template,omitempty"`
    Mode        string   `yaml:"mode,omitempty"`
    DependsOn   []string `yaml:"depends_on,omitempty"`
}
```

### 4. `pkg/resource/file.go` — `GeneratorSpec`

```go
type GeneratorSpec struct {
    SourceKey          string   `yaml:"sourceKey"`
    Template           string   `yaml:"template,omitempty"`
    SourceFilePattern  string   `yaml:"sourceFilePattern,omitempty"`
    DestinationPattern string   `yaml:"destinationPattern"`
    Mode               string   `yaml:"mode,omitempty"`
    DependsOn          []string `yaml:"depends_on,omitempty"` // inherited by all generated FileSpecs
}
```

When generator expands into FileSpec entries, each generated FileSpec inherits the generator's `DependsOn`.

### 5. `pkg/resource/resource.go` — `Metadata`

```go
type Metadata struct {
    Name        string            `yaml:"name"`
    Namespace   string            `yaml:"namespace"`
    Labels      map[string]string `yaml:"labels,omitempty"`
    Annotations map[string]string `yaml:"annotations,omitempty"`
    DependsOn   []string          `yaml:"depends_on,omitempty"` // resource-level
}
```

## New Package: `pkg/graph`

Create `pkg/graph/` with:

### `pkg/graph/node.go`

```go
type NodeID string  // e.g. "HomeBrewTaps/my-taps" or "HomeBrewTaps/my-taps[homebrew/cask-fonts]"

type NodeKind int
const (
    NodeKindResource NodeKind = iota  // whole resource group
    NodeKindItem                      // individual item within a group
)

type Node struct {
    ID        NodeID
    Kind      NodeKind
    DependsOn []NodeID
}
```

### `pkg/graph/dag.go`

- `Build(resources []resource.Resource) (*DAG, error)` — constructs graph from all loaded resources; returns error on unknown references
- `Validate() error` — detects cycles using DFS; returns error with the cycle path
- `TopologicalOrder() ([]NodeID, error)` — Kahn's algorithm; returns sorted node IDs, error if cycle
- Internal: `detectCycle() []NodeID` — returns cycle path for error messages

Algorithm: **Kahn's algorithm** (BFS-based):
1. Compute in-degree for every node
2. Queue all nodes with in-degree 0
3. Process queue: dequeue node, add to result, decrement in-degree of dependents
4. If result length < total nodes → cycle detected; identify cycle path with DFS for error message

### `pkg/graph/resolver.go`

- `ResolveAddress(addr string, resources []resource.Resource) (NodeID, error)` — validates address format and existence
- `ResourceID(kind, group string) NodeID` — constructs resource-level NodeID
- `ItemID(kind, group, item string) NodeID` — constructs item-level NodeID

## Engine Changes

### `pkg/engine/plan.go`

After `resourcesToGroups()`, before `groupResourcesByProvider()`:

1. Call `graph.Build(resources)` to construct DAG
2. Call `dag.Validate()` — abort plan with cycle error if invalid
3. Call `dag.TopologicalOrder()` — get sorted NodeIDs
4. Store sorted order in `PlanResult`

### `pkg/engine/apply.go`

Replace current arbitrary provider map iteration with DAG-ordered execution:

1. Load topological order from plan
2. Process items in topological order (sequential)
3. Each item: look up its provider, call `prov.Apply(singleItemPlan)`
4. Track failures; a failed item marks its dependents as `skipped` (not failed)

Add `SkippedDueToDependencyFailure` to `GroupPlan` for tracking skipped items.

### `pkg/engine/plan_result.go` (or equivalent)

Add to plan result:
```go
type PlanResult struct {
    // ... existing fields ...
    DependencyOrder []graph.NodeID  // topological order for apply
    CycleError      error           // non-nil if cycle detected
}
```

## `pkg/provider/provider.go` — `GroupPlan`

Add:
```go
type ItemStatus int
const (
    // ... existing ...
    ItemSkipped ItemStatus = iota  // skipped because a dependency failed
)
```

## Validation

Add to resource `Validate()` methods:
- Validate `DependsOn` address format (parse-time) — must match `Kind/Group` or `Kind/Group[Item]` pattern
- Cross-resource validation deferred to graph build phase (need all resources loaded)

## YAML Examples After Change

### Resource-level dependency
```yaml
apiVersion: github.com/wasilak/dotisan/v1
kind: HomeBrewPackages
metadata:
  name: core-tools
  depends_on:
    - HomeBrewTaps/my-taps
spec:
  formulae:
    - name: ripgrep
    - name: neovim
```

### Item-level dependency
```yaml
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: nvim-config
spec:
  files:
    - destination: ~/.config/nvim/init.lua
      sourceFile: templates/nvim/init.lua
      depends_on:
        - HomeBrewPackages/core-tools[neovim]
```

### Generator with inherited dependency
```yaml
spec:
  generator:
    sourceKey: skills
    destinationPattern: ~/.config/skills/{{ .Item }}.yaml
    template: "skill: {{ .Item }}"
    depends_on:
      - HomeBrewPackages/core-tools[ripgrep]
```

### Tap with version (for consistency)
```yaml
spec:
  taps:
    - name: homebrew/cask-fonts
      version: "2.0"
```

## Error Messages

### Cycle error
```
Error: dependency cycle detected:
  HomeBrewPackages/core-tools → HomeBrewTaps/my-taps → HomeBrewPackages/core-tools
```

### Unknown reference error
```
Error: resource "HomeBrewPackages/core-tools" has unknown dependency "HomeBrewTaps/nonexistent"
```

### Dependency failure skip
```
  ✓ HomeBrewTaps/my-taps[homebrew/cask-fonts]    added
  ✗ HomeBrewPackages/core-tools[neovim]            failed: ...
  ⊘ ManagedFile/nvim-config[~/.config/nvim/...]   skipped (dependency HomeBrewPackages/core-tools[neovim] failed)
```

## Test Strategy

- Unit tests for `pkg/graph`: cycle detection (simple cycle, transitive cycle, no cycle), topological order correctness, address resolution
- Unit tests for schema: YAML round-trip for all item types with `depends_on`
- Unit tests for generator expansion: `DependsOn` inheritance into generated FileSpecs
- Integration tests: full plan+apply with dependency ordering — verify apply order matches DAG order
- Integration tests: cycle detected at plan time → no apply attempted
- Integration tests: skip propagation — item failure → dependents show as skipped

## Affected Files

| File | Change |
|------|--------|
| `pkg/resource/types_common.go` | Add `DependsOn` to `Package`; add `Version`+`DependsOn` to `Tap` |
| `pkg/resource/go.go` | Add `DependsOn` to `GoPackage` |
| `pkg/resource/file.go` | Add `DependsOn` to `FileSpec` and `GeneratorSpec` |
| `pkg/resource/generator.go` | Propagate `GeneratorSpec.DependsOn` to expanded `FileSpec` entries |
| `pkg/resource/resource.go` | Add `DependsOn []string` to `Metadata` |
| `pkg/graph/` (new package) | `node.go`, `dag.go`, `resolver.go` |
| `pkg/engine/plan.go` | Build DAG after resource load; store topological order |
| `pkg/engine/apply.go` | Apply in DAG order; propagate skips on failure |
| `pkg/provider/provider.go` | Add `ItemSkipped` status to GroupPlan |
| Tests throughout | Unit + integration tests for all above |

## Out of Scope / Future Work

- Parallel apply for independent items (Phase 2)
- Implicit dependency inference (like Terraform's expression references)
- `depends_on` in values.yaml
- Dependency visualization (`dotisan graph` command)
