package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:       "init <shell>",
	Short:     "Output shell integration code",
	Long:      "Print a shell wrapper function for awss. Source the output in your shell config file.",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := args[0]

		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolving executable path: %w", err)
		}
		exe, err = filepath.EvalSymlinks(exe)
		if err != nil {
			return fmt.Errorf("resolving symlinks: %w", err)
		}

		out, err := renderInit(shell, exe)
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	},
}

type initData struct {
	BinaryPath string
	Shell      string
	ConfigFile string
}

const posixTmpl = `# awss shell integration
# Add to your ~/{{.ConfigFile}}:
#   eval "$({{.BinaryPath}} init {{.Shell}})"

awss() {
  local output exit_code
  if [ "$#" -eq 0 ]; then
    # Piped input produces a plain list, which must never be evaluated.
    if [ ! -t 0 ]; then
      command "{{.BinaryPath}}"
      return $?
    fi
    output=$(command "{{.BinaryPath}}")
    exit_code=$?
  else
    case "$1" in
      init|list|login|help|completion|--help|-h)
        command "{{.BinaryPath}}" "$@"
        return $?
        ;;
    esac
    output=$(command "{{.BinaryPath}}" select "$@")
    exit_code=$?
  fi
  if [ $exit_code -eq 0 ]; then
    eval "$output"
  fi
  return $exit_code
}
`

const fishTmpl = `# awss shell integration
# Add to your ~/.config/fish/{{.ConfigFile}}:
#   {{.BinaryPath}} init fish | source

function awss
  set -l output
  set -l cmd_status
  if test (count $argv) -eq 0
    # Piped input produces a plain list, which must never be evaluated.
    if not test -t 0
      command "{{.BinaryPath}}"
      return $status
    end
    set output (command "{{.BinaryPath}}" --shell fish)
    set cmd_status $status
  else
    switch "$argv[1]"
      case init list login help completion --help -h
        command "{{.BinaryPath}}" $argv
        return $status
    end
    set output (command "{{.BinaryPath}}" select --shell fish $argv)
    set cmd_status $status
  end
  if test $cmd_status -eq 0
    # Fish splits command substitutions into lines; preserve those separators.
    printf '%s\n' $output | source
  end
  return $cmd_status
end
`

func renderInit(shell, binaryPath string) (string, error) {
	var tmplStr string
	var data initData

	data.BinaryPath = binaryPath
	data.Shell = shell

	switch shell {
	case "bash":
		tmplStr = posixTmpl
		data.ConfigFile = ".bashrc"
	case "zsh":
		tmplStr = posixTmpl
		data.ConfigFile = ".zshrc"
	case "fish":
		tmplStr = fishTmpl
		data.ConfigFile = "config.fish"
	default:
		return "", fmt.Errorf("unsupported shell: %q (valid: bash, zsh, fish)", shell)
	}

	tmpl, err := template.New("init").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}
