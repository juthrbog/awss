package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/x/term"
	"github.com/juthrbog/awss/internal/config"
	"github.com/juthrbog/awss/internal/picker"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "awss",
	Short: "AWS profile switcher",
	Long:  "A fast, interactive AWS profile and region switcher. With no arguments, pick a profile interactively (or list profiles when stdin is not a terminal).",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDefault(cmd.OutOrStdout(), cmd.ErrOrStderr(), pickerShell, term.IsTerminal(os.Stdin.Fd()), func(names []string, current string) (string, error) {
			return picker.Run(names, current, os.Stdin, cmd.ErrOrStderr())
		})
	},
}

var pickerShell string

func init() {
	rootCmd.Flags().StringVar(&pickerShell, "shell", "", "output syntax: bash, zsh, or fish (default bash)")
}

func runDefault(out, diagnostic io.Writer, shell string, interactive bool, choose func([]string, string) (string, error)) error {
	names, err := config.ListProfiles(config.DefaultConfigPath(), config.DefaultCredentialsPath())
	if err != nil {
		return fmt.Errorf("loading profiles: %w", err)
	}
	if !interactive {
		for _, name := range names {
			if _, err := fmt.Fprintln(out, name); err != nil {
				return err
			}
		}
		return nil
	}
	if len(names) == 0 {
		_, err := fmt.Fprintf(diagnostic, "No AWS profiles found in %s or %s.\nConfigure an AWS profile or run 'awss login' to generate SSO profiles.\n", config.DefaultConfigPath(), config.DefaultCredentialsPath())
		return err
	}
	name, err := choose(names, os.Getenv("AWS_PROFILE"))
	if err != nil {
		return err
	}
	if name == "" {
		return nil // Cancellation must leave stdout empty for shell eval.
	}
	return writeProfileExports(out, name, shell)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
