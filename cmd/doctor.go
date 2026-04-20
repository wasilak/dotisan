package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/state"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system prerequisites",
	Long: `doctor checks each provider's Available() status, state backend connectivity,
config file validity, and template rendering. Reports issues and suggests fixes.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func runDoctor() error {
	// Define lipgloss styles
	greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	yellowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	headerStyle := lipgloss.NewStyle().Bold(true).Underline(true)

	hasErrors := false
	issues := []string{}
	warnings := []string{}

	fmt.Println(headerStyle.Render("dotisan doctor"))
	fmt.Println()

	// 1. Check Provider Availability
	fmt.Println("Checking providers...")
	availableProviders := provider.CheckAvailable()
	for name, info := range availableProviders {
		if info.Available {
			fmt.Printf("  %s %s\n", greenStyle.Render("✓"), name)
		} else {
			fmt.Printf("  %s %s: %s\n", yellowStyle.Render("⚠"), name, info.Message)
			warnings = append(warnings, fmt.Sprintf("Provider %s: %s", name, info.Message))
		}
	}
	fmt.Println()

	// 2. Check State Backend
	fmt.Println("Checking state backend...")
	dotisanDir := os.ExpandEnv("$HOME/.dotisan")
	if err := os.MkdirAll(dotisanDir, 0755); err != nil {
		fmt.Printf("  %s Cannot create dotisan directory: %s\n", redStyle.Render("✗"), err)
		hasErrors = true
		issues = append(issues, fmt.Sprintf("Cannot create dotisan directory: %s", err))
	} else {
		// Try to load state to check connectivity
		statePath := dotisanDir + "/state.json"
		backend := state.NewLocalBackend(statePath)
		ctx := context.Background()
		_, err := backend.Load(ctx)
		if err != nil {
			// Error is acceptable if state file doesn't exist yet
			if os.IsNotExist(err) {
				fmt.Printf("  %s State backend (local) ready\n", greenStyle.Render("✓"))
			} else {
				fmt.Printf("  %s State backend error: %s\n", redStyle.Render("✗"), err)
				hasErrors = true
				issues = append(issues, fmt.Sprintf("State backend error: %s", err))
			}
		} else {
			fmt.Printf("  %s State backend (local) ready\n", greenStyle.Render("✓"))
		}
	}
	fmt.Println()

	// 3. Check Config Files
	fmt.Println("Checking configuration files...")

	// Check ~/.dotisan/config.yaml
	configPath := os.ExpandEnv("$HOME/.dotisan/config.yaml")
	_, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  %s config.yaml not found (will use defaults)\n", yellowStyle.Render("⚠"))
			warnings = append(warnings, "No config.yaml found, will use defaults")
		} else {
			fmt.Printf("  %s Cannot read config.yaml: %s\n", redStyle.Render("✗"), err)
			hasErrors = true
			issues = append(issues, fmt.Sprintf("Cannot read config.yaml: %s", err))
		}
	} else {
		// Try to parse config
		_, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("  %s Cannot parse config.yaml: %s\n", redStyle.Render("✗"), err)
			hasErrors = true
			issues = append(issues, fmt.Sprintf("Cannot parse config.yaml: %s", err))
		} else {
			fmt.Printf("  %s config.yaml valid\n", greenStyle.Render("✓"))
		}
	}

	// Check ~/.dotfiles/values.yaml
	valuesPath := os.ExpandEnv("$HOME/.dotfiles/values.yaml")
	_, err = os.Stat(valuesPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  %s values.yaml not found (optional)\n", greenStyle.Render("✓"))
		} else {
			fmt.Printf("  %s Cannot read values.yaml: %s\n", redStyle.Render("✗"), err)
			hasErrors = true
			issues = append(issues, fmt.Sprintf("Cannot read values.yaml: %s", err))
		}
	} else {
		// Try to parse values
		_, err := config.LoadValues(valuesPath)
		if err != nil {
			fmt.Printf("  %s Cannot parse values.yaml: %s\n", redStyle.Render("✗"), err)
			hasErrors = true
			issues = append(issues, fmt.Sprintf("Cannot parse values.yaml: %s", err))
		} else {
			fmt.Printf("  %s values.yaml valid\n", greenStyle.Render("✓"))
		}
	}

	// Check ~/.dotfiles/ directory
	dotfilesPath := os.ExpandEnv("$HOME/.dotfiles")
	_, err = os.Stat(dotfilesPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  %s .dotfiles/ directory not found\n", redStyle.Render("✗"))
			hasErrors = true
			issues = append(issues, ".dotfiles/ directory not found - this is where your resource definitions should be stored")
		} else {
			fmt.Printf("  %s Cannot read .dotfiles/: %s\n", redStyle.Render("✗"), err)
			hasErrors = true
			issues = append(issues, fmt.Sprintf("Cannot read .dotfiles/: %s", err))
		}
	} else {
		fmt.Printf("  %s .dotfiles/ directory exists\n", greenStyle.Render("✓"))
	}
	fmt.Println()

	// 4. Summary
	fmt.Println(headerStyle.Render("Summary"))
	if hasErrors {
		fmt.Printf("  %s Issues found: %d\n", redStyle.Render("✗"), len(issues))
		for _, issue := range issues {
			fmt.Printf("    - %s\n", issue)
		}
		fmt.Println()
		fmt.Println(yellowStyle.Render("Some checks failed. Please fix the issues above before running 'dotisan apply'."))
		os.Exit(1)
	} else if len(warnings) > 0 {
		fmt.Printf("  %s Working, but %d warnings:\n", yellowStyle.Render("⚠"), len(warnings))
		for _, warning := range warnings {
			fmt.Printf("    - %s\n", warning)
		}
		fmt.Println()
		fmt.Println(greenStyle.Render("dotisan is functional but some features may be limited."))
	} else {
		fmt.Printf("  %s All checks passed!\n", greenStyle.Render("✓"))
		fmt.Println()
		fmt.Println(greenStyle.Render("Your dotisan setup looks good. Ready to use 'dotisan plan' and 'dotisan apply'."))
	}

	return nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
