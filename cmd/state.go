package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/providers"
	"github.com/wasilak/dotisan/pkg/state"
	"github.com/wasilak/dotisan/pkg/style"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
)

// stateCmd represents the state command
var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "State management commands",
	Long: `state provides subcommands for managing the state file:

  import  - Bring existing resource under management
  remove  - Drop from state without touching system
  list    - Show all managed resources + status
  pull    - Fetch state from remote backend
  push    - Write local state to remote backend`,
}

// stateImportCmd imports an existing resource into state
var stateImportCmd = &cobra.Command{
	Use:          "import ID ACTUAL_VALUE",
	SilenceUsage: true,
	Short:        "Import existing resource into state",
	Long: `import discovers an existing resource on your system and adds it to
the state file without making any changes to the system.

Examples:
  dotisan state import ManagedFile/zshrc ~/.zshrc
  dotisan state import BrewPackages/core-tools[ripgrep] ripgrep`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		actualValue := args[1]
		return runStateImport(id, actualValue)
	},
}

// kindToProvider maps resource kind to provider name
func kindToProvider(kind string) string {
	switch strings.ToLower(kind) {
	case "managedfile", "manageddirectory":
		return "file"
	case "brewpackages", "homebrew":
		return "homebrew"
	case "npmpackages":
		return "npm"
	case "gopackages":
		return "go"
	case "cargopackages":
		return "cargo"
	default:
		return strings.ToLower(kind)
	}
}

var resourceRefRegex = regexp.MustCompile(`^([a-zA-Z0-9_-]+)(?:\[([^\]]+)\])?$`)

// parseResourceRef parses a resource reference that may contain an item key.
// Examples:
//   - "zshrc" -> name="zshrc", itemKey="", hasItemKey=false
//   - "core-tools[ripgrep]" -> name="core-tools", itemKey="ripgrep", hasItemKey=true
//   - "core-tools[0]" -> name="core-tools", itemKey="0", hasItemKey=true
func parseResourceRef(ref string) (name string, itemKey string, hasItemKey bool) {
	matches := resourceRefRegex.FindStringSubmatch(ref)
	if matches == nil {
		return ref, "", false
	}
	name = matches[1]
	itemKey = matches[2]
	hasItemKey = itemKey != ""
	return name, itemKey, hasItemKey
}

// ensureProvidersRegistered registers all providers if they haven't been registered yet.
// This is needed for commands that don't go through the Engine.
func ensureProvidersRegistered() {
	// Try to get each provider, and register if not found
	if _, err := provider.Get("file"); err != nil {
		provider.Register("file", providers.NewFileProvider(nil, nil, ""))
	}
	if _, err := provider.Get("homebrew"); err != nil {
		provider.Register("homebrew", providers.NewBrewProvider())
	}
	if _, err := provider.Get("npm"); err != nil {
		provider.Register("npm", providers.NewNpmProvider())
	}
	if _, err := provider.Get("go"); err != nil {
		provider.Register("go", providers.NewGoProvider())
	}
	if _, err := provider.Get("cargo"); err != nil {
		provider.Register("cargo", providers.NewCargoProvider())
	}
}

