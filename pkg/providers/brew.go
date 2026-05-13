package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/wasilak/nim/pkg/cmdutil"
	"github.com/wasilak/nim/pkg/provider"
	"github.com/wasilak/nim/pkg/resource"
)

// BrewProvider implements the Provider interface for Homebrew packages.
type BrewProvider struct {
	httpClient *http.Client
}

// NewBrewProvider creates a new BrewProvider.
func NewBrewProvider() *BrewProvider {
	return &BrewProvider{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Name returns the provider name.
func (p *BrewProvider) Name() string {
	return "homebrew"
}

// Available checks if the brew executable is available on this system.
func (p *BrewProvider) Available() (bool, string) {
	path := cmdutil.CheckExecutable("brew")
	if path == "" {
		return false, "brew not found in PATH; install from https://brew.sh"
	}
	return true, fmt.Sprintf("brew found at %s", path)
}

// Reconcile compares the desired resource groups with the current system state.
func (p *BrewProvider) Reconcile(ctx context.Context,
	desired []resource.ResourceGroup[any],
	state []provider.ResourceState,
) provider.GroupPlan {
	plan := provider.GroupPlan{}

	// Track items already processed to avoid duplicates
	processedItems := make(map[string]bool) // key: "group/item"

	// Index state by group name for quick lookup. Include both legacy and new Homebrew kinds.
	stateIndex := make(map[string]provider.ResourceState)
	for _, s := range state {
		if resource.IsBrewKind(s.Kind) {
			stateIndex[s.Group] = s
		}
	}

	// Build separate formula/cask name sets — queried with --formula/--cask to avoid
	// ambiguity when a name exists as both (e.g. oh-my-posh).
	// Taps are excluded — brew info only understands formulae and casks.
	formulaNamesSet := make(map[string]bool)
	caskNamesSet := make(map[string]bool)
	for _, group := range desired {
		switch group.Kind {
		case resource.KindHomeBrewPackages:
			for _, item := range group.Items {
				formulaNamesSet[item.Name] = true
			}
		case resource.KindHomeBrewCasks:
			for _, item := range group.Items {
				caskNamesSet[item.Name] = true
			}
		}
	}
	for _, s := range state {
		switch s.Kind {
		case resource.KindHomeBrewPackages:
			for _, it := range s.Items {
				formulaNamesSet[it.Name] = true
			}
		case resource.KindHomeBrewCasks:
			for _, it := range s.Items {
				caskNamesSet[it.Name] = true
			}
		}
	}

	formulaNames := make([]string, 0, len(formulaNamesSet))
	for n := range formulaNamesSet {
		formulaNames = append(formulaNames, n)
	}
	caskNames := make([]string, 0, len(caskNamesSet))
	for n := range caskNamesSet {
		caskNames = append(caskNames, n)
	}

	// Query brew only for the targeted set of names. If discovery fails, fall back to
	// the old behavior: treat all desired items as additions.
	installed, err := p.getInstalledPackagesFor(ctx, formulaNames, caskNames)
	// Add installed taps so compareGroupItems can detect them as in-sync.
	// Homebrew strips the "homebrew-" prefix from repo names in its output:
	// "brew tap stigoleg/homebrew-tap" registers as "stigoleg/tap".
	// Register both forms so lookups against user-specified names succeed.
	for _, t := range p.listInstalledTaps(ctx) {
		installed[t] = ""
		if parts := strings.SplitN(t, "/", 2); len(parts) == 2 {
			installed[parts[0]+"/homebrew-"+parts[1]] = ""
		}
	}
	if err != nil {
		slog.Warn("brew targeted discovery failed; falling back to additions-only", "err", err)
		for _, group := range desired {
			if resource.IsBrewKind(group.Kind) {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: group.Items,
				})
			}
		}
		return plan
	}

	// Process each desired group
	for _, group := range desired {
		if !resource.IsBrewKind(group.Kind) {
			continue
		}

		stateGroup, exists := stateIndex[group.Name]

		if !exists {
			// New group - check which items are already installed vs need installation
			var toInstall, toImport []resource.ResourceItem

			for _, item := range group.Items {
				name := item.Name
				// For casks we rely on the resource.Kind to indicate cask vs formula.
				lookupName := name
				if _, isInstalled := installed[lookupName]; isInstalled {
					// Already installed - needs to be imported
					toImport = append(toImport, item)
				} else {
					// Not installed - needs installation
					toInstall = append(toInstall, item)
				}
			}

			// Add items that need installation
			if len(toInstall) > 0 {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: toInstall,
				})
			}

			// Add items that are already installed (for import)
			if len(toImport) > 0 {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: toImport,
				})

				// Add warning about import
				itemNames := make([]string, 0, len(toImport))
				candidates := make([]provider.ImportCandidate, 0, len(toImport))
				for _, item := range toImport {
					itemNames = append(itemNames, item.Name)
					candidates = append(candidates, provider.ImportCandidate{
						ID: fmt.Sprintf("%s/%s[%s]", group.Kind, group.Name, item.Name),
					})
				}
				plan.Warnings = append(plan.Warnings, provider.PlanWarning{
					GroupID:     fmt.Sprintf("%s/%s", group.Kind, group.Name),
					Severity:    "warning",
					Message:     fmt.Sprintf("Items already installed but not tracked: %s", strings.Join(itemNames, ", ")),
					Suggestion:  fmt.Sprintf("nim state import %s/%s[<item>]", group.Kind, group.Name),
					ImportItems: candidates,
				})
			}
		} else {
			// Existing group - compare items
			additions, removals, cleanupItems, modifications, inSync := p.compareGroupItems(
				group, stateGroup, installed,
			)

			if len(additions) > 0 {
				plan.Additions = append(plan.Additions, provider.GroupAddition{
					Kind:  group.Kind,
					Group: group.Name,
					Items: additions,
				})
			}

			// Filter out already processed items and track new ones
			var filteredRemovals []resource.ResourceItem
			for _, item := range removals {
				key := fmt.Sprintf("%s/%s", group.Name, item.Name)
				if !processedItems[key] {
					processedItems[key] = true
					filteredRemovals = append(filteredRemovals, item)
				}
			}
			if len(filteredRemovals) > 0 {
				plan.Removals = append(plan.Removals, provider.GroupRemoval{
					Kind:  group.Kind,
					Group: group.Name,
					Items: filteredRemovals,
				})
				for _, it := range filteredRemovals {
					slog.Debug("plan.removal.append", "group", group.Name, "item", it.Name)
				}
			}

			var filteredCleanup []resource.ResourceItem
			for _, item := range cleanupItems {
				key := fmt.Sprintf("%s/%s", group.Name, item.Name)
				if !processedItems[key] {
					processedItems[key] = true
					filteredCleanup = append(filteredCleanup, item)
				}
			}
			if len(filteredCleanup) > 0 {
				plan.Cleanup = append(plan.Cleanup, provider.GroupCleanup{
					Kind:   group.Kind,
					Group:  group.Name,
					Items:  filteredCleanup,
					Reason: "not_in_config_and_not_installed",
				})
			}

			if len(modifications) > 0 {
				plan.Modifications = append(plan.Modifications, provider.GroupModification{
					Kind:    group.Kind,
					Group:   group.Name,
					Changes: modifications,
				})
			}

			if len(inSync) > 0 && len(additions) == 0 && len(removals) == 0 && len(cleanupItems) == 0 && len(modifications) == 0 {
				plan.InSync = append(plan.InSync, provider.GroupState{
					Kind:  group.Kind,
					Group: group.Name,
					Items: inSync,
				})
			}
		}
	}

	// Check for removals (groups in state but not in desired)
	desiredGroups := make(map[string]bool)
	for _, group := range desired {
		if resource.IsBrewKind(group.Kind) {
			desiredGroups[group.Name] = true
		}
	}

	for groupName, stateGroup := range stateIndex {
		if !desiredGroups[groupName] {
			// Tap groups are not tracked in the formula `installed` map — skip the
			// installation check entirely and treat all items as stale state cleanup.
			if stateGroup.Kind == resource.KindHomeBrewTaps {
				var tapCleanup []resource.ResourceItem
				for _, item := range stateGroup.Items {
					key := fmt.Sprintf("%s/%s", groupName, item.Name)
					if !processedItems[key] {
						tapCleanup = append(tapCleanup, resource.ResourceItem{Name: item.Name, Version: item.Version})
						processedItems[key] = true
					}
				}
				if len(tapCleanup) > 0 {
					plan.Cleanup = append(plan.Cleanup, provider.GroupCleanup{
						Kind:   stateGroup.Kind,
						Group:  groupName,
						Items:  tapCleanup,
						Reason: "not_in_config_and_not_installed",
					})
				}
				continue
			}

			// Entire group should be removed - but distinguish between
			// actual removals (installed) and cleanup (not installed)
			var removalItems []resource.ResourceItem
			var cleanupItems []resource.ResourceItem

			for _, item := range stateGroup.Items {
				lookupName := item.Name

				// Check if installed (handle tap packages with base name fallback)
				isInstalled := false
				if _, ok := installed[lookupName]; ok {
					isInstalled = true
				} else if strings.Contains(lookupName, "/") {
					baseName := path.Base(lookupName)
					if _, ok := installed[baseName]; ok {
						isInstalled = true
					}
				}

				key := fmt.Sprintf("%s/%s", groupName, item.Name)
				if processedItems[key] {
					// Skip already processed items
					continue
				}

				if isInstalled {
					removalItems = append(removalItems, resource.ResourceItem{
						Name:    item.Name,
						Version: item.Version,
					})
				} else {
					cleanupItems = append(cleanupItems, resource.ResourceItem{
						Name:    item.Name,
						Version: item.Version,
					})
				}
				processedItems[key] = true
			}

			if len(removalItems) > 0 {
				plan.Removals = append(plan.Removals, provider.GroupRemoval{
					Kind:  stateGroup.Kind,
					Group: groupName,
					Items: removalItems,
				})
				for _, it := range removalItems {
					slog.Debug("plan.removal.append.entire_group", "group", groupName, "item", it.Name)
				}
			}

			if len(cleanupItems) > 0 {
				plan.Cleanup = append(plan.Cleanup, provider.GroupCleanup{
					Kind:   stateGroup.Kind,
					Group:  groupName,
					Items:  cleanupItems,
					Reason: "not_in_config_and_not_installed",
				})
			}
		}
	}

	return plan
}

