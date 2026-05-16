# Testing

**Analysis Date:** 2026-05-16

## Framework

- **Stdlib only**: `testing` package — no testify, gomock, or other assertion libraries
- **Assertions**: manual `t.Errorf`, `t.Fatalf`, `t.Fatal`
- **Runner**: `go test -race -v ./...` (enforced in `ci/main.go`)

## Test Organization

Tests co-located with source files in the same package directory:

```
pkg/engine/apply_dag_test.go      (package engine)
pkg/providers/file_test.go        (package providers)
pkg/providers/provider_test.go    (package provider — black-box)
pkg/graph/graph_test.go
pkg/diff/diff_test.go
pkg/resource/generator_test.go
```

### Naming Patterns

| Pattern | Type | Build Tag |
|---------|------|-----------|
| `*_test.go` | Unit | none |
| `*_unit_test.go` | Explicit unit | none |
| `*_integration_test.go` | Integration | `//go:build integration` |

Integration tests: `cmd/apply_cancel_integration_test.go`

## Table-Driven Tests

Standard Go table-driven pattern used in:
- `pkg/resource/generator_test.go`
- `pkg/providers/file_test.go`
- `pkg/diff/diff_test.go`

```go
tests := []struct {
    name  string
    input string
    want  string
}{
    {"empty input", "", ""},
    {"normal case", "foo", "foo"},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        got := fn(tt.input)
        if got != tt.want {
            t.Errorf("got %q, want %q", got, tt.want)
        }
    })
}
```

## Fakes & Mocks

All hand-written in the same test file — no mocking framework:

| Fake | File | Purpose |
|------|------|---------|
| `fakeProvider` | `pkg/engine/apply_dag_test.go` | Simulates provider apply/delete |
| `mockProvider` | `pkg/providers/provider_test.go` | Records calls to provider methods |
| `noopStateBackend` | `pkg/providers/provider_test.go` | No-op state backend |
| `blockingProvider` | `pkg/providers/cancellation_test.go` | Blocks to test context cancellation |

## Fixtures

- `t.TempDir()` for filesystem isolation in file provider tests
- `os.WriteFile()` to create temp YAML manifest files inline in test functions
- No `testdata/` directory found — fixtures created in-memory or via temp files

## Architecture Enforcement Test

`tools/check_context_usage_test.go` is a special meta-test that:
1. Scans all `.go` files under `pkg/`
2. Fails if any file calls `context.Background()`
3. Enforces the convention that library code must not create root contexts

```go
// This test will FAIL if found in pkg/
ctx := context.Background()
```

## CI Integration

CI pipeline in `ci/main.go` (Dagger):
- `go vet ./...` — static analysis gate
- `go test -race -v ./...` — all unit tests with race detector
- Integration tests run separately with `-tags integration`

## Coverage

- No enforced coverage threshold
- No `.coverprofile` configuration
- **Gap**: GoProvider, NpmProvider, AISkillProvider, and S3 backend have no unit tests

---

*Testing analysis: 2026-05-16*
