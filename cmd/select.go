package cmd

import (
	"fmt"
	"strings"

	"github.com/juthrbog/awss/internal/config"
	"github.com/spf13/cobra"
)

var shellFlag string

func init() {
	selectCmd.Flags().StringVar(&shellFlag, "shell", "", "output syntax: bash, zsh, or fish (default bash)")
	rootCmd.AddCommand(selectCmd)
}

var selectCmd = &cobra.Command{
	Use:   "select <profile>",
	Short: "Output export statements for a profile",
	Long:  "Print export statements that set AWS_PROFILE (and optionally AWS_REGION) for the given profile. Intended for use with eval.",
	Args:  cobra.ExactArgs(1),
	// Cobra prints RunE errors to stderr automatically, keeping stdout clean.
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		profile, err := config.LoadProfile(
			config.DefaultConfigPath(),
			config.DefaultCredentialsPath(),
			name,
		)
		if err != nil {
			return err
		}

		fmt.Print(formatExports(profile, shellFlag))
		return nil
	},
}

// formatExports builds the export statements for a given profile.
//
// Always emits AWS_PROFILE. Emits AWS_REGION when the profile defines one,
// otherwise unsets it to prevent stale values from a previous switch.
// When shell is "fish", outputs fish-compatible set/set -e syntax.
func formatExports(p config.Profile, shell string) string {
	var b strings.Builder
	if shell == "fish" {
		fmt.Fprintf(&b, "set -gx AWS_PROFILE %s\n", p.Name)
		if p.Region != "" {
			fmt.Fprintf(&b, "set -gx AWS_REGION %s\n", p.Region)
		} else {
			b.WriteString("set -e AWS_REGION\n")
		}
	} else {
		fmt.Fprintf(&b, "export AWS_PROFILE=%s\n", p.Name)
		if p.Region != "" {
			fmt.Fprintf(&b, "export AWS_REGION=%s\n", p.Region)
		} else {
			b.WriteString("unset AWS_REGION\n")
		}
	}
	return b.String()
}