// compareGroupItems compares desired group items with state and installed packages.
// Returns additions, removals (installed items to uninstall), cleanup (not installed - state only),
// modifications, and inSync items.
func (p *BrewProvider) compareGroupItems(
	group resource.ResourceGroup[any],
	stateGroup provider.ResourceState,
	installed map[string]string,
) (additions, removals, cleanup []resource.ResourceItem, modifications []provider.ItemChange, inSync []resource.ItemState) {
	// Index state items by name
	stateItems := make(map[string]resource.ItemState)
	for _, item := range stateGroup.Items {
		stateItems[item.Name] = item
	}

	stateKeys := make([]string, 0, len(stateItems))
	for k := range stateItems {
		stateKeys = append(stateKeys, k)
	}
	installedKeys := make([]string, 0, len(installed))
	for k := range installed {
		installedKeys = append(installedKeys, k)
	}
	slog.Debug("compareGroupItems",
		"group", group.Name,
		"state_items", stateKeys,
		"installed", installedKeys,
	)

	// Check each desired item
	for _, desiredItem := range group.Items {
		name := desiredItem.Name
		lookupName := name

		// Try to find in installed - check full name first, then base name for tap packages
		isInstalled := false
		installedVersion := ""
		if v, ok := installed[lookupName]; ok {
			isInstalled = true
			installedVersion = v
		} else if strings.Contains(lookupName, "/") {
			// Try base name for tap packages (e.g., "dagger/tap/dagger" -> "dagger")
			baseName := path.Base(lookupName)
			if v, ok := installed[baseName]; ok {
				isInstalled = true
				installedVersion = v
			}
		}

		// Check state - full name and base name for tap packages
		stateItem, inState := stateItems[lookupName]
		if !inState && strings.Contains(lookupName, "/") {
			// Try base name for tap packages in state
			if si, ok := stateItems[path.Base(lookupName)]; ok {
				stateItem = si
				inState = true
			}
		}

		slog.Debug("evaluate.desired_item", "desired", name, "lookup", lookupName, "is_installed", isInstalled, "in_state", inState, "installed_version", installedVersion)

		if !isInstalled {
			// Not installed - needs to be added
			additions = append(additions, desiredItem)
		} else if inState {
			// Installed and tracked - check for modifications
			// Use installed version for comparison
			compareVersion := installedVersion
			if compareVersion == "" {
				compareVersion = stateItem.Version
			}
			if compareVersion != desiredItem.Version && desiredItem.Version != "" {
				modifications = append(modifications, provider.ItemChange{
					ItemName: lookupName,
					OldState: stateItem,
					NewState: resource.ItemState{
						Name:    lookupName,
						Version: desiredItem.Version,
						Status:  "present",
					},
					Diff: fmt.Sprintf("version: %s -> %s", compareVersion, desiredItem.Version),
				})
			} else {
				// In sync - update state with actual version
				stateItem.Version = installedVersion
				inSync = append(inSync, stateItem)
			}
		} else {
			// Installed but not in state - will be imported
			additions = append(additions, desiredItem)
		}
	}

	// Check for items in state but not in desired (removals or cleanup)
	desiredItems := make(map[string]bool)
	for _, item := range group.Items {
		desiredItems[item.Name] = true
	}

	desiredKeys := make([]string, 0, len(desiredItems))
	for k := range desiredItems {
		desiredKeys = append(desiredKeys, k)
	}
	slog.Debug("desiredItems", "group", group.Name, "items", desiredKeys)

	for name, stateItem := range stateItems {
		if !desiredItems[name] {
			// Check if installed to determine if removal or cleanup
			lookupName := name
			// Legacy code previously appended " (cask)" to state item names.
			// We no longer encode caskness in the name; rely on the group's Kind
			// to distinguish casks vs formulae. Keep lookupName as-is and
			// allow base-name fallback for tapped packages (handled below).
			lookupName = name

			isInstalled := false
			if _, ok := installed[lookupName]; ok {
				isInstalled = true
			} else if strings.Contains(lookupName, "/") {
				baseName := path.Base(lookupName)
				if _, ok := installed[baseName]; ok {
					isInstalled = true
				}
			}

			if isInstalled {
				removals = append(removals, resource.ResourceItem{
					Name:    name,
					Version: stateItem.Version,
				})
			} else {
				cleanup = append(cleanup, resource.ResourceItem{
					Name:    name,
					Version: stateItem.Version,
				})
			}
		}
	}

	return
}

