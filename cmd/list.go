package cmd

import (
	"fmt"

	"github.com/juthrbog/awss/internal/config"
	"github.com/spf13/cobra"
)

func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all AWS profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveAWSFiles(cmd)
			if err != nil {
				return err
			}
			profiles, err := config.ListProfiles(files.configPath, files.credentialsPath)
			if err != nil {
				return fmt.Errorf("loading profiles: %w", err)
			}
			for _, p := range profiles {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), p); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
