# E2E Testing Guide for dotisan

This document describes how to run comprehensive end-to-end tests for dotisan to verify all functionality works correctly.

## Why This Guide Exists

dotisan manages critical system files (dotfiles, packages, directories). A bug in apply, drift detection, or state management could corrupt or delete user files. All changes should be tested against test data, never production files.

## Test Environment Setup

### Important: Always Use a Test Directory

NEVER test with real dotfiles. Always use a temporary test directory.

```bash
# Create test directories
mkdir -p ~/dotfiles-test/dotfiles/shell
mkdir -p ~/dotfiles-test/resources/dotfiles/shell
mkdir -p ~/dotfiles-test/resources/dotfiles/git
```

### Configure dotisan to Use Test Directory

Create `~/.dotisan/config.yaml` (or any config path dotisan reads):

```yaml
# Configure test environment
dotfiles_root: /Users/piotrek/dotfiles-test
state:
  backend: local
  path: /Users/piotrek/dotfiles-test/state.json
```

### Resource Definition Files

Place resource YAML files in `~/dotfiles-test/resources/` (or any subdirectory under resources/). Example:

```yaml
# ~/dotfiles-test/resources/shell/config.yaml
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: testrc
spec:
  sourceFile: dotfiles/shell/testrc.sh
  destination: ~/.testrc
  mode: "0644"
```

Source files referenced in `sourceFile` should be in `~/dotfiles-test/resources/` (the "resources" subdirectory under dotfiles_root).

## Test Scenarios

### File Provider Tests

#### Test 1: Initial Apply (Create File)

```bash
# Set up test file
echo '# Test config' > ~/dotfiles-test/resources/dotfiles/shell/testrc.sh

# Create resource definition
cat > ~/dotfiles-test/resources/shell/config.yaml <<'EOF'
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: testrc
spec:
  sourceFile: dotfiles/shell/testrc.sh
  destination: ~/.testrc
EOF

# Run apply
./dotisan apply --confirm
# Expected: File created at ~/.testrc

# Verify
cat ~/.testrc  # Should show content
cat state.json  # Should have dest_hash
```

#### Test 2: In-Sync Check

```bash
./dotisan plan
# Expected: "No changes. Your infrastructure matches the configuration."
```

#### Test 3: Modification Detection

```bash
# Modify source file
echo '# Modified content' > ~/dotfiles-test/resources/dotfiles/shell/testrc.sh

./dotisan plan
# Expected: Shows "~1 to change" with diff
```

#### Test 4: Apply Modification

```bash
./dotisan apply --confirm
# Expected: File updated at destination
```

#### Test 5: Drift Detection

```bash
# Manually modify destination (simulate external change)
echo '# Manual drift' > ~/.testrc

./dotisan plan
# Expected: Shows drift with diff
```

#### Test 6: Apply Restores Drift

```bash
./dotisan apply --confirm
# Expected: File restored to source content
cat ~/.testrc  # Should show source content, not drift
```

#### Test 7: Resource Removal

```bash
# Remove resource definition
rm ~/dotfiles-test/resources/shell/config.yaml

./dotisan plan
# Expected: Shows "-1 to destroy"
```

### Directory Provider Tests

```bash
# Set up directory source
mkdir -p ~/dotfiles-test/resources/dotfiles/git
echo 'test' > ~/dotfiles-test/resources/dotfiles/git/file.txt

# Create definition
cat > ~/dotfiles-test/resources/shell/dirs.yaml <<'EOF'
---
apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedDirectory
metadata:
  name: testdir
spec:
  sourceDir: dotfiles/git
  destination: ~/.testdir
  recursive: true
EOF

./dotisan apply --confirm
# Expected: Directory synced to ~/.testdir/
```

### Brew Provider Tests

```bash
# Create brew package definition
cat > ~/dotfiles-test/resources/brew.yaml <<'EOF'
---
apiVersion: github.com/wasilak/dotisan/v1
kind: BrewPackages
metadata:
  name: test-packages
spec:
  formulae:
    - name: hello
EOF

./dotisan apply --confirm
# Expected: Package installed, in state
```

### State Management Tests

#### Test 13: State Import

```bash
# Create an existing file
echo '# Existing' > ~/.test-import

# Import it into state (without applying any managed content)
./dotisan state import ManagedFile/test-import ~/.test-import
# After confirmation, file is tracked but not managed

./dotisan state list
# Expected: Shows as "orphaned" (exists but not in config)
```

#### Test 14: State Remove

```bash
# Remove from state (without deleting file)
./dotisan state remove ManagedFile/test-import

./dotisan state list
# Expected: Resource removed from list
# File ~/.test-import still exists on disk
```

### Utility Commands

```bash
./dotisan doctor
# Expected: All checks pass

./dotisan init
# Creates default ~/.config/dotisan structure
```

## Cleanup

After all tests, clean up test files:

```bash
# Remove test dotfiles
rm -rf ~/dotfiles-test

# Remove test config
rm ~/.dotisan/config.yaml

# Remove any test destination files
rm ~/.testrc ~/.test-import ~/.testdir 2>/dev/null
```

## Expected Test Results

All tests should pass:

| Test | Expected |
|------|----------|
| Initial apply | File created with checksum in state |
| Plan in sync | "No changes" message |
| Modification | Shows ~N to change |
| Drift detection | Shows drift with diff |
| Apply modification | Updates file correctly |
| Restore drift | Restores file to source |
| Removal | Shows -N to destroy |
| State import | Resource in list as orphaned |
| State remove | Resource removed from list |
| Doctor | All checks pass |

## Common Issues and Fixes

### Issue: "config.yaml" Not Loading

The loader skips `config.yaml` files. If your resource definition isn't loading, make sure:
1. It's named something other than `config.yaml` (e.g., `files.yaml`, `shell.yaml`)
2. It has the correct `apiVersion` and `kind` fields
3. It's in the resources/ subdirectory under dotfiles_root

### Issue: Source File Not Found

Source files in `sourceFile` are relative to the `resources/` subdirectory under dotfiles_root. If your source file is at `~/dotfiles-test/resources/dotfiles/shell/test.sh`, use `sourceFile: dotfiles/shell/test.sh`.

### Issue: Drift Shows But Apply Doesn't Restore

Ensure the fixes from commit `f42bd8d` are applied:
- FileProvider.Apply() processes drift
- HasChanges includes TotalDrifted
- resourceToStateEntry calculates checksums

### Issue: Old State Showing

If old resources appear in state list, check:
1. Are you using the correct state path from config?
2. Is there an old state file at `~/.config/dotisan/state.json`?

Clean up old state: `rm ~/.config/dotisan/state.json`