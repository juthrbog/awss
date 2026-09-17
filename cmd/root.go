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

var rootCmd = newRootCommand()

const interactiveRegion = "interactive"

func newRootCommand() *cobra.Command {
	var shell, region string
	var current bool
	cmd := &cobra.Command{
		Use:               "awss [profile|-]",
		Short:             "AWS profile and region switcher",
		Long:              "A fast, interactive AWS profile and region switcher. With no arguments, pick a profile interactively (or list profiles when stdin is not a terminal).",
		Args:              cobra.MaximumNArgs(1),
		SilenceUsage:      true,
		ValidArgsFunction: completeRoot,
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveAWSFiles(cmd)
			if err != nil {
				return err
			}
			interactive := term.IsTerminal(os.Stdin.Fd())
			choose := func(kind string) func([]string, string) (string, error) {
				return func(names []string, current string) (string, error) {
					return picker.Run(names, current, kind, os.Stdin, cmd.ErrOrStderr())
				}
			}
			switch {
			case current:
				if len(args) != 0 {
					return fmt.Errorf("--current does not accept a profile argument")
				}
				return writeCurrent(cmd.OutOrStdout())
			case cmd.Flags().Changed("region"):
				if region == interactiveRegion {
					region = ""
				}
				if len(args) == 1 {
					if region != "" {
						return fmt.Errorf("specify only one region")
					}
					region = args[0]
				}
				return runRegion(cmd.OutOrStdout(), shell, region, interactive, choose("region"))
			case len(args) == 1:
				return files.writeProfileExports(cmd.OutOrStdout(), args[0], shell)
			default:
				return files.runDefault(cmd.OutOrStdout(), cmd.ErrOrStderr(), shell, interactive, choose("profile"))
			}
		},
	}
	cmd.PersistentFlags().StringVar(&shell, "shell", "", "output syntax: bash, zsh, or fish (default bash)")
	cmd.PersistentFlags().String("config-file", "", "AWS config file (overrides AWS_CONFIG_FILE)")
	cmd.PersistentFlags().String("credentials-file", "", "AWS credentials file (overrides AWS_SHARED_CREDENTIALS_FILE)")
	cmd.Flags().BoolVarP(&current, "current", "c", false, "print current AWS_PROFILE and AWS_REGION")
	cmd.Flags().StringVarP(&region, "region", "r", "", "pick a region, or set it with --region <region>")
	cmd.Flags().Lookup("region").NoOptDefVal = interactiveRegion
	cmd.MarkFlagsMutuallyExclusive("current", "region")
	if err := cmd.RegisterFlagCompletionFunc("region", completeRegions); err != nil {
		panic(err)
	}
	if err := cmd.RegisterFlagCompletionFunc("shell", cobra.FixedCompletions([]string{"bash", "zsh", "fish"}, cobra.ShellCompDirectiveNoFileComp)); err != nil {
		panic(err)
	}
	cmd.AddCommand(newSelectCommand(), newListCommand())
	return cmd
}

func (files awsFiles) runDefault(out, diagnostic io.Writer, shell string, interactive bool, choose func([]string, string) (string, error)) error {
	if err := validateShell(shell); err != nil {
		return err
	}
	names, err := config.ListProfiles(files.configPath, files.credentialsPath)
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
		_, err := fmt.Fprintf(diagnostic, "No AWS profiles found in %s or %s.\nConfigure an AWS profile or run 'awss login' to generate SSO profiles.\n", files.configPath, files.credentialsPath)
		return err
	}
	name, err := choose(names, os.Getenv("AWS_PROFILE"))
	if err != nil {
		return err
	}
	if name == "" {
		return nil // Cancellation must leave stdout empty for shell eval.
	}
	return files.writeProfileExports(out, name, shell)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
