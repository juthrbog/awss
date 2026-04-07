package cmd

import (
	"fmt"

	"github.com/juthrbog/awss/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all AWS profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := config.ListProfiles(
			config.DefaultConfigPath(),
			config.DefaultCredentialsPath(),
		)
		if err != nil {
			return fmt.Errorf("loading profiles: %w", err)
		}
		for _, p := range profiles {
			fmt.Println(p)
		}
		return nil
	},
}
