# dotisan Bug Fix PRD - State & Plan Issues

**Document Type:** PRD (Project Requirements Document)  
**Status:** Draft  
**Created:** 2025-04-24  
**Last Updated:** 2025-04-24  
**Priority:** P0 - Critical Bug Fixes

---

## Executive Summary

The dotisan CLI has several critical bugs affecting state persistence and the plan/apply workflow:

1. State is overwritten on each apply (loses previously tracked items)
2. TotalAdditions counts groups, not individual items (shows "1" instead of "4")
3. InSync resources are not detected after apply
4. State list shows multiple items in single table row

These bugs make the app essentially non-functional for tracking package state across multiple runs.

---

## Root Cause Analysis

### Issue Flow (Current)
```
1. User runs: dotisan apply --confirm
2. Plan shows 4 packages as additions
3. Apply completes "successfully"
4. Next run: dotisan plan
5. Shows 1 package as "to add" (dagger only!)
6. Other 3 show nothing OR as new additions
```

**Root Cause:** `ApplyWithProgress` creates a BRAND NEW state object, adds only currently-successful items, and saves it - destroying all previously tracked resources.

```go
// Current code in engine.go:442-456
if successCount > 0 {
    newState := state.NewState()  // ← BUG: Creates empty new state!
    for i, item := range workItems {
        if results[i].success {
            newState.SetResourceGroup(...)  // Only adds current items
        }
    }
    e.StateBackend.Save(ctx, newState)  // Overwrites previous state!
}
```

---

## Task List

### Task 1: Fix State Persistence - State Overwrite Bug [P0-CRITICAL]
**Issue:** State file is overwritten, losing all previously tracked items  
**Files:** `pkg/engine/engine.go:442-456`  
**Effort:** 2h

**Current behavior:**
```go
if successCount > 0 {
    newState := state.NewState()  // ← Creates BRAND NEW state!
    for i, item := range workItems {
        if results[i].success {
            newState.SetResourceGroup(...)
        }
    }
    // Previous state is LOST!
}
```

**Required behavior:**
- Load existing state first (if exists)
- Merge new items INTO existing state (not replace)
- Keep items that were already tracked

**Steps:**
- [ ] Load current state with `e.StateBackend.Load(ctx)`
- [ ] Handle case when no state exists (create new)
- [ ] Iterate over existing resources in current state
- [ ] Add new successful items to existing state  
- [ ] Only create new state if none exists initially

**Acceptance Criteria:**
1. Run apply with 3 packages → success
2. Run apply again with same 3 packages → shows all 3 as InSync
3. Run state list → shows all 3 packages tracked

---

### Task 2: Fix TotalAdditions Count - Counts Groups Not Items [P0-CRITICAL]
**Issue:** Shows "1 to add" when 4 items exist  
**Files:** `pkg/engine/engine.go:127`  
**Effort:** 1h

**Current code:**
```go
result.TotalAdditions += len(plan.Additions)  // ← Counts groups, not items!
```

**Required behavior:** Count individual items within each GroupAddition

**Steps:**
- [ ] Change: `result.TotalAdditions += len(plan.Additions)`
- [ ] To: Loop and sum len(addition.Items) for each addition
- [ ] Apply same fix to TotalModifications, TotalRemovals, TotalInSync

**Acceptance Criteria:**
1. Plan shows 4 items → Footer shows "4 to add" (not "1")
2. All item counts in plan summary are accurate

---

### Task 3: Fix Plan Shows All Items as Additions (InSync Not Working) [P1]
**Issue:** All previously-installed packages show as ✚ additions, not ✓ in-sync  
**Files:** `pkg/providers/brew.go`, `pkg/engine/engine.go`  
**Effort:** 3h

**Root cause:** After Task 1 fixed (state persists), provider's Reconcile should detect InSync. May auto-resolve once state is properly saved.

**Steps:**
- [ ] Fix Task 1 first
- [ ] Add debug logging to verify state loads correctly
- [ ] Trace through provider's Reconcile logic
- [ ] Verify items in both desired AND state return as InSync

**Acceptance Criteria:**
1. After apply, run plan → ✓ shows for already-tracked items
2. ✚ shows only for truly NEW packages

---

### Task 4: Fix State List - One Row Per Item [P1]
**Issue:** Table shows "ripgrep, htop, podman" in single cell  
**Files:** `cmd/state.go:340`  
**Effort:** 1h

