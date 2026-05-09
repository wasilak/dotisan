package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wasilak/nim/pkg/audit"
	"github.com/wasilak/nim/pkg/config"
	"github.com/wasilak/nim/pkg/provider"
	"github.com/wasilak/nim/pkg/resource"
	"github.com/wasilak/nim/pkg/state"
	"github.com/wasilak/nim/pkg/style"
	"github.com/wasilak/nim/pkg/ui"

	"github.com/spf13/cobra"
)

var doctorValidateFlag bool
var doctorOutputFlag string
var doctorSeverityFlag string

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:          "doctor",
	SilenceUsage: true,
	Short:        "Check system prerequisites",
	Long:         "Check system prerequisites, provider availability, and configuration validity.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor(cmd.Context())
	},
}

func runDoctor(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("internal: context is nil")
	}
	hasErrors := false
	issues := []string{}
	warnings := []string{}

	// Header
	headerBox := style.InfoBox.Render(
		style.Header.Render("nim doctor") + "\n\n" +
			style.DimStyle.Render("Checking system prerequisites and configuration"),
	)
	fmt.Println(headerBox)
	fmt.Println()

	// 1. Providers Section
	fmt.Println(style.Header.Render("Providers"))
	ensureProvidersRegistered()
	availableProviders := provider.CheckAvailable()
	for name, info := range availableProviders {
		if info.Available {
			fmt.Printf("  %s %-20s %s\n", style.StyledIconSuccess, name, style.Success.Render("installed"))
		} else {
			fmt.Printf("  %s %-20s %s\n", style.IconWarning, name, info.Message)
			warnings = append(warnings, fmt.Sprintf("Provider %s: %s", name, info.Message))
		}
	}
	fmt.Println()

	// 2. Check State Backend
	fmt.Println("Checking state backend...")
	nimDir := os.ExpandEnv("$HOME/.config/nim")
	if err := os.MkdirAll(nimDir, 0755); err != nil {
		fmt.Printf("  %s Cannot create nim directory: %s\n", style.IconError, err)
		hasErrors = true
		issues = append(issues, fmt.Sprintf("Cannot create nim directory: %s", err))
	} else {
		// Try to load state to check connectivity
		statePath := nimDir + "/state.json"
		backend := state.NewLocalBackend(statePath)
		_, err := backend.Load(ctx)
		if err != nil {
			// Error is acceptable if state file doesn't exist yet
			if os.IsNotExist(err) {
				fmt.Printf("  %s State backend (local) ready\n", style.StyledIconSuccess)
			} else {
				fmt.Printf("  %s State backend error: %s\n", style.StyledIconError, err)
				hasErrors = true
				issues = append(issues, fmt.Sprintf("State backend error: %s", err))
			}
		} else {
			fmt.Printf("  %s State backend (local) ready\n", style.StyledIconSuccess)
		}
	}
	fmt.Println()

	// 3. Check Config Files
	fmt.Println("Checking configuration files...")

	// Check ~/.config/nim/config.yaml
	configPath := os.ExpandEnv("$HOME/.config/nim/config.yaml")
	_, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  %s config.yaml not found (will use defaults)\n", style.IconWarning)
			warnings = append(warnings, "No config.yaml found, will use defaults")
		} else {
			fmt.Printf("  %s Cannot read config.yaml: %s\n", style.IconError, err)
			hasErrors = true
			issues = append(issues, fmt.Sprintf("Cannot read config.yaml: %s", err))
		}
	} else {
		// Try to parse config
		_, err := config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("  %s Cannot parse config.yaml: %s\n", style.IconError, err)
			hasErrors = true
			issues = append(issues, fmt.Sprintf("Cannot parse config.yaml: %s", err))
		} else {
			fmt.Printf("  %s config.yaml valid\n", style.StyledIconSuccess)
		}
	}

	// Check ~/.config/nim/values.yaml
	valuesPath := os.ExpandEnv("$HOME/.config/nim/values.yaml")
	_, err = os.Stat(valuesPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  %s values.yaml not found (optional)\n", style.StyledIconSuccess)
		} else {
			fmt.Printf("  %s Cannot read values.yaml: %s\n", style.IconError, err)
			hasErrors = true
			issues = append(issues, fmt.Sprintf("Cannot read values.yaml: %s", err))
		}
	} else {
		// Try to parse values
		_, err := config.LoadValues(valuesPath)
		if err != nil {
			fmt.Printf("  %s Cannot parse values.yaml: %s\n", style.IconError, err)
			hasErrors = true
			issues = append(issues, fmt.Sprintf("Cannot parse values.yaml: %s", err))
		} else {
			fmt.Printf("  %s values.yaml valid\n", style.StyledIconSuccess)
		}
	}

	// Check ~/.config/nim/ directory
	configDir := os.ExpandEnv("$HOME/.config/nim")
	_, err = os.Stat(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  %s ~/.config/nim/ directory not found\n", style.IconError)
			hasErrors = true
			issues = append(issues, "~/.config/nim/ directory not found - this is where your resource definitions should be stored")
		} else {
			fmt.Printf("  %s Cannot read ~/.config/nim/: %s\n", style.IconError, err)
			hasErrors = true
			issues = append(issues, fmt.Sprintf("Cannot read ~/.config/nim/: %s", err))
		}
	} else {
		fmt.Printf("  %s ~/.config/nim/ directory exists\n", style.StyledIconSuccess)
	}
	fmt.Println()

	// 4. Validate Resources (if requested)
	if doctorValidateFlag {
		// Use provided context so cancellation from signals propagates here
		// Use the spinner for validation
		// Run enhanced audit using pkg/audit
		auditor := audit.NewAuditor(configDir)
		var auditRes *audit.AuditResult
		var auditErr error
		auditErr = ui.RunWithSpinner(ctx, style.Info, "Validating resource files...", "validation cancelled", func(ctx context.Context, publish func(ui.MessageLevel, string)) error {
			var err error
			auditRes, err = auditor.Run()
			return err
		})
		if auditErr != nil {
			fmt.Printf("  %s Resource audit failed: %v\n", style.IconError, auditErr)
		} else {
			// Output according to doctorOutputFlag
			min := audit.SeverityWarning
			if doctorSeverityFlag == "error" {
				min = audit.SeverityError
			} else if doctorSeverityFlag == "info" {
				min = audit.SeverityInfo
			}
			if doctorOutputFlag == "json" {
				out, err := audit.ReportJSON(auditRes)
				if err != nil {
					fmt.Printf("  %s Failed to render audit JSON: %v\n", style.IconError, err)
				} else {
					fmt.Println(string(out))
				}
			} else {
				txt := audit.ReportText(auditRes, min)
				fmt.Println(txt)
			}

			if auditRes.Summary.Errors > 0 {
				hasErrors = true
				issues = append(issues, fmt.Sprintf("%d audit errors", auditRes.Summary.Errors))
			}
			if auditRes.Summary.Warnings > 0 {
				warnings = append(warnings, fmt.Sprintf("%d audit warnings", auditRes.Summary.Warnings))
			}
		}
		fmt.Println()
	}

	// 5. Summary
	fmt.Println(style.Header.Render("Summary"))
	if hasErrors {
		fmt.Printf("  %s Issues found: %d\n", style.IconError, len(issues))
		for _, issue := range issues {
			fmt.Printf("    - %s\n", issue)
		}
		fmt.Println()
		fmt.Println(style.Warning.Render("Some checks failed. Please fix the issues above before running 'nim apply'."))
		os.Exit(1)
	} else if len(warnings) > 0 {
		fmt.Printf("  %s Working, but %d warnings:\n", style.IconWarning, len(warnings))
		for _, warning := range warnings {
			fmt.Printf("    - %s\n", warning)
		}
		fmt.Println()
		fmt.Println(style.Success.Render("nim is functional but some features may be limited."))
	} else {
		fmt.Printf("  %s All checks passed!\n", style.StyledIconSuccess)
		fmt.Println()
		fmt.Println(style.Success.Render("Your nim setup looks good. Ready to use 'nim plan' and 'nim apply'."))
	}

	return nil
}

