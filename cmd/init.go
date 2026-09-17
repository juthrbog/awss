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
  local output exit_code arg short_flags
  local switching=0 skip_value=0 operand=0 literal=0
  # Account for file flags before subcommands and options-only invocations.
  # Human-readable output and Cobra completion responses must never be eval'd.
  for arg in "$@"; do
    if [ "$skip_value" -eq 1 ]; then
      skip_value=0
      continue
    fi
    if [ "$literal" -eq 1 ]; then
      switching=1
      continue
    fi
    case "$arg" in
      --) literal=1 ;;
      --config-file|--credentials-file|--shell) skip_value=1 ;;
      --current|--current=*|--help|--help=*)
        command "{{.BinaryPath}}" "$@"
        return $?
        ;;
      --region|--region=*) switching=1 ;;
      --*) ;;
      -*)
        switching=1
        short_flags=${arg%%=*}
        case "$short_flags" in
          *[ch]*)
            command "{{.BinaryPath}}" "$@"
            return $?
            ;;
        esac
        ;;
      *)
        if [ "$operand" -eq 0 ]; then
          case "$arg" in
            init|list|login|help|completion|__complete|__completeNoDesc)
              command "{{.BinaryPath}}" "$@"
              return $?
              ;;
          esac
          operand=1
        fi
        switching=1
        ;;
    esac
  done
  if [ "$switching" -eq 0 ] && [ ! -t 0 ]; then
    command "{{.BinaryPath}}" "$@"
    return $?
  fi
  output=$(command "{{.BinaryPath}}" "$@")
  exit_code=$?
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
  set -l switching 0
  set -l skip_value 0
  set -l operand 0
  set -l literal 0
  for arg in $argv
    if test $skip_value -eq 1
      set skip_value 0
      continue
    end
    if test $literal -eq 1
      set switching 1
      continue
    end
    switch "$arg"
      case --
        set literal 1
      case --config-file --credentials-file --shell
        set skip_value 1
      case --current '--current=*' --help '--help=*'
        command "{{.BinaryPath}}" $argv
        return $status
      case --region '--region=*'
        set switching 1
      case '--*'
      case '-*'
        set switching 1
        set -l short_flags (string split -m1 = -- "$arg")[1]
        if string match -qr '[ch]' -- "$short_flags"
          command "{{.BinaryPath}}" $argv
          return $status
        end
      case '*'
        if test $operand -eq 0
          switch "$arg"
            case init list login help completion __complete __completeNoDesc
              command "{{.BinaryPath}}" $argv
              return $status
          end
          set operand 1
        end
        set switching 1
    end
  end
  if test $switching -eq 0; and not test -t 0
    command "{{.BinaryPath}}" $argv
    return $status
  end
  set -l output (command "{{.BinaryPath}}" --shell fish $argv)
  set -l cmd_status $status
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
