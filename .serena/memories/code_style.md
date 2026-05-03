# dotisan — Code Style & Conventions

## Language
- **Go** — All implementation in Go
- Go version: Latest stable (1.22+)

## Code Organization

### Directory Structure (Planned)
```
dotisan/
├── cmd/                    # Cobra command implementations
│   ├── root.go
│   ├── plan.go
│   ├── apply.go
│   └── ...
├── pkg/
│   ├── config/            # Config loading and parsing
│   ├── engine/            # Plan/Apply engine
│   ├── providers/         # Resource providers
│   │   ├── brew.go
│   │   ├── npm.go
│   │   ├── go.go
│   │   ├── cargo.go
│   │   └── file.go
│   ├── state/             # State management
│   ├── diff/              # Diff engine
│   └── template/          # Template rendering
└── internal/
    └── util/              # Internal utilities
```

## Naming Conventions

### General Go Conventions
- **Packages**: Lowercase, short, no underscores (`brew`, `file`, `config`)
- **Files**: Lowercase with underscores for multi-word (`brew_provider.go`)
- **Types**: PascalCase (`BrewProvider`, `ManagedFile`)
- **Interfaces**: Noun-like, often end in `-er` (`Provider`, `StateBackend`)
- **Functions**: PascalCase for exported, camelCase for private
- **Constants**: PascalCase or ALL_CAPS for exported
- **Variables**: camelCase

### Provider Pattern
- Provider structs: `<Resource>Provider` (e.g., `BrewProvider`, `FileProvider`)
- Interface methods: `Name()`, `Available()`, `Reconcile()`, `Apply()`, `Import()`

### Resource Kinds
- Resource kinds: PascalCase (`HomeBrewPackages`, `ManagedFile`) — `ManagedDirectory` has been removed
- Spec types: `<Kind>Spec` (e.g., `HomeBrewPackagesSpec`)

## Style Guidelines

### General Principles
1. **Follow standard Go conventions** — Use `gofmt`, `go vet`
2. **Explicit error handling** — Always check errors, wrap with context
3. **Interface-based design** — `Provider`, `StateBackend` interfaces
4. **Dependency injection** — Pass dependencies via constructors
5. **Context propagation** — Use `context.Context` for cancellation/timeouts

### Error Handling
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// Use custom error types for specific cases
var ErrProviderUnavailable = errors.New("provider unavailable")
```

### Documentation
- All exported types, functions, and methods must have doc comments
- Follow Go doc conventions (start with the name being declared)
```go
// BrewProvider manages Homebrew packages.
type BrewProvider struct {
    execPath string
}

// Reconcile compares desired state with actual state and returns a plan.
func (p *BrewProvider) Reconcile(desired []Resource, state []ResourceState) (Plan, error) {
    // ...
}
```

### Struct Tags
- YAML: `yaml:"field_name"`
- Validation: `validate:"required,min=1"`
```go
type HomeBrewPackagesSpec struct {
    Taps     []Tap     `yaml:"taps" validate:"dive"`
    Formulae []Package `yaml:"formulae" validate:"dive"`
}
```

### Testing Conventions
- Test files: `<name>_test.go`
- Test functions: `Test<Name>` (e.g., `TestBrewProvider_Available`)
- Table-driven tests preferred
- Use `t.Parallel()` where appropriate
- Mock external dependencies (os/exec, HTTP)

## Configuration Patterns

### Config Loading
```go
// Load from YAML
type Config struct {
    StateBackend string `yaml:"state_backend"`
    S3           S3Config `yaml:"s3,omitempty"`
}

// Use functional options for optional configuration
type ProviderOption func(*ProviderConfig)
```

## CLI Patterns (Cobra)
- Commands in `cmd/` package
- Each command in its own file
- Use `PersistentPreRunE` for shared setup
- Flags: use persistent flags for global options

## Provider Implementation Template
```go
package providers

import "context"

// Ensure interface compliance
var _ Provider = (*BrewProvider)(nil)

// BrewProvider manages Homebrew packages.
type BrewProvider struct {
    execPath string
}

// NewBrewProvider creates a new BrewProvider.
func NewBrewProvider(opts ...BrewProviderOption) *BrewProvider {
    p := &BrewProvider{}
    for _, opt := range opts {
        opt(p)
    }
    return p
}

// Name returns the provider name.
func (p *BrewProvider) Name() string {
    return "brew"
}

// Available checks if the brew executable is available.
func (p *BrewProvider) Available() (bool, string) {
    // ...
}

// Reconcile compares desired state with actual state.
func (p *BrewProvider) Reconcile(desired []Resource, state []ResourceState) (Plan, error) {
    // ...
}

// Apply executes the plan.
func (p *BrewProvider) Apply(plan Plan) error {
    // ...
}

// Import snapshots an existing resource.
func (p *BrewProvider) Import(id string) (ResourceState, error) {
    // ...
}
```

## Design Patterns Used
- **Provider Pattern** — Pluggable resource managers
- **Repository Pattern** — State backends
- **Strategy Pattern** — Diff engines
- **Factory Pattern** — Provider registry, backend selection
- **Template Method** — Base provider with common functionality
