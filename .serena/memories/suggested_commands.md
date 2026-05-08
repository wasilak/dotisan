# nim — Suggested Commands

## Project Initialization
```bash
# Initialize Go module (first task)
go mod init nim

# Install Cobra
go get github.com/spf13/cobra

# Install other dependencies
go get github.com/Masterminds/sprig/v3
go get github.com/martinohmann/go-difflib
go get github.com/sergi/go-diff
go get github.com/charmbracelet/lipgloss
go get github.com/go-playground/validator/v10
go get github.com/minio/minio-go/v7
go get gopkg.in/yaml.v3
```

## Development Commands
```bash
# Build the project
go build -o nim .

# Run with plan command (dry-run default)
go run main.go plan

# Run with apply and confirm
go run main.go apply --confirm

# Run tests
go test ./...

# Run specific package tests
go test ./pkg/providers/...

# Run with verbose output
go test -v ./...
```

## Task Master Commands
```bash
# Get next task
task-master next

# Show task details
task-master show <id>

# List all tasks
task-master list

# Mark task complete
task-master set-status --id=<id> --status=done

# Update subtask with notes
task-master update-subtask --id=<id> --prompt="implementation notes"
```

## MCP TaskMaster Tools (via Claude Code)
```
# Get all tasks
taskmaster_get_tasks

# Get specific task
taskmaster_get_task --id=1

# Get next recommended task
taskmaster_next_task

# Set task status
taskmaster_set_task_status --id=1 --status=in-progress
```

## Git Commands (Standard)
```bash
# Check status
git status

# Stage changes
git add .

# Commit with meaningful message
git commit -m "feat: implement X feature"

# View recent commits
git log --oneline -10
```

## File System Commands (Darwin)
```bash
# List files
ls -la

# Find Go files
fd "\.go$"

# Search code
rg "func.*Reconcile"

# Check Go version
go version
```

## Testing Commands (to be added as project grows)
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. ./...
```

## Linting & Formatting (to be configured)
```bash
# Format Go code
go fmt ./...

# Vet for issues
go vet ./...

# Run golangci-lint (if configured)
golangci-lint run
```
