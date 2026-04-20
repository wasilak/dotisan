# dotisan — Task Completion Checklist

## Before Marking Task Complete

### 1. Code Implementation
- [ ] All subtasks implemented according to spec
- [ ] Code follows project conventions (see `code_style.md`)
- [ ] Error handling is comprehensive (no naked returns)
- [ ] Documentation comments added for all exported items
- [ ] No TODO/FIXME comments left (or documented as future work)

### 2. Testing
- [ ] Unit tests written for new functions/methods
- [ ] Tests pass: `go test ./...`
- [ ] Edge cases covered (empty inputs, errors, nil)
- [ ] Mock external dependencies (exec, HTTP, file system)

### 3. Manual Verification
- [ ] Build succeeds: `go build -o dotisan .`
- [ ] Basic functionality tested manually
- [ ] Help text and CLI output verified

### 4. Code Quality
- [ ] Code formatted: `go fmt ./...`
- [ ] No vet issues: `go vet ./...`
- [ ] Imports organized (standard, third-party, internal)

### 5. Task Master Update
- [ ] Implementation notes added: `task-master update-subtask --id=<id> --prompt="notes"`
- [ ] Task marked complete: `task-master set-status --id=<id> --status=done`
- [ ] Next task identified: `task-master next`

## Provider Implementation Specifics

### When Implementing a Provider (Brew, NPM, Go, Cargo, File)
- [ ] `Available()` checks for required executable via `exec.LookPath`
- [ ] `Reconcile()` generates proper `Plan` with all changes
- [ ] `Apply()` executes commands with proper error handling
- [ ] `Import()` can snapshot existing resources
- [ ] Provider registered in global registry

### When Working with State
- [ ] State schema version handled correctly
- [ ] Checksum calculation implemented
- [ ] Timestamps in RFC3339 format
- [ ] S3 backend credentials handled securely (env vars preferred)

### When Working with Templates
- [ ] Two-pass rendering implemented correctly
- [ ] Sprig functions available
- [ ] `values.yaml` templated before parsing
- [ ] Error messages include template context

## Git Workflow

### Commit Messages
Format: `<type>: <description>`

Types:
- `feat:` — New feature
- `fix:` — Bug fix
- `refactor:` — Code refactoring
- `test:` — Test additions/changes
- `docs:` — Documentation only
- `chore:` — Build/tooling changes

Example:
```
feat: implement BrewProvider Available() check

- Uses exec.LookPath to find brew executable
- Returns descriptive warning if not found
- Part of task 1.5.1
```

### Pre-Commit Checklist
- [ ] Tests pass
- [ ] No debug print statements
- [ ] No hardcoded secrets/credentials
- [ ] Commit message follows convention

## Completion Steps

1. **Run tests**: `go test ./...`
2. **Format code**: `go fmt ./...`
3. **Vet code**: `go vet ./...`
4. **Build binary**: `go build -o dotisan .`
5. **Update TaskMaster**: `task-master update-subtask --id=<id> --prompt="..."`
6. **Mark done**: `task-master set-status --id=<id> --status=done`
7. **Get next task**: `task-master next`

## Documentation Updates

If the task involves user-facing changes:
- [ ] Update README.md if needed
- [ ] Add example to docs/
- [ ] Update CLAUDE.md if workflow changes

## Common Pitfalls to Avoid
- [ ] Don't forget to register new providers
- [ ] Don't hardcode paths (use config)
- [ ] Don't ignore errors from os/exec
- [ ] Don't forget to close file handles
- [ ] Don't use fmt.Println for CLI output (use proper UI components)
