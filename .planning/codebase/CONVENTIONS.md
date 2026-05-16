# Code Conventions

**Analysis Date:** 2026-05-16

## Naming

| Symbol | Convention | Example |
|--------|-----------|---------|
| Files | `snake_case` | `apply_dag.go` |
| Packages | lowercase, no underscores | `package engine` |
| Exported types | `PascalCase` | `type ResourceState struct` |
| Unexported | `camelCase` | `func ensureProvidersRegistered()` |
| Constructors | `New<Type>(ctx, ...) (*T, error)` | `func NewEngine(ctx, cfg) (*Engine, error)` |
| Interfaces | noun or `<Verb>er` | `Provider`, `StateBackend` |
| Test fakes | `fake<Type>`, `mock<Type>`, `noop<Type>` | `fakeProvider`, `noopStateBackend` |

## Error Handling

- Always wrap with `fmt.Errorf("operation context: %w", err)` — never discard the error chain
- Use `os.IsNotExist(err)` for graceful defaults (e.g., missing state file → fresh state)
- No panics in library code (`pkg/`) — return errors to the caller
- Sentinel errors defined as package-level `var` where callers need to match with `errors.Is`

```go
// Correct
if err := provider.Apply(ctx, res); err != nil {
    return fmt.Errorf("apply %s/%s: %w", res.Kind, res.Name, err)
}

// Incorrect — never seen in this codebase
panic("something went wrong")
```

## Logging

- Package: `log/slog` (structured, key-value)
- Levels: `Debug`, `Info`, `Warn` — `Error` rare (prefer returning errors)
- Key-value pairs, not format strings: `slog.Info("applying resource", "kind", res.Kind, "name", res.Name)`
- Logging **forbidden** in `pkg/graph/` and `pkg/state/` — caller's responsibility
- `log_level` configured via `~/.nim/config.yaml`

## Comments & Documentation

- Every file in `pkg/` opens with a package doc comment: `// Package engine provides ...`
- All exported symbols have godoc comments
- Inline comments only for non-obvious logic — not for restating what the code does

## Context Usage

- `context.Context` is **always the first parameter** of any function that does I/O
- `context.Background()` is **banned in `pkg/`** — enforced by `tools/check_context_usage_test.go`
- All provider methods accept and respect context for cancellation

```go
// Correct — in pkg/
func (p *FileProvider) Apply(ctx context.Context, res resource.Resource) error { ... }

// Banned — in pkg/
ctx := context.Background()  // fails the architecture lint test
```

## Build Tags

- Integration tests: `//go:build integration` at top of file
- Integration test files: named `*_integration_test.go`
- Run with: `go test -tags integration ./...`

## Import Ordering

Three groups, separated by blank lines:

```go
import (
    "context"       // 1. stdlib
    "fmt"

    "gopkg.in/yaml.v3"                 // 2. external
    "github.com/spf13/cobra"

    "github.com/wasilak/nim/pkg/config" // 3. internal
)
```

## Code Style

- `gofmt` / `goimports` enforced — CI runs `go vet ./...`
- No linter config (`.golangci.yml`) found — `go vet` is the enforced gate
- Short variable names (`r`, `res`, `cfg`) acceptable in short scopes; descriptive names in long functions
- Prefer early returns over deep nesting

---

*Conventions analysis: 2026-05-16*