// getInstalledPackages retrieves currently installed Homebrew packages
func (p *BrewProvider) getInstalledPackages(ctx context.Context) (map[string]string, error) {
	if ctx == nil {
		slog.Warn("brew getInstalledPackages called with nil context; returning empty set")
		return make(map[string]string), nil
	}
	packages := make(map[string]string)

	// Get formulae
	stdout, _, err := cmdutil.RunSimpleFn(ctx, "brew", "list", "--formula", "--versions")
	if err == nil {
		lines := strings.Split(stdout, "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				packages[parts[0]] = parts[1]
			}
		}
	}

	// Get casks (store plain names; cask vs formula is inferred from resource.Kind)
	stdout, _, err = cmdutil.RunSimpleFn(ctx, "brew", "list", "--cask", "--versions")
	if err == nil {
		lines := strings.Split(stdout, "\n")
		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				name := parts[0]
				version := ""
				if len(parts) >= 2 {
					version = parts[1]
				}
				packages[name] = version
			}
		}
	}

	return packages, nil
}

// getInstalledPackagesFor returns a map[name]version for the requested formulae and casks.
//
// Casks are detected via `brew list --cask --versions`, which reads installed files without
// loading API metadata — immune to the Homebrew `to_sym` API bug that plagues `brew info --cask`.
//
// Formulae are queried with `brew info --formula` (needed for alias resolution, e.g. kubectl →
// kubernetes-cli). Tap-qualified formula names (containing "/") are queried without a type flag
// because brew rejects --formula for that format.
func (p *BrewProvider) getInstalledPackagesFor(ctx context.Context, formulaNames, caskNames []string) (map[string]string, error) {
	packages := make(map[string]string)
	if ctx == nil {
		slog.Warn("brew getInstalledPackagesFor called with nil context; returning empty set")
		return packages, nil
	}

	// Casks: use brew list to avoid the API metadata bug.
	if len(caskNames) > 0 {
		p.mergeCasksFromList(ctx, caskNames, packages)
	}

	// Formulae: use brew info for alias resolution.
	// Split into simple names (use --formula flag) and tap-qualified (no flag).
	var simpleFormulae, tapFormulae []string
	for _, n := range formulaNames {
		if strings.Contains(n, "/") {
			tapFormulae = append(tapFormulae, n)
		} else {
			simpleFormulae = append(simpleFormulae, n)
		}
	}
	p.mergeFormulaeFromInfo(ctx, simpleFormulae, "--formula", packages)
	p.mergeFormulaeFromInfo(ctx, tapFormulae, "", packages)

	return packages, nil
}

