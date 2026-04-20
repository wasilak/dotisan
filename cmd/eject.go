package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/wasilak/dotisan/pkg/state"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var ejectForceFlag bool

// ejectCmd represents the eject command
var ejectCmd = &cobra.Command{
	Use:   "eject KIND NAME",
	Short: "Stop managing a resource",
	Long: `eject removes a resource from state. The system is left untouched.
This is the opposite of import.

Usage: dotisan eject KIND NAME

This will:
  1. Remove the resource from the state file
  2. Leave the actual resource untouched on your system
  3. You should manually remove its definition from config files

Use --force to skip confirmation prompts.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind := args[0]
		name := args[1]
		return runEject(kind, name)
	},
}

func runEject(kind, name string) error {
	// Define lipgloss styles
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	boldStyle := lipgloss.NewStyle().Bold(true)

	fmt.Printf("Ejecting %s/%s...\n\n", kind, name)

	// 1. Load current state
	ctx := context.Background()
	dotisanDir := os.ExpandEnv("$HOME/.dotisan")
	statePath := dotisanDir + "/state.json"
	backend := state.NewLocalBackend(statePath)
	currentState, err := backend.Load(ctx)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot load state: %w", err)
		}
		currentState = state.NewState()
	}

	// 2. Find the resource
	found := false
	resourceID := ""
	for _, r := range currentState.Resources {
		if r.Kind == kind && r.Name == name {
			found = true
			resourceID = r.ID
			break
		}
	}

	if !found {
		fmt.Printf("%s Resource %s/%s not found in state\n", yellowStyle.Render("⚠"), kind, name)
		fmt.Println("It may not have been managed by dotisan, or was already ejected.")
		return nil
	}

	fmt.Printf("%s Found in state (ID: %s)\n", greenStyle.Render("✓"), resourceID)

	// 3. Confirm unless --force
	if !ejectForceFlag {
		fmt.Printf("\nThis will remove %s/%s from dotisan's state.\n", kind, name)
		fmt.Println("The actual resource on your system will NOT be modified.")
		fmt.Print("\nProceed? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Ejection cancelled.")
			return nil
		}
	}

	// 4. Remove from state
	for i, r := range currentState.Resources {
		if r.Kind == kind && r.Name == name {
			// Remove by swapping with last and truncating
			currentState.Resources[i] = currentState.Resources[len(currentState.Resources)-1]
			currentState.Resources = currentState.Resources[:len(currentState.Resources)-1]
			break
		}
	}

	// 5. Save state
	if err := backend.Save(ctx, currentState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	fmt.Printf("%s Removed from state\n", greenStyle.Render("✓"))

	// 6. Check for config file and warn
	dotfilesDir := os.ExpandEnv("$HOME/.config/dotisan")
	files, err := os.ReadDir(dotfilesDir)
	if err == nil {
		for _, file := range files {
			if file.IsDir() || (!strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml")) {
				continue
			}

			path := dotfilesDir + "/" + file.Name()
			content, err := os.ReadFile(path)
			if err != nil {
				continue
			}

			// Simple check: look for "kind: KIND" and "name: NAME" in the file
			contentStr := string(content)
			if strings.Contains(contentStr, fmt.Sprintf("kind: %s", kind)) &&
				strings.Contains(contentStr, fmt.Sprintf("name: %s", name)) {
				fmt.Printf("\n%s Note: This resource is still defined in:\n", yellowStyle.Render("⚠"))
				fmt.Printf("   %s\n", boldStyle.Render(path))
				fmt.Println("   You may want to manually remove it from this file.")
				break
			}
		}
	}

	fmt.Println()
	fmt.Printf("%s Ejection complete.\n", greenStyle.Render("✓"))
	fmt.Println("The resource was removed from dotisan's state but remains on your system.")

	return nil
}

func init() {
	rootCmd.AddCommand(ejectCmd)
	ejectCmd.Flags().BoolVar(&ejectForceFlag, "force", false, "Skip confirmation prompts")
}
