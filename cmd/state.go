package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/wasilak/dotisan/pkg/ui"
	"golang.org/x/term"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/providers"
	"github.com/wasilak/dotisan/pkg/state"
	"github.com/wasilak/dotisan/pkg/style"

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
	Use:          "import KIND/GROUP ITEM",
	SilenceUsage: true,
	Short:        "Import existing resource into state",
	Long: `import discovers an existing resource on your system and adds it to
the state file without making any changes to the system.

Examples:
  dotisan state import BrewPackages/core-tools ripgrep
  dotisan state import ManagedFile/zshrc ~/.zshrc`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		actualValue := args[1]
		return runStateImport(cmd.Context(), id, actualValue)
	},
}

// kindToProvider maps resource kind to provider name
func kindToProvider(kind string) string {
	switch strings.ToLower(kind) {
	case "managedfile", "manageddirectory":
		return "file"
	case "brewpackages":
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

// parseID parses a resource ID to extract kind and group
// ID format: Kind/group
// Examples: ManagedFile/zshrc, BrewPackages/core-tools
func parseID(id string) (kind, group string, err error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ID format: %s (expected Kind/group)", id)
	}
	return parts[0], parts[1], nil
}

func runStateImport(ctx context.Context, id, actualValue string) error {
	kind, group, err := parseID(id)
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

	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
	resourceState, err := p.ImportItem(ctx, group, actualValue)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Ensure kind is set
	resourceState.Kind = kind
	resourceState.Group = group

	// Load current state
	cfg, cfgErr := config.LoadConfigFromDefaultPath()
	if cfgErr != nil && !errors.Is(cfgErr, fs.ErrNotExist) {
		slog.Warn("failed to load config", "err", cfgErr)
	}
	statePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg != nil && cfg.State.Path != "" {
		statePath = os.ExpandEnv(cfg.State.Path)
	}
	backend := state.NewLocalBackend(statePath)
	currentState, err := backend.Load(ctx)
	if err != nil {
		currentState = state.NewState()
	}

	// Check if resource group already exists
	if _, exists := currentState.GetResourceGroup(kind, group); exists {
		// Append to existing group
		existing, _ := currentState.GetResourceGroup(kind, group)
		existing.Items = append(existing.Items, resourceState.Items...)
		currentState.SetResourceGroup(existing)
	} else {
		currentState.SetResourceGroup(resourceState)
	}

	// Save state
	if err := backend.Save(ctx, currentState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Printf("%s Successfully imported %s/%s: %s\n", style.IconSuccess, kind, group, actualValue)
	return nil
}

// ensureProvidersRegistered registers all providers
func ensureProvidersRegistered() {
	if _, err := provider.Get("file"); err != nil {
		provider.Register("file", providers.NewFileProvider(""))
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

// stateMvCmd moves an item between resource groups in state
var stateMvCmd = &cobra.Command{
	Use:          "mv SOURCE DESTINATION",
	SilenceUsage: true,
	Short:        "Move an item between resource groups in state",
	Long: `mv moves an item from one resource group to another in state only.
The actual system resource is not modified.

Source and destination format: Kind/Group/Item or Kind/Group
If destination item name is not provided, the source item name is used.

The destination group must exist in the desired configuration.

Examples:
  dotisan state mv BrewPackages/core-tools/ripgrep BrewPackages/homebrew-packages/ripgrep
  dotisan state mv BrewPackages/core-tools/ripgrep BrewPackages/homebrew-packages/`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStateMv(cmd.Context(), args[0], args[1])
	},
}

func runStateMv(ctx context.Context, source, destination string) error {
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}

	// Create engine
	eng, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Run state mv
	result, err := eng.StateMv(ctx, engine.StateMvOptions{
		Source:      source,
		Destination: destination,
	})
	if err != nil {
		fmt.Println()
		fmt.Println(style.Error.Render("✖ Move failed"))
		fmt.Println()
		fmt.Printf("  %s\n", err)
		fmt.Println()
		return fmt.Errorf("state mv failed")
	}

	// Display result
	engine.DisplayStateMvResult(result)
	return nil
}

// stateRemoveCmd removes a resource from state
var stateRemoveCmd = &cobra.Command{
	Use:          "remove KIND/GROUP",
	SilenceUsage: true,
	Short:        "Remove resource group from state",
	Long: `remove deletes the resource group entry from the state file without
affecting the actual system.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		return runStateRemoveByID(cmd.Context(), id)
	},
}

var stateRemoveForce bool

func init() {
	stateRemoveCmd.Flags().BoolVarP(&stateRemoveForce, "force", "f", false, "Skip confirmation prompt")
}

func runStateRemoveByID(ctx context.Context, id string) error {
	parts := strings.Split(id, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid ID format: %s (expected Kind/group or Kind/group/item)", id)
	}

	kind := parts[0]
	group := parts[1]
	item := ""
	if len(parts) >= 3 {
		item = parts[2]
	}

	if !stateRemoveForce {
		title := ""
		if item == "" {
			title = fmt.Sprintf("Remove %s/%s from state?", kind, group)
		} else {
			title = fmt.Sprintf("Remove %s/%s/%s from state?", kind, group, item)
		}
		hint := "(actual resource will not be modified)"

		// Robust confirmation: TTY vs fallback
		isTTY := term.IsTerminal(int(os.Stdout.Fd()))
		var confirm bool
		if isTTY {
			err := huh.NewConfirm().
				Title(title).Affirmative("Yes, remove from state").Negative("No, cancel").Value(&confirm).Run()
			if err != nil {
				return fmt.Errorf("confirmation prompt error: %w", err)
			}
		} else {
			fmt.Printf("%s %s [y/N]: ", title, hint)
			var resp string
			_, err := fmt.Fscanln(os.Stdin, &resp)
			if err != nil && err.Error() != "unexpected newline" {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}
			resp = strings.TrimSpace(strings.ToLower(resp))
			confirm = (resp == "y" || resp == "yes")
		}
		if !confirm {
			fmt.Println()
			fmt.Println(style.Info.Render("→ Remove cancelled."))
			return nil
		}
	}

	cfg, cfgErr := config.LoadConfigFromDefaultPath()
	if cfgErr != nil && !errors.Is(cfgErr, fs.ErrNotExist) {
		slog.Warn("failed to load config", "err", cfgErr)
	}
	statePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg != nil && cfg.State.Path != "" {
		statePath = os.ExpandEnv(cfg.State.Path)
	}
	backend := state.NewLocalBackend(statePath)
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
	currentState, err := backend.Load(ctx)
	if err != nil {
		return fmt.Errorf("cannot load state: %w", err)
	}

	var removed bool
	if item == "" {
		removed = currentState.RemoveResourceGroup(kind, group)
	} else {
		removed = currentState.RemoveResourceItem(kind, group, item)
	}

	if !removed {
		if item == "" {
			fmt.Printf("%s Resource %s/%s not found in state\n", style.IconError, kind, group)
		} else {
			fmt.Printf("%s Resource %s/%s/%s not found in state\n", style.IconError, kind, group, item)
		}
		return nil
	}

	if err := backend.Save(ctx, currentState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	if item == "" {
		fmt.Printf("%s Removed %s/%s from state\n", style.IconSuccess, kind, group)
	} else {
		fmt.Printf("%s Removed %s/%s/%s from state\n", style.IconSuccess, kind, group, item)
	}
	return nil
}

// stateListCmd lists all managed resources
var stateListCmd = &cobra.Command{
	Use:          "list",
	SilenceUsage: true,
	Short:        "List all managed resources",
	Long: `list displays all resources currently tracked in the state file
along with their status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStateList(cmd.Context())
	},
}

var stateOutputFlag string

func runStateList(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
	cfg, cfgErr := config.LoadConfigFromDefaultPath()
	if cfgErr != nil && !errors.Is(cfgErr, fs.ErrNotExist) {
		slog.Warn("failed to load config", "err", cfgErr)
	}
	statePath := os.ExpandEnv("$HOME/.config/dotisan/state.json")
	if cfg != nil && cfg.State.Path != "" {
		statePath = os.ExpandEnv(cfg.State.Path)
	}
	backend := state.NewLocalBackend(statePath)
	var currentState *state.State
	var loadErr error
	// Use provided context so signal cancellation propagates to spinner and backend.Load
	err := style.RunWithSpinner(ctx, "Loading state...", func(ctx context.Context) error {
		currentState, loadErr = backend.Load(ctx)
		return loadErr
	})
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

	// Determine output format
	outputFormat := output.Format(stateOutputFlag)
	if outputFormat == "" {
		outputFormat = output.FormatPlain
	}

	switch outputFormat {
	case output.FormatTree:
		return displayStateTree(currentState)
	case output.FormatJSON:
		return displayStateJSON(currentState)
	default:
		return displayStateTable(currentState)
	}
}

func displayStateTree(currentState *state.State) error {
	var resources []diff.StateResource
	for _, res := range currentState.Resources {
		items := make([]string, 0, len(res.Items))
		for _, item := range res.Items {
			items = append(items, item.Name)
		}
		resources = append(resources, diff.StateResource{
			Kind:   res.Kind,
			Group:  res.Group,
			Items:  items,
			Status: "managed",
		})
	}

	treeFormatter := diff.NewTreeFormatter()
	fmt.Println(treeFormatter.FormatStateAsTree(resources))
	return nil
}

func displayStateJSON(currentState *state.State) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(currentState)
}