// mergeCasksFromList uses `brew list --cask` + Caskroom directory reads to populate dst.
// `brew list --cask --versions` also loads API metadata (triggering a Homebrew Ruby bug),
// so versions are read directly from subdirectory names under Caskroom/<token>/.
func (p *BrewProvider) mergeCasksFromList(ctx context.Context, wantedCasks []string, dst map[string]string) {
	stdout, _, err := cmdutil.RunSimpleFn(ctx, "brew", "list", "--cask")
	if err != nil {
		slog.Warn("brew list --cask failed", "err", err)
		return
	}

	// Installed token set from brew list.
	installed := make(map[string]bool)
	for _, line := range strings.Split(stdout, "\n") {
		t := strings.TrimSpace(line)
		if t != "" {
			installed[strings.ToLower(t)] = true
		}
	}

	// Resolve brew prefix for Caskroom path.
	prefix, _, err := cmdutil.RunSimpleFn(ctx, "brew", "--prefix")
	if err != nil {
		prefix = "/opt/homebrew"
	}
	prefix = strings.TrimSpace(prefix)
	caskroomBase := prefix + "/Caskroom"

	wanted := make(map[string]bool, len(wantedCasks))
	for _, n := range wantedCasks {
		wanted[strings.ToLower(n)] = true
	}

	for _, n := range wantedCasks {
		token := strings.ToLower(n)
		if !installed[token] {
			continue
		}
		// Read version from Caskroom/<token>/<version>/ directory name.
		ver := p.caskroomVersion(caskroomBase, n)
		dst[n] = ver
		if token != n {
			dst[token] = ver
		}
	}
	_ = wanted
}

