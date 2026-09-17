package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/juthrbog/awss/internal/config"
	"github.com/juthrbog/awss/internal/state"
	"github.com/spf13/cobra"
)

func newSelectCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "select <profile>",
		Short:             "Output export statements for a profile",
		Long:              "Print export statements that set AWS_PROFILE (and optionally AWS_REGION) for the given profile. Intended for use with eval.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProfiles,
		RunE: func(cmd *cobra.Command, args []string) error {
			shell, err := cmd.Flags().GetString("shell")
			if err != nil {
				return err
			}
			files, err := resolveAWSFiles(cmd)
			if err != nil {
				return err
			}
			return files.writeProfileExports(cmd.OutOrStdout(), args[0], shell)
		},
	}
}

func (files awsFiles) writeProfileExports(out io.Writer, name, shell string) error {
	if err := validateShell(shell); err != nil {
		return err
	}
	if name == "-" {
		previous, err := state.Previous()
		if err != nil {
			return err
		}
		name = previous
	}
	profile, err := config.LoadProfile(files.configPath, files.credentialsPath, name)
	if err != nil {
		return err
	}
	// Validate the destination before touching history or emitting shell code.
	if err := state.SavePrevious(os.Getenv("AWS_PROFILE"), name); err != nil {
		return err
	}
	_, err = fmt.Fprint(out, files.exports(shell)+formatExports(profile, shell))
	return err
}

// formatExports builds the export statements for a given profile.
//
// Always emits AWS_PROFILE. Emits AWS_REGION when the profile defines one,
// otherwise unsets it to prevent stale values from a previous switch.
// When shell is "fish", outputs fish-compatible set/set -e syntax.
func formatExports(p config.Profile, shell string) string {
	var b strings.Builder
	p.Name = quoteShellValue(p.Name, shell)
	if shell == "fish" {
		fmt.Fprintf(&b, "set -gx AWS_PROFILE %s\n", p.Name)
	} else {
		fmt.Fprintf(&b, "export AWS_PROFILE=%s\n", p.Name)
	}
	b.WriteString(formatRegionExport(p.Region, shell))
	return b.String()
}
