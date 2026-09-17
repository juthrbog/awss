package cmd

import (
	"strings"

	"github.com/juthrbog/awss/internal/config"
	"github.com/juthrbog/awss/internal/regions"
	"github.com/spf13/cobra"
)

func completeRoot(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
	if cmd.Flags().Changed("current") {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if cmd.Flags().Changed("region") {
		region, _ := cmd.Flags().GetString("region")
		if len(args) != 0 || (region != "" && region != interactiveRegion) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completeRegions(cmd, args, prefix)
	}
	return completeProfiles(cmd, args, prefix)
}

func completeProfiles(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	files, err := resolveAWSFiles(cmd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
	}
	names, err := config.ListProfiles(files.configPath, files.credentialsPath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
	}
	return matchingNames(names, prefix), cobra.ShellCompDirectiveNoFileComp
}

func completeRegions(_ *cobra.Command, _ []string, prefix string) ([]string, cobra.ShellCompDirective) {
	return matchingNames(regions.Names(), prefix), cobra.ShellCompDirectiveNoFileComp
}

func matchingNames(names []string, prefix string) []string {
	var matches []string
	for _, name := range names {
		if strings.HasPrefix(name, prefix) {
			matches = append(matches, name)
		}
	}
	return matches
}