// caskroomVersion returns the installed version for a cask by reading its Caskroom directory.
func (p *BrewProvider) caskroomVersion(caskroomBase, token string) string {
	entries, err := os.ReadDir(caskroomBase + "/" + token)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() && e.Name() != ".metadata" {
			return e.Name()
		}
	}
	return ""
}

// mergeFormulaeFromInfo uses `brew info --json=v2 [flag] <names>` to populate dst.
// Retries individually if the batch fails.
func (p *BrewProvider) mergeFormulaeFromInfo(ctx context.Context, names []string, flag string, dst map[string]string) {
	if len(names) == 0 {
		return
	}
	parsed, err := p.brewInfoBatch(ctx, names, flag)
	if err != nil {
		slog.Warn("brew info batch failed; retrying individually", "flag", flag, "err", err)
		for _, name := range names {
			single, singleErr := p.brewInfoBatch(ctx, []string{name}, flag)
			if singleErr != nil {
				slog.Warn("brew info failed for formula; skipping", "name", name, "err", singleErr)
				continue
			}
			mergeBrewInfoIntoMap(single, dst)
		}
		return
	}
	mergeBrewInfoIntoMap(parsed, dst)
}

// mergeBrewInfoIntoMap copies formulae and cask entries from parsed brew info into dst.
func mergeBrewInfoIntoMap(parsed *brewInfoOutput, dst map[string]string) {
	for _, f := range parsed.Formulae {
		ver := f.InstalledVersion()
		if f.Name != "" {
			dst[f.Name] = ver
		}
		// Index by aliases too so user-facing names like "kubectl" resolve to
		// their canonical formula (e.g. "kubernetes-cli").
		for _, alias := range f.Aliases {
			if alias != "" {
				dst[alias] = ver
			}
		}
	}
	for _, c := range parsed.Casks {
		if c.Token != "" {
			dst[c.Token] = c.InstalledVersion()
		}
	}
}