func runStateImport(id, actualValue string) error {
	// Parse ID to extract kind and name
	// ID format: Kind/name or Kind/name[itemKey]
	// Examples: ManagedFile/zshrc, BrewPackages/core-tools[ripgrep]
	kind, name, hasItemKey, err := parseID(id)
	if err != nil {
		return err
	}

	// Ensure providers are registered
	ensureProvidersRegistered()

	// Map kind to provider name
	providerName := kindToProvider(kind)

	// Get the provider
	p, err := provider.Get(providerName)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	// Check if provider is available
	available, msg := p.Available()
	if !available {
		return fmt.Errorf("provider %s is not available: %s", kind, msg)
	}

	// Import the resource
	ctx := context.Background()
	var resourceState provider.ResourceState
	if hasItemKey {
		// Use ImportItem for indexed resources
		resourceState, err = p.ImportItem(ctx, name, actualValue)
	} else {
		// Use regular Import for non-indexed resources
		resourceState, err = p.Import(ctx, actualValue)
	}
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Set the resource ID, name and kind
	resourceState.ID = id
	resourceState.Name = name
	resourceState.Kind = kind

	// Load current state - use configured path from config
	cfg, _ := config.LoadConfigFromDefaultPath()
	statePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg != nil && cfg.State.Path != "" {
		statePath = os.ExpandEnv(cfg.State.Path)
	}
	backend := state.NewLocalBackend(statePath)
	currentState, err := backend.Load(ctx)
	if err != nil {
		currentState = state.NewState()
	}

	// Check if resource already exists in state (Terraform behavior)
	if _, exists := currentState.GetResource(id); exists {
		fmt.Printf("%s Cannot import: resource already exists\n", style.IconError)
		fmt.Printf("  %s Use 'dotisan state remove %s' first\n", style.Dim.Render("→"), id)
		return nil
	}

	// Add the imported resource
	currentState.SetResource(resourceState)

	// Save state
	if err := backend.Save(ctx, currentState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Success message
	fmt.Printf("%s Successfully imported %s[%s]\n", style.IconSuccess, id, resourceState.ID)
	fmt.Printf("  %s Run 'dotisan state list' to view\n", style.Dim.Render("→"))

	return nil
}

// parseID parses a resource ID to extract kind, name, and whether it has an item key.
// ID format: Kind/name or Kind/name[itemKey]
// Examples:
//   - "ManagedFile/zshrc" -> kind="ManagedFile", name="zshrc", hasItemKey=false
//   - "BrewPackages/core-tools[ripgrep]" -> kind="BrewPackages", name="core-tools", hasItemKey=true
func parseID(id string) (kind, name string, hasItemKey bool, err error) {
	// First check if there's a bracket (item key)
	bracketIdx := strings.Index(id, "[")
	if bracketIdx == -1 {
		// No bracket - simple Kind/name format
		parts := strings.SplitN(id, "/", 2)
		if len(parts) != 2 {
			return "", "", false, fmt.Errorf("invalid ID format: %s (expected Kind/name)", id)
		}
		return parts[0], parts[1], false, nil
	}

	// Has bracket - extract kind/name from part before bracket
	prefix := id[:bracketIdx]
	parts := strings.SplitN(prefix, "/", 2)
	if len(parts) != 2 {
		return "", "", false, fmt.Errorf("invalid ID format: %s", id)
	}

	return parts[0], parts[1], true, nil
}

// stateRemoveCmd removes a resource from state
var stateRemoveCmd = &cobra.Command{
	Use:          "remove ID",
	SilenceUsage: true,
	Short:        "Remove resource from state only",
	Long: `remove deletes the resource entry from the state file without
affecting the actual system. Use this when you want dotisan to stop
tracking a resource without removing it from your system.

Use --force to skip confirmation prompts.

Example: dotisan state remove BrewPackages/core-tools[ripgrep]`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		return runStateRemoveByID(id)
	},
}

var stateRemoveForce bool

func init() {
	stateRemoveCmd.Flags().BoolVarP(&stateRemoveForce, "force", "f", false, "Skip confirmation prompt")
}

