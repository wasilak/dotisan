package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/output"
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
		return runStateImport(id, actualValue)
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

func runStateImport(id, actualValue string) error {
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

	// Import the resource item
	ctx := context.Background()
	resourceState, err := p.ImportItem(ctx, group, actualValue)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Ensure kind is set
	resourceState.Kind = kind
	resourceState.Group = group

	// Load current state
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
		return runStateRemoveByID(id)
	},
}

var stateRemoveForce bool

func init() {
	stateRemoveCmd.Flags().BoolVarP(&stateRemoveForce, "force", "f", false, "Skip confirmation prompt")
}

func runStateRemoveByID(id string) error {
	kind, group, err := parseID(id)
	if err != nil {
		return err
	}

	if !stateRemoveForce {
		prompt := style.InfoBox.Render(
			fmt.Sprintf("Remove %s/%s from state?\n", kind, group) +
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

	if !currentState.RemoveResourceGroup(kind, group) {
		fmt.Printf("%s Resource %s/%s not found in state\n", style.IconError, kind, group)
		return nil
	}

	if err := backend.Save(ctx, currentState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Printf("%s Removed %s/%s from state\n", style.IconSuccess, kind, group)
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
		return runStateList()
	},
}

var stateOutputFlag string

func runStateList() error {
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
	headerStyle := lipgloss.NewStyle().Bold(true).Align(lipgloss.Center)
	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	t := table.New()
	t.Border(lipgloss.NormalBorder())
	t.BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(style.Blue)))
	t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == table.HeaderRow {
			return headerStyle
		}
		return cellStyle
	})
	t.Headers("KIND", "GROUP", "ITEMS", "STATUS")

	for _, res := range currentState.Resources {
		itemNames := make([]string, 0, len(res.Items))
		for _, item := range res.Items {
			itemNames = append(itemNames, item.Name)
		}
		t.Row(res.Kind, res.Group, strings.Join(itemNames, ", "), "managed")
	}

	fmt.Println(style.Header.Render("Managed Resources"))
	fmt.Println()
	fmt.Println(t.Render())
	fmt.Printf("\nTotal: %d resource groups\n", len(currentState.Resources))
	return nil
}

func init() {
	rootCmd.AddCommand(stateCmd)
	stateCmd.AddCommand(stateImportCmd)
	stateCmd.AddCommand(stateRemoveCmd)
	stateCmd.AddCommand(stateListCmd)
	stateListCmd.Flags().StringVarP(&stateOutputFlag, "output", "o", "", "Output format (plain, tree, json)")
}