**Current code:**
```go
t.Row(res.Kind, res.Group, strings.Join(itemNames, ", "), "managed")
// → "ripgrep, htop, podman" in single cell
```

**Required behavior:** One row per item

**Steps:**
- [ ] Change loop to iterate over `res.Items`
- [ ] Create one `t.Row()` call per item
- [ ] Duplicate Kind and Group for each row
- [ ] Handle empty Items gracefully

**Acceptance Criteria:**
```
┌──────────────┬────────────┬────────────┬─────────┐
│     KIND     │   GROUP    │    ITEM   │ STATUS  │
├──────────────┼────────────┼────────────┼─────────┤
│ BrewPackages │ core-tools │  ripgrep  │ managed │
│ BrewPackages │ core-tools │   htop   │ managed │
│ BrewPackages │ core-tools │  podman  │ managed │
└──────────────┴────────────┴────────────┴─────────┘
```

---

### Task 5: Verify InSync Handling in DisplayPlanList [P2]
**Issue:** When skipEmpty=true (apply), groups with only InSync are filtered out  
**Files:** `cmd/plan.go:DisplayPlanList`  
**Effort:** 1h

**Verification needed:**
- [ ] After Tasks 1-3 fixed, run plan to verify InSync items show with ✓ icon
- [ ] If not showing, trace through provider's Reconcile logic
- [ ] Ensure DisplayPlanList handles InSync when skipEmpty=false

---

### Task 6: Add Debug Logging for State Operations [P2]
**Issue:** Hard to diagnose state bugs  
**Files:** `pkg/engine/engine.go`, `pkg/state/state.go`  
**Effort:** 1h

**Purpose:** Make state bugs easier to diagnose

**Steps:**
- [ ] Add log statement when loading state (count, path)
- [ ] Add log statement when saving state (resource count)
- [ ] Add verbose flag to enable debug logs

---

### Task 7: Add Integration Test for Apply + Plan Flow [P3]
**Issue:** No test coverage for apply→plan cycle  
**Files:** `pkg/engine/engine_test.go`  
**Effort:** 2h

**Purpose:** Prevent regression

**Test scenario:**
```go
func TestApplyPlanLifecycle(t *testing.T) {
    // 1. Create test config with 3 packages
    // 2. Run apply
    // 3. Verify state saved with 3 items
    // 4. Run plan again
    // 5. Verify all 3 show as InSync (✓), not additions (✚)
}
```

---

## Testing Plan

### Test Scenario 1: Fresh Apply
```bash
# Setup: Clean state (delete if exists)
rm -f ~/.local/share/dotisan/state.json
rm -f ~/.config/dotisan/state.json

# Run apply
dotisan apply --confirm

# Verify
dotisan state list
# Should show all 4 packages

dotisan plan
# Should show all 4 as ✓ in-sync
```

### Test Scenario 2: Second Apply
```bash
# Run apply again (no changes)
dotisan apply --confirm

# Verify
dotisan plan
# Should show "No changes to apply" or all ✓ in-sync
```

### Test Scenario 3: State Persistence
```bash
# Kill app, restart
dotisan plan

# Verify state loaded correctly
dotisan state list
# Should show same 4 packages
```

---

## Priority Summary

| Priority | Task | Impact | Effort |
|----------|------|--------|--------|
| P0 | #1 State Overwrite | APP BROKEN | 2h |
| P0 | #2 Count Bug | Shows wrong numbers | 1h |
| P1 | #4 Table Rows | UX issue | 1h |
| P1 | #3 InSync | May auto-resolve | 3h |
| P2 | #5 Verify | Verify fixed | 1h |
| P2 | #6 Debug | Future debugging | 1h |
| P3 | #7 Tests | Regression prevention | 2h |

**Total estimated effort:** ~11 hours

---

## Related Files

- `pkg/engine/engine.go` - Engine with apply logic
- `pkg/state/state.go` - State management
- `pkg/providers/brew.go` - Homebrew provider 
- `cmd/state.go` - State list command
- `cmd/plan.go` - Plan display logic
- `cmd/apply.go` - Apply command

---

## Notes

- Task 1 is the CRITICAL fix - without it, the app cannot track multiple packages
- Tasks 1-3 are related to the state lifecycle
- Task 4 is independent but important UX fix
- Tasks 5-7 are verification/improvements after core fixes