// brewInfoBatch runs `brew info --json=v2 [flag] <names...>` and returns the parsed output.
// flag should be "--formula", "--cask", or "" (no type flag, for tap-qualified names).
func (p *BrewProvider) brewInfoBatch(ctx context.Context, names []string, flag string) (*brewInfoOutput, error) {
	base := []string{"info", "--json=v2"}
	if flag != "" {
		base = append(base, flag)
	}
	args := append(base, names...)
	stdout, stderr, err := cmdutil.RunSimpleFn(ctx, "brew", args...)
	if err != nil {
		return nil, fmt.Errorf("brew info failed: %s: %w", stderr, err)
	}
	var parsed brewInfoOutput
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse brew info json: %w", err)
	}
	return &parsed, nil
}

// Apply executes the given GroupPlan
func (p *BrewProvider) Apply(ctx context.Context, plan provider.GroupPlan) ([]provider.ApplyItemResult, error) {
	// Taps must be processed before formulae and casks so that tap-qualified
	// formulae (e.g. user/tap/formula) can be installed in the same apply run.
	sort.SliceStable(plan.Additions, func(i, j int) bool {
		return plan.Additions[i].Kind == resource.KindHomeBrewTaps &&
			plan.Additions[j].Kind != resource.KindHomeBrewTaps
	})

	var results []provider.ApplyItemResult

	for _, addition := range plan.Additions {
		results = append(results, p.applyGroupAddition(ctx, addition)...)
	}
	for _, removal := range plan.Removals {
		results = append(results, p.applyGroupRemoval(ctx, removal)...)
	}
	for _, modification := range plan.Modifications {
		results = append(results, p.applyGroupModification(ctx, modification)...)
	}

	return results, nil
}

// applyGroupAddition installs items in a group
func (p *BrewProvider) applyGroupAddition(ctx context.Context, addition provider.GroupAddition) []provider.ApplyItemResult {
	switch addition.Kind {
	case resource.KindHomeBrewTaps:
		// brew tap accepts one tap at a time; no batching possible.
		// item.Name may be "tap/name" or "tap/name https://url" (space-separated).
		// Stop on first tap failure — subsequent tap-qualified formulae would fail anyway.
		var results []provider.ApplyItemResult
		for _, item := range addition.Items {
			tapArgs := append([]string{"tap"}, strings.Fields(item.Name)...)
			_, stderr, err := cmdutil.RunSimpleFn(ctx, "brew", tapArgs...)
			r := provider.ApplyItemResult{Kind: addition.Kind, Group: addition.Group, Item: item.Name, Op: "add"}
			if err != nil {
				r.Err = fmt.Errorf("failed to tap %s: %s: %w", item.Name, stderr, err)
				results = append(results, r)
				// Mark remaining items as failed too
				for _, remaining := range addition.Items[len(results):] {
					results = append(results, provider.ApplyItemResult{
						Kind: addition.Kind, Group: addition.Group, Item: remaining.Name, Op: "add",
						Err: fmt.Errorf("skipped: previous tap failed"),
					})
				}
				return results
			}
			results = append(results, r)
		}
		return results

	case resource.KindHomeBrewCasks:
		// Skip casks that are already installed to avoid Homebrew API metadata bugs.
		installedCasks := p.listInstalledCasks(ctx)
		names := make([]string, 0, len(addition.Items))
		for _, item := range addition.Items {
			if !installedCasks[strings.ToLower(item.Name)] {
				names = append(names, item.Name)
			}
		}
		failed := batchWithFallback(names, func(ns []string) error {
			args := append([]string{"install", "--cask"}, ns...)
			_, stderr, err := cmdutil.RunSimpleFn(ctx, "brew", args...)
			if err != nil {
				if len(ns) == 1 {
					return fmt.Errorf("failed to install cask %s: %s: %w", ns[0], stderr, err)
				}
				return err
			}
			return nil
		})
		var results []provider.ApplyItemResult
		for _, item := range addition.Items {
			r := provider.ApplyItemResult{Kind: addition.Kind, Group: addition.Group, Item: item.Name, Op: "add"}
			if err, bad := failed[item.Name]; bad {
				r.Err = err
			}
			results = append(results, r)
		}
		return results

	default: // KindHomeBrewPackages
		names := make([]string, 0, len(addition.Items))
		for _, item := range addition.Items {
			names = append(names, item.Name)
		}
		failed := batchWithFallback(names, func(ns []string) error {
			args := append([]string{"install"}, ns...)
			_, stderr, err := cmdutil.RunSimpleFn(ctx, "brew", args...)
			if err != nil {
				if len(ns) == 1 {
					return fmt.Errorf("failed to install %s: %s: %w", ns[0], stderr, err)
				}
				return err
			}
			return nil
		})
		var results []provider.ApplyItemResult
		for _, item := range addition.Items {
			r := provider.ApplyItemResult{Kind: addition.Kind, Group: addition.Group, Item: item.Name, Op: "add"}
			if err, bad := failed[item.Name]; bad {
				r.Err = err
			}
			results = append(results, r)
		}
		return results
	}
}

