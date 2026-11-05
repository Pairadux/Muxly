package cmd

import (
	"errors"
	"fmt"

	"github.com/Pairadux/muxly/internal/checks"
	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/spf13/cobra"
)

var doctorQuiet bool

// errDoctorFoundProblems makes muxly exit 1 after "muxly doctor" reports an error.
var errDoctorFoundProblems = errors.New("doctor found problems")

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate muxly's environment and configuration",
	Long: `Runs diagnostic checks to verify that muxly is properly configured.

Checks performed:
  • External dependencies (tmux, fzf, editor)
  • Configuration file validity, by the same rules as muxly itself
  • Directory accessibility

The config file doesn't have to load for this to run, which makes it the place to find
out what's wrong with one.

Exit codes:
  0 - All checks pass (warnings allowed)
  1 - One or more errors found`,
	Annotations: map[string]string{
		constants.AnnotationSkipUtilsCheck:  "true",
		constants.AnnotationSkipConfigCheck: "true",
	},
	RunE: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVarP(&doctorQuiet, "quiet", "q", false, "Only show warnings and errors")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	var allResults []checks.CheckResult

	externalResults := checks.CheckExternalUtils()
	externalResults = append(externalResults, checks.CheckEditor(config.Editor(state.Cfg.Settings.Editor)))
	allResults = append(allResults, externalResults...)
	fmt.Print(checks.FormatSection("External Dependencies", externalResults, doctorQuiet))

	configResults := checks.CheckConfig(state.CfgFilePath, &state.Cfg, state.CfgLoad)
	allResults = append(allResults, configResults...)
	fmt.Print(checks.FormatSection("Configuration", configResults, doctorQuiet))

	dirResults := checks.ValidateDirectories(&state.Cfg)
	allResults = append(allResults, dirResults...)
	if len(dirResults) > 0 {
		fmt.Print(checks.FormatSection("Directories", dirResults, doctorQuiet))
	}

	fmt.Println()
	fmt.Println(checks.FormatSummary(allResults))

	if checks.HasErrors(allResults) {
		// The summary has said it all, so this only sets the exit code.
		cmd.SilenceErrors = true
		return errDoctorFoundProblems
	}

	return nil
}
