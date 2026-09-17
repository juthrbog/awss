package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/juthrbog/awss/internal/regions"
)

func writeCurrent(out io.Writer) error {
	profile, region := os.Getenv("AWS_PROFILE"), os.Getenv("AWS_REGION")
	if profile == "" {
		profile = "<unset>"
	}
	if region == "" {
		region = "<unset>"
	}
	_, err := fmt.Fprintf(out, "%s (%s)\n", profile, region)
	return err
}

func runRegion(out io.Writer, shell, region string, interactive bool, choose func([]string, string) (string, error)) error {
	if err := validateShell(shell); err != nil {
		return err
	}
	if region == "" {
		if !interactive {
			return errors.New("region picker requires terminal input; use awss --region <region>")
		}
		var err error
		region, err = choose(regions.Names(), os.Getenv("AWS_REGION"))
		if err != nil {
			return err
		}
		if region == "" {
			return nil
		}
	}
	if !regions.Valid(region) {
		return fmt.Errorf("unknown AWS region %q", region)
	}
	_, err := fmt.Fprint(out, formatRegionExport(region, shell))
	return err
}

func formatRegionExport(region, shell string) string {
	if shell == "fish" {
		if region == "" {
			return "set -e AWS_REGION\n"
		}
		return "set -gx AWS_REGION " + quoteShellValue(region, shell) + "\n"
	}
	if region == "" {
		return "unset AWS_REGION\n"
	}
	return "export AWS_REGION=" + quoteShellValue(region, shell) + "\n"
}

func validateShell(shell string) error {
	switch shell {
	case "", "bash", "zsh", "fish":
		return nil
	default:
		return fmt.Errorf("unsupported shell: %q (valid: bash, zsh, fish)", shell)
	}
}

// Profile names can contain whitespace and shell metacharacters. Quote values
// before emitting code that the shell wrapper will evaluate.
func quoteShellValue(value, shell string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		safe := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_./:-", r)
		return !safe
	}) == -1 {
		return value
	}
	if shell == "fish" {
		value = strings.ReplaceAll(value, "\\", "\\\\")
		return "'" + strings.ReplaceAll(value, "'", "\\'") + "'"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
