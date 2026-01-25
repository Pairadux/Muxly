package cmd

import (
	"fmt"
	"os"

	"github.com/Pairadux/muxly/internal/checks"
	"github.com/spf13/cobra"
)

var doctorQuiet bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate muxly's environment and configuration",
	Long: `Runs diagnostic checks to verify that muxly is properly configured.

Checks performed:
  • External dependencies (tmux, fzf, editor)
  • Configuration file validity
  • Directory accessibility

Exit codes:
  0 - All checks pass (warnings allowed)
  1 - One or more errors found`,
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVarP(&doctorQuiet, "quiet", "q", false, "Only show warnings and errors")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	var allResults []checks.CheckResult

	externalResults := checks.CheckExternalUtils()
	externalResults = append(externalResults, checks.CheckEditor(cfg.Settings.Editor))
	allResults = append(allResults, externalResults...)
	fmt.Print(checks.FormatSection("External Dependencies", externalResults, doctorQuiet))

	var configResults []checks.CheckResult
	configResults = append(configResults, checks.CheckConfigFile(cfgFilePath))
	configResults = append(configResults, checks.ValidateConfig(&cfg)...)
	allResults = append(allResults, configResults...)
	fmt.Print(checks.FormatSection("Configuration", configResults, doctorQuiet))

	dirResults := checks.ValidateDirectories(&cfg)
	allResults = append(allResults, dirResults...)
	if len(dirResults) > 0 {
		fmt.Print(checks.FormatSection("Directories", dirResults, doctorQuiet))
	}

	fmt.Println()
	fmt.Println(checks.FormatSummary(allResults))

	if checks.HasErrors(allResults) {
		os.Exit(1)
	}

	return nil
}