func runStateRemoveByID(id string) error {
	// Ask for confirmation if --force is not set
	if !stateRemoveForce {
		prompt := style.InfoBox.Render(
			fmt.Sprintf("Remove %s from state?\n", id) +
				style.Dim.Render("  (actual resource will not be modified)\n\n") +
				fmt.Sprintf("%s %s, remove from state\n", style.Info.Render("[Y]"), style.Dim.Render("Yes")) +
				fmt.Sprintf("%s %s, keep it\n", style.Info.Render("[N]"), style.Dim.Render("No")),
		)
		fmt.Print(prompt)

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Printf("%s Cancelled.\n", style.Warning.Render("→"))
			return nil
		}
	}

	// Load current state - use configured path from config
	ctx := context.Background()
	cfg, _ := config.LoadConfigFromDefaultPath()
	statePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg != nil && cfg.State.Path != "" {
		statePath = os.ExpandEnv(cfg.State.Path)
	}
	backend := state.NewLocalBackend(statePath)
	currentState, err := backend.Load(ctx)
	if err != nil {
		return fmt.Errorf("cannot load state: %w", err)
	}

	// Find and remove the resource by ID
	found := false
	for i, r := range currentState.Resources {
		if r.ID == id {
			// Remove by swapping with last and truncating
			currentState.Resources[i] = currentState.Resources[len(currentState.Resources)-1]
			currentState.Resources = currentState.Resources[:len(currentState.Resources)-1]
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("%s Resource %s not found in state\n", style.IconError, id)
		return nil
	}

	// Save state
	if err := backend.Save(ctx, currentState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Success message
	fmt.Printf("%s Removed %s from state\n", style.IconSuccess, id)
	fmt.Println(style.Dim.Render("Note: The actual resource was not modified on your system."))

	return nil
}

// stateListCmd lists all managed resources
var stateListCmd = &cobra.Command{
	Use:          "list",
	SilenceUsage: true,
	Short:        "List all managed resources",
	Long: `list displays all resources currently tracked in the state file
along with their status (in_sync, drift, missing).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStateList()
	},
}

var stateTreeFlag bool

func runStateList() error {
	// Create engine to run plan (for accurate status)
	eng, err := engine.NewEngine()
	if err != nil {
		// Fallback to basic list if engine fails
		return runStateListBasic()
	}

	// Run plan to get actual status
	ctx := context.Background()
	result, err := eng.Plan(ctx, nil)
	if err != nil {
		return runStateListBasic()
	}

	// Load state for resource list - use configured path from config
	cfg, err := config.LoadConfigFromDefaultPath()
	statePath := ""
	if err == nil && cfg.State.Path != "" {
		statePath = os.ExpandEnv(cfg.State.Path)
	}
	if statePath == "" {
		statePath = os.ExpandEnv("$HOME/.config/dotisan/state.json")
	}
	backend := state.NewLocalBackend(statePath)
	currentState, err := backend.Load(ctx)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No state file found. Run 'dotisan apply' first.")
			return nil
		}
		return fmt.Errorf("cannot load state: %w", err)
	}

	if len(currentState.Resources) == 0 {
		fmt.Println("No managed resources found.")
		return nil
	}

	// Build status map from plan result
	statusMap := buildStatusMap(result)

	// Use shared styles
	inSyncStyle := style.Success
	driftStyle := style.Warning
	missingStyle := style.Error
	unknownStyle := style.Dim

    // Define styles for header and rows
    headerStyle := lipgloss.NewStyle().Bold(true).Align(lipgloss.Center)
    cellStyle := lipgloss.NewStyle().Padding(0, 1)
    oddRowStyle := cellStyle.Foreground(lipgloss.Color(style.Gray))
    evenRowStyle := cellStyle

    t := table.New()
    t.Border(lipgloss.NormalBorder())
    t.BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(style.Blue)))
    t.StyleFunc(func(row, col int) lipgloss.Style {
        switch {
        case row == table.HeaderRow:
            return headerStyle
        case row%2 == 0:
            return evenRowStyle
        default:
            return oddRowStyle
        }
    })
    t.Headers("KIND", "NAME", "ID", "STATUS")

	// Track counts for summary and build resources list
	inSyncCount := 0
	driftCount := 0
	orphanCount := 0
	var stateResources []diff.StateResource

    // Display resources with accurate status
    for _, r := range currentState.Resources {
        status, _ := getResourceStatus(r, statusMap, inSyncStyle, driftStyle, missingStyle, unknownStyle)

        // Count statuses
        switch status {
        case "in_sync":
            inSyncCount++
        case "drift", "modified":
            driftCount++
        case "orphaned", "pending":
            orphanCount++
        }

		// Build tree data
		stateResources = append(stateResources, diff.StateResource{
			Kind:   r.Kind,
			Name:   r.Name,
			ID:     r.ID,
			Status: status,
		})

        // Add row to table. We'll apply row styling via the table StyleFunc.
        t.Row(truncate(r.Kind, 17), truncate(r.Name, 22), truncate(r.ID, 32), status)
    }

	// Render as tree if flag is set OR if configured in config
	useTree := stateTreeFlag || eng.Config.UI.Tree
	if useTree {
		treeFormatter := diff.NewTreeFormatter()
		fmt.Println(treeFormatter.FormatStateAsTree(stateResources))
	} else {
		// Print header and table
		fmt.Println(style.Header.Render("Managed Resources"))
		fmt.Println()
		fmt.Println(t.Render())
	}

	// Summary footer
	fmt.Println()
	if inSyncCount > 0 {
		fmt.Printf("%s %d in sync\n", style.IconSuccess, inSyncCount)
	}
	if driftCount > 0 {
		fmt.Printf("%s %d warnings\n", style.RowWarning.Render("⚠"), driftCount)
	}
	if orphanCount > 0 {
		fmt.Printf("%s %d orphaned\n", style.RowWarning.Render("⊘"), orphanCount)
	}
	if inSyncCount+driftCount+orphanCount == 0 {
		fmt.Println("No managed resources.")
	} else {
		fmt.Printf("\nTotal: %d resources\n", len(currentState.Resources))
	}

	return nil
}

func styledInline(row string, status string, st lipgloss.Style) string {
	switch status {
	case "in_sync":
		return style.RowSuccess.Render(row)
	case "drift", "modified", "pending", "orphaned":
		return style.RowWarning.Render(row)
	}
	return st.Render(status)
}

// Helper to truncate strings
func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

// buildStatusMap creates a map of resource ID to status from plan result
func buildStatusMap(result *engine.PlanResult) map[string]string {
	statusMap := make(map[string]string)

	for _, plan := range result.ProviderPlans {
		// In sync resources
		for _, res := range plan.InSync {
			id := fmt.Sprintf("%s/%s", res.GetKind(), res.GetMetadata().Name)
			statusMap[id] = "in_sync"
		}
		// Additions (not in state yet)
		for _, res := range plan.Additions {
			id := fmt.Sprintf("%s/%s", res.GetKind(), res.GetMetadata().Name)
			statusMap[id] = "pending"
		}
		// Modifications
		for _, mod := range plan.Modifications {
			id := fmt.Sprintf("%s/%s", mod.Resource.GetKind(), mod.Resource.GetMetadata().Name)
			statusMap[id] = "modified"
		}
		// Removals (orphaned)
		for _, res := range plan.Removals {
			id := fmt.Sprintf("%s/%s", res.GetKind(), res.GetMetadata().Name)
			statusMap[id] = "orphaned"
		}
		// Drifted
		for _, drift := range plan.Drifted {
			id := fmt.Sprintf("%s/%s", drift.Resource.GetKind(), drift.Resource.GetMetadata().Name)
			statusMap[id] = "drift"
		}
	}

	return statusMap
}

// getResourceStatus returns the status and style for a resource
func getResourceStatus(r provider.ResourceState, statusMap map[string]string,
	inSyncStyle, driftStyle, missingStyle, unknownStyle lipgloss.Style) (string, lipgloss.Style) {

	// Try full ID first
	status, exists := statusMap[r.ID]
	if !exists {
		// Try parent ID (without item key) - for indexed resources like BrewPackages/core-tools[ripgrep]
		parentID := fmt.Sprintf("%s/%s", r.Kind, r.Name)
		status, exists = statusMap[parentID]
	}
	if !exists {
		// Resource in state but not in config (orphaned)
		return "orphaned", missingStyle
	}

	switch status {
	case "in_sync":
		return "in_sync", inSyncStyle
	case "drift":
		return "drift", driftStyle
	case "modified":
		return "modified", driftStyle
	case "orphaned", "pending":
		return status, missingStyle
	default:
		return status, unknownStyle
	}
}

// runStateListBasic is a fallback that just lists resources without status check
func runStateListBasic() error {
	ctx := context.Background()
	cfg, _ := config.LoadConfigFromDefaultPath()
	statePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg != nil && cfg.State.Path != "" {
		statePath = os.ExpandEnv(cfg.State.Path)
	}
	backend := state.NewLocalBackend(statePath)
	currentState, err := backend.Load(ctx)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No state file found. Run 'dotisan apply' first.")
			return nil
		}
		return fmt.Errorf("cannot load state: %w", err)
	}

	if len(currentState.Resources) == 0 {
		fmt.Println("No managed resources found.")
		return nil
	}

	fmt.Println(style.Header.Render("Managed Resources"))
	fmt.Println()
	fmt.Printf("%-20s %-25s %-30s\n", "KIND", "NAME", "ID")
	fmt.Println(strings.Repeat("-", 75))

	for _, r := range currentState.Resources {
		fmt.Printf("%-20s %-25s %-30s\n",
			truncate(r.Kind, 20),
			truncate(r.Name, 25),
			truncate(r.ID, 30))
	}

	fmt.Println()
	fmt.Printf("Total: %d resources\n", len(currentState.Resources))
	return nil
}

// statePullCmd pulls state from remote backend
var statePullCmd = &cobra.Command{
	Use:          "pull",
	SilenceUsage: true,
	Short:        "Fetch state from remote backend",
	Long: `pull downloads the state from a configured remote backend (S3)
and overwrites the local state file. Use with caution.

Note: This requires S3 backend configuration in config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatePull()
	},
}