func displayStateTable(currentState *state.State) error {
	fmt.Println(style.Header.Render("Managed Resources"))
	fmt.Println()

	// Use the unified Bubbletea Table for state
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 120
	}
	// Convert currentState (typed) to []ui.ResourceRow explicitly to avoid reflection
	rows := make([]ui.ResourceRow, 0)
	for _, res := range currentState.Resources {
		kind := res.Kind
		group := res.Group
		for _, it := range res.Items {
			status := it.Status
			if status == "" {
				status = "sync"
			}
			idParts := []string{}
			if kind != "" {
				idParts = append(idParts, kind)
			}
			if group != "" {
				idParts = append(idParts, group)
			}
			if it.Name != "" {
				idParts = append(idParts, it.Name)
			}
			id := strings.Join(idParts, "/")
			rows = append(rows, ui.ResourceRow{
				Status: status,
				ID:     id,
				Kind:   kind,
				Group:  group,
				Name:   it.Name,
				Info:   it.Version,
			})
		}
	}
	fmt.Println(ui.RenderResourceTable(width, rows, true))

	totalItems := 0
	for _, res := range currentState.Resources {
		totalItems += len(res.Items)
	}
	fmt.Printf("\nTotal: %d resources across %d groups\n", totalItems, len(currentState.Resources))
	return nil
}

func init() {
	rootCmd.AddCommand(stateCmd)
	stateCmd.AddCommand(stateImportCmd)
	stateCmd.AddCommand(stateMvCmd)
	stateCmd.AddCommand(stateRemoveCmd)
	stateCmd.AddCommand(stateListCmd)
	stateListCmd.Flags().StringVarP(&stateOutputFlag, "output", "o", "", "Output format (plain, tree, json)")
}