// validateResources scans all YAML files in the resources directory and validates them.
// It returns a list of validation errors with file paths.
func validateResources(ctx context.Context, configDir, valuesPath string) ([]string, error) {
	var errors []string

	// Load values for templating
	values, _ := config.LoadValues(valuesPath)

	// Create template context
	envVars := make(map[string]string)
	for _, e := range os.Environ() {
		if i := strings.Index(e, "="); i > 0 {
			envVars[e[:i]] = e[i+1:]
		}
	}
	hostname, _ := os.Hostname()
	tctx := &config.TemplateContext{
		Env:    envVars,
		OS:     config.OSInfo{Hostname: hostname},
		Values: values,
	}

	// Create loader
	loader := resource.NewLoader(configDir, tctx)
	_ = loader // Not used directly, but we walk manually below

	// Walk the directory manually to validate each file
	resourcesDir := filepath.Join(configDir, "resources")
	_, err := os.Stat(resourcesDir)
	if os.IsNotExist(err) {
		// No resources directory yet - that's OK
		return errors, nil
	}

	err = filepath.Walk(resourcesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process YAML files
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		// Allow cancellation between files
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read file
		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: cannot read file: %v", path, err))
			return nil
		}

		// Try to unmarshal (this validates apiVersion and kind)
		res, err := resource.UnmarshalYAML(data)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		// Validate the resource struct
		if err := res.Validate(); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		return nil
	})

	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to walk resources directory: %v", err))
	}

	return errors, nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorValidateFlag, "validate", false, "Also validate all resource YAML files")
	doctorCmd.Flags().StringVar(&doctorOutputFlag, "output", "text", "Output format: text or json")
	doctorCmd.Flags().StringVar(&doctorSeverityFlag, "severity", "warning", "Minimum severity to report: error, warning, info")
}
