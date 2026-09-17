package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestShellQualityOfLife(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shells")
	}
	binary := filepath.Join(t.TempDir(), "awss")
	build := exec.Command("go", "build", "-o", binary, "..") // #nosec G204 -- build this repository into the test directory
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building test binary: %v\n%s", err, out)
	}

	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed", shell)
			}
			profileFixture(t)
			t.Setenv("AWS_REGION", "us-west-2")
			// Verify exports preserve shell metacharacters as data in each shell.
			name := "team's dev\\ops;$HOME"
			t.Setenv("AWSS_TEST_PROFILE", name)
			file, err := os.OpenFile(filepath.Clean(os.Getenv("AWS_CONFIG_FILE")), os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := file.WriteString("\n[profile " + name + "]\n"); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			wrapper, err := renderInit(shell, binary)
			if err != nil {
				t.Fatal(err)
			}
			script := wrapper + `
awss --version >/dev/null
awss -v >/dev/null
awss production
awss -c
awss -r eu-west-1
awss --current
awss -
awss -c
awss -
awss --current
awss select staging
awss -c
awss -r=eu-central-1
awss --current
awss "$AWSS_TEST_PROFILE"
awss -c
awss -r --help >/dev/null
awss -rc --help >/dev/null
awss __complete pr 2>/dev/null
awss --shell bash -c
awss -r invalid-region
`
			if shell == "fish" {
				script += `printf 'failure:%s\n' $status
awss completion fish | source
complete -C 'awss pr'
`
			} else {
				script += `printf 'failure:%s\n' "$?"
`
			}
			if shell == "bash" {
				script += `source <(awss completion bash)
# The full completion engine requires bash-completion; verify registration
# here, and the dynamic completion protocol above, without that dependency.
case "$(complete -p awss)" in
  *__start_awss*) printf 'bash-completion-registered\n' ;;
  *) exit 1 ;;
esac
`
			}
			if shell == "zsh" {
				script += `autoload -Uz compinit
compinit -D
source <(awss completion zsh)
print -r -- "${_comps[awss]}"
`
			}
			args := []string{"--noprofile", "--norc", "-c", script}
			switch shell {
			case "zsh":
				args = []string{"-f", "-c", script}
			case "fish":
				args = []string{"--no-config", "-c", script}
			}
			out, err := exec.Command(path, args...).CombinedOutput() // #nosec G204 -- known shells running a generated test fixture
			if err != nil {
				t.Fatalf("shell failed: %v\n%s", err, out)
			}
			want := strings.Join([]string{
				"production (us-east-1)",
				"production (eu-west-1)",
				"default (us-west-2)",
				"production (us-east-1)",
				"staging (<unset>)",
				"staging (eu-central-1)",
				name + " (<unset>)",
				"production", ":4",
				name + " (<unset>)",
				`Error: unknown AWS region "invalid-region"`,
				"failure:1",
			}, "\n") + "\n"
			switch shell {
			case "zsh":
				want += "_awss\n"
			case "bash":
				want += "bash-completion-registered\n"
			case "fish":
				want += "production\n"
			}
			if string(out) != want {
				t.Fatalf("output = %q\nwant = %q", out, want)
			}
		})
	}
}