func runStatePull() error {
	fmt.Printf("%s Pulling state from remote...\n", style.Info.Render("→"))

	// Load config to check for S3 backend
	configPath := os.ExpandEnv("$HOME/.dotisan/config.yaml")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	// Check if S3 backend is configured
	if cfg.State.Backend != "s3" {
		return fmt.Errorf("S3 backend not configured in config.yaml (current: %s)", cfg.State.Backend)
	}

	// Create S3 backend
	ctx := context.Background()
	backend, err := state.NewS3Backend(state.S3Config{
		Endpoint:        cfg.State.S3.Endpoint,
		Bucket:          cfg.State.S3.Bucket,
		Key:             cfg.State.S3.Key,
		Region:          cfg.State.S3.Region,
		AccessKeyID:     cfg.State.S3.AccessKeyID,
		SecretAccessKey: cfg.State.S3.SecretAccessKey,
		UseSSL:          true, // Default to SSL
	})
	if err != nil {
		return fmt.Errorf("failed to initialize S3 backend: %w", err)
	}

	// Load remote state
	remoteState, err := backend.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load remote state: %w", err)
	}

	// Save to local backend - use configured path
	localStatePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg.State.Path != "" {
		localStatePath = os.ExpandEnv(cfg.State.Path)
	}
	localBackend := state.NewLocalBackend(localStatePath)
	if err := localBackend.Save(ctx, remoteState); err != nil {
		return fmt.Errorf("failed to save local state: %w", err)
	}

	fmt.Printf("%s Pulled state from remote (%d resources)\n",
		style.IconSuccess, len(remoteState.Resources))

	return nil
}

