package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/juthrbog/awss/internal/config"
	"github.com/spf13/cobra"
)

type awsFiles struct {
	configPath          string
	credentialsPath     string
	configOverride      bool
	credentialsOverride bool
}

func defaultAWSFiles() awsFiles {
	return awsFiles{
		configPath:      config.DefaultConfigPath(),
		credentialsPath: config.DefaultCredentialsPath(),
	}
}

// resolveAWSFiles applies flag > environment > default precedence independently
// for each file, without changing the process environment (including completion).
func resolveAWSFiles(cmd *cobra.Command) (awsFiles, error) {
	files := defaultAWSFiles()
	if cmd == nil {
		return files, nil
	}
	for _, option := range []struct {
		flag     string
		path     *string
		override *bool
	}{
		{"config-file", &files.configPath, &files.configOverride},
		{"credentials-file", &files.credentialsPath, &files.credentialsOverride},
	} {
		if !cmd.Flags().Changed(option.flag) {
			continue
		}
		value, err := cmd.Flags().GetString(option.flag)
		if err != nil {
			return files, err
		}
		if value == "" {
			return files, fmt.Errorf("--%s requires a non-empty path", option.flag)
		}
		// Absolute paths remain valid when a shell changes directories after
		// selecting a profile with a relative file override.
		path, err := filepath.Abs(value)
		if err != nil {
			return files, fmt.Errorf("resolving --%s: %w", option.flag, err)
		}
		*option.path, *option.override = path, true
	}
	return files, nil
}

// exports keeps subsequent AWS CLI/SDK calls aligned with an explicitly selected
// file. Read-only operations and picker cancellation emit no path exports.
func (files awsFiles) exports(shell string) string {
	var out strings.Builder
	for _, value := range []struct {
		name, path string
		override   bool
	}{
		{"AWS_CONFIG_FILE", files.configPath, files.configOverride},
		{"AWS_SHARED_CREDENTIALS_FILE", files.credentialsPath, files.credentialsOverride},
	} {
		if !value.override {
			continue
		}
		if shell == "fish" {
			fmt.Fprintf(&out, "set -gx %s %s\n", value.name, quoteShellValue(value.path, shell))
		} else {
			fmt.Fprintf(&out, "export %s=%s\n", value.name, quoteShellValue(value.path, shell))
		}
	}
	return out.String()
}