// listInstalledCasks returns a lowercase-keyed set of installed cask tokens via `brew list --cask`.
// Does not load API metadata, so it is safe from the Homebrew `to_sym` Ruby bug.
func (p *BrewProvider) listInstalledCasks(ctx context.Context) map[string]bool {
	installed := make(map[string]bool)
	stdout, _, err := cmdutil.RunSimpleFn(ctx, "brew", "list", "--cask")
	if err != nil {
		slog.Warn("brew list --cask failed in listInstalledCasks", "err", err)
		return installed
	}
	for _, line := range strings.Split(stdout, "\n") {
		t := strings.TrimSpace(line)
		if t != "" {
			installed[strings.ToLower(t)] = true
		}
	}
	return installed
}

// listInstalledTaps returns the list of currently tapped repo names via `brew tap`.
func (p *BrewProvider) listInstalledTaps(ctx context.Context) []string {
	stdout, _, err := cmdutil.RunSimpleFn(ctx, "brew", "tap")
	if err != nil {
		slog.Warn("brew tap (list) failed", "err", err)
		return nil
	}
	var taps []string
	for _, line := range strings.Split(stdout, "\n") {
		t := strings.TrimSpace(line)
		if t != "" {
			taps = append(taps, t)
		}
	}
	return taps
}