// statePushCmd pushes state to remote backend
var statePushCmd = &cobra.Command{
	Use:          "push",
	SilenceUsage: true,
	Short:        "Write local state to remote backend",
	Long: `push uploads the local state file to a configured remote backend (S3),
overwriting the remote state. Use with caution.

Note: This requires S3 backend configuration in config.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatePush()
	},
}

func runStatePush() error {
	fmt.Printf("%s Pushing state to remote...\n", style.Info.Render("→"))

	// Load config to check for S3 backend
	configPath := os.ExpandEnv("$HOME/.dotisan/config.yaml")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("cannot load config: %w", err)
	}

	// Check if S3 backend is configured
	if cfg.State.Backend != "s3" {
		return fmt.Errorf("S3 backend not configured in config.yaml (current: %s)", cfg.State.Backend)
	}

	// Load local state - use configured path
	ctx := context.Background()
	localStatePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg.State.Path != "" {
		localStatePath = os.ExpandEnv(cfg.State.Path)
	}
	localBackend := state.NewLocalBackend(localStatePath)
	localState, err := localBackend.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load local state: %w", err)
	}

	// Create S3 backend and save
	backend, err := state.NewS3Backend(state.S3Config{
		Endpoint:        cfg.State.S3.Endpoint,
		Bucket:          cfg.State.S3.Bucket,
		Key:             cfg.State.S3.Key,
		Region:          cfg.State.S3.Region,
		AccessKeyID:     cfg.State.S3.AccessKeyID,
		SecretAccessKey: cfg.State.S3.SecretAccessKey,
		UseSSL:          true, // Default to SSL
	})
	if err != nil {
		return fmt.Errorf("failed to initialize S3 backend: %w", err)
	}

	if err := backend.Save(ctx, localState); err != nil {
		return fmt.Errorf("failed to push state: %w", err)
	}

	fmt.Printf("%s Pushed state to remote (%d resources)\n",
		style.IconSuccess, len(localState.Resources))

	return nil
}

func init() {
	rootCmd.AddCommand(stateCmd)

	// Add subcommands
	stateCmd.AddCommand(stateImportCmd)
	stateCmd.AddCommand(stateRemoveCmd)
	stateCmd.AddCommand(stateListCmd)
	stateCmd.AddCommand(statePullCmd)
	stateCmd.AddCommand(statePushCmd)

	// Add flags
	stateListCmd.Flags().BoolVar(&stateTreeFlag, "tree", false, "Render output as tree structure")
}
