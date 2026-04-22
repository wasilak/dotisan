# PRD: npm and Go Provider Implementation Verification

## Status: Draft

## 1. Overview

### Goals
- Verify npm provider works correctly in E2E scenarios
- Fix Go provider's getInstalledPackages() bug causing all packages to always appear as additions
- Add drift detection capability for npm packages
- Add import functionality for both npm and Go providers
- Ensure state tracking works correctly for both providers

### Background
The npm and Go providers exist in dotisan but have not been comprehensively tested in E2E scenarios. Issues similar to those discovered in the file/brew providers during recent testing may exist, and some features like drift detection are missing or incomplete.

## 2. Current Implementation Status

### npm Provider (npm.go)
| Feature | Status | Notes |
|---------|--------|-------|
| Reconcile | Implemented | Compares desired vs installed global packages |
| Apply Install | Implemented | Uses npm install -g |
| Apply Remove | Implemented | Uses npm uninstall -g |
| Drift Detection | Not Implemented | No mechanism to detect manual uninstalls |
| Import | Not Implemented | Returns error |
| Modification Detection | N/A | Package versions not tracked |

### Go Provider (go.go)
| Feature | Status | Notes |
|---------|--------|-------|
| Reconcile | Partial | Uses which for binaries, broken getInstalledPackages |
| Apply Install | Implemented | Uses go install |
| Apply Remove | Implemented | Uses rm to delete binary |
| Drift Detection | Not Implemented | No mechanism |
| Import | Not Implemented | Returns error |
| Modification Detection | N/A | No version tracking |

### Critical Bug in Go Provider
The getInstalledPackages() function always returns an empty map:
```go
func (p *GoProvider) getInstalledPackages() (map[string]string, error) {
    return make(map[string]string), nil
}
```
This causes ALL packages to always show as additions since the provider thinks nothing is installed.

## 3. Functional Requirements

### FR1: Fix Go getInstalledPackages
As a user, when I run `dotisan plan` with Go modules already installed, I should see them as "in sync" not additions.

Current Behavior: All packages show as additions
Desired Behavior: Installed packages show as in sync

### FR2: Add npm Import Capability
As a user, I want to import existing npm packages into dotisan management using dotisan state import.

Current Behavior: Import returns error
Desired Behavior: Import discovers existing global npm packages

### FR3: Add Go Import Capability  
As a user, I want to import existing Go modules into dotisan management using dotisan state import.

Current Behavior: Import returns error
Desired Behavior: Import discovers existing Go modules

### FR4: npm Drift Detection
As a user, when I manually uninstall an npm package, I want dotisan to detect this drift.

Current Behavior: No detection
Desired Behavior: Warning or drift detection in plan output

### FR5: Go Drift Detection
As a user, when I manually remove a Go binary, I want dotisan to detect this drift.

Current Behavior: No detection
Desired Behavior: Drift detected in plan output

## 4. Technical Details

### npm Provider Implementation

#### getInstalledPackages
Uses npm list -g --json to get global packages with versions.

```go
func (p *NpmProvider) getInstalledPackages() (map[string]string, error)
```

#### Import Implementation Needed
Parse npm list -g --json output to discover existing packages.

### Go Provider Implementation

#### getInstalledPackages Fix Required
Replace empty map return with actual detection:

Option 1: Use `go list -m all` to get installed modules
Option 2: Check PATH directories for binaries (GOBIN, GOPATH/bin, ~/go/bin)

#### Binary Name Extraction
Current implementation extracts binary name from module path:
```go
parts := strings.Split(module, "/")
binaryName := parts[len(parts)-1]
```

## 5. Implementation Plan

### Phase 1: Bug Fixes

#### Task 1: Fix Go getInstalledPackages
- Replace empty map return with proper detection logic
- Use go list -m all OR check PATH directories
- Test: Apply then plan should show in sync

#### Task 2: Add npm Import
- Implement Import() using npm list -g --json
- Parse output to create ResourceState
- Test: Import discovers existing packages

#### Task 3: Add Go Import
- Implement Import() using go list 
- Parse output to create ResourceState
- Test: Import discovers existing modules

### Phase 2: E2E Testing

#### Task 4: npm Provider Tests
1. Create NpmPackages resource definition
2. Apply to install package
3. Verify in state with version
4. Plan shows in sync
5. Manual uninstall shows warning/drift
6. Remove resource removes package

#### Task 5: Go Provider Tests  
1. Create GoPackages resource definition
2. Apply to install module
3. Verify in state
4. Plan shows in sync (after bug fix)
5. Manual removal shows drift
6. Remove resource removes binary

#### Task 6: State Management Tests
1. Verify state.json contains npm package versions
2. Verify state.json contains Go module info
3. Verify state list shows correct status

## 6. Success Criteria

### npm Provider
- [ ] dotisan apply installs global npm packages
- [ ] dotisan plan shows correct in-sync/addition status  
- [ ] dotisan state list shows managed npm packages
- [ ] dotisan state import discovers existing packages
- [ ] Apply removes packages when resource deleted

### Go Provider
- [ ] getInstalledPackages returns installed modules (BUG FIX)
- [ ] dotisan apply installs Go modules  
- [ ] dotisan plan shows correct in-sync/addition status
- [ ] dotisan state list shows managed Go packages
- [ ] dotisan state import discovers existing modules
- [ ] Apply removes binaries when resource deleted

## 7. Risks and Mitigations

### Risk 1: Go Module Detection Complexity
Go modules can be in multiple locations (GOPATH, GOBIN, vendor, ~/go/bin)
Mitigation: Start with binary check via which; expand locations as needed

### Risk 2: Version Comparison  
npm and Go have different version resolution (semver, ranges, latest)
Mitiation: Don't implement version comparison for MVP; track presence only

### Risk 3: Test Environment
Tests require npm/Go installed which may not be in CI
Mitigation: E2E tests run locally with real binaries

## 8. Dependencies

- pkg/providers/npm.go (existing, needs fixes)
- pkg/providers/go.go (existing, needs fixes)  
- pkg/resource/npm.go (existing)
- pkg/resource/go.go (existing)
- pkg/cmdutil (existing)

## 9. Out of Scope

- Project-local npm/Go packages (global only)
- Version pinning enforcement
- Private npm registries / private Go modules
- Upgrade/downgrade detection