// applyGroupRemoval uninstalls items from a group
func (p *BrewProvider) applyGroupRemoval(ctx context.Context, removal provider.GroupRemoval) []provider.ApplyItemResult {
	switch removal.Kind {
	case resource.KindHomeBrewTaps:
		// brew untap accepts one tap at a time; no batching possible.
		var results []provider.ApplyItemResult
		for _, item := range removal.Items {
			r := provider.ApplyItemResult{Kind: removal.Kind, Group: removal.Group, Item: item.Name, Op: "remove"}
			_, stderr, err := cmdutil.RunSimpleFn(ctx, "brew", "untap", item.Name)
			if err != nil {
				if strings.Contains(stderr, "No such tap") {
					slog.Warn("tap not present; skipping untap", "tap", item.Name)
				} else {
					r.Err = fmt.Errorf("failed to untap %s: %s: %w", item.Name, stderr, err)
				}
			}
			results = append(results, r)
		}
		return results

	case resource.KindHomeBrewCasks:
		names := make([]string, 0, len(removal.Items))
		for _, item := range removal.Items {
			names = append(names, item.Name)
		}
		failed := batchWithFallback(names, func(ns []string) error {
			args := append([]string{"uninstall", "--cask"}, ns...)
			_, stderr, err := cmdutil.RunSimpleFn(ctx, "brew", args...)
			if err != nil {
				if len(ns) == 1 {
					name := ns[0]
					if strings.Contains(stderr, "No available formula or cask with the name") || strings.Contains(stderr, "is not installed") {
						slog.Warn("cask not installed; skipping uninstall", "cask", name)
						return nil
					}
					if strings.Contains(stderr, "Refusing to uninstall") {
						return fmt.Errorf("failed to uninstall cask %s: %s", name, stderr)
					}
					return fmt.Errorf("failed to uninstall cask %s: %s: %w", name, stderr, err)
				}
				return err
			}
			return nil
		})
		var results []provider.ApplyItemResult
		for _, item := range removal.Items {
			r := provider.ApplyItemResult{Kind: removal.Kind, Group: removal.Group, Item: item.Name, Op: "remove"}
			if err, bad := failed[item.Name]; bad {
				r.Err = err
			}
			results = append(results, r)
		}
		return results

	default: // KindHomeBrewPackages
		names := make([]string, 0, len(removal.Items))
		for _, item := range removal.Items {
			names = append(names, item.Name)
		}
		failed := batchWithFallback(names, func(ns []string) error {
			args := append([]string{"uninstall"}, ns...)
			_, stderr, err := cmdutil.RunSimpleFn(ctx, "brew", args...)
			if err != nil {
				if len(ns) == 1 {
					name := ns[0]
					if strings.Contains(stderr, "No available formula or cask with the name") || strings.Contains(stderr, "is not installed") {
						slog.Warn("package not installed; skipping uninstall", "package", name)
						return nil
					}
					if strings.Contains(stderr, "Refusing to uninstall") {
						depsOut, _, _ := cmdutil.RunSimpleFn(ctx, "brew", "uses", "--installed", name)
						hint := strings.TrimSpace(depsOut)
						if hint != "" {
							stderr = stderr + "\nInstalled dependents:\n" + hint
						}
						return fmt.Errorf("failed to uninstall %s: %s", name, stderr)
					}
					return fmt.Errorf("failed to uninstall %s: %s: %w", name, stderr, err)
				}
				return err
			}
			return nil
		})
		var results []provider.ApplyItemResult
		for _, item := range removal.Items {
			r := provider.ApplyItemResult{Kind: removal.Kind, Group: removal.Group, Item: item.Name, Op: "remove"}
			if err, bad := failed[item.Name]; bad {
				r.Err = err
			}
			results = append(results, r)
		}
		return results
	}
}

// applyGroupModification updates items in a group
func (p *BrewProvider) applyGroupModification(ctx context.Context, modification provider.GroupModification) []provider.ApplyItemResult {
	var results []provider.ApplyItemResult
	for _, change := range modification.Changes {
		name := change.ItemName
		r := provider.ApplyItemResult{Kind: modification.Kind, Group: modification.Group, Item: name, Op: "modify"}
		if _, stderr, err := cmdutil.RunSimpleFn(ctx, "brew", "reinstall", name); err != nil {
			r.Err = fmt.Errorf("failed to update %s: %s: %w", name, stderr, err)
		}
		results = append(results, r)
	}
	return results
}

// Import performs discovery for the requested group. Historically provider-level
// import for Homebrew was used by the CLI to perform a best-effort discovery
// of installed packages. This implementation performs discovery and returns
// an error when discovery fails. Callers that previously relied on an empty
// fallback must explicitly handle errors and decide whether to append an
// item to state.
func (p *BrewProvider) Import(ctx context.Context, group string) (provider.ResourceState, error) {
	if ctx == nil {
		return provider.ResourceState{}, fmt.Errorf("nil context")
	}

	// Attempt to list installed packages as a minimal discovery.
	installed, err := p.getInstalledPackages(ctx)
	if err != nil {
		return provider.ResourceState{}, fmt.Errorf("failed to discover installed brew packages: %w", err)
	}

	items := make([]resource.ItemState, 0, len(installed))
	for name, version := range installed {
		items = append(items, resource.ItemState{Name: name, Version: version, Status: "present"})
	}

	// Return a group-level state. Default Kind is HomeBrewPackages to align
	// with historical behavior and test expectations.
	return provider.ResourceState{Kind: resource.KindHomeBrewPackages, Group: group, Items: items}, nil
}
