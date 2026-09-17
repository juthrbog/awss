package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRenderInit_Bash(t *testing.T) {
	out, err := renderInit("bash", "/usr/local/bin/awss")
	if err != nil {
		t.Fatalf("renderInit(bash) error: %v", err)
	}
	for _, want := range []string{
		"awss()",
		"/usr/local/bin/awss",
		".bashrc",
		`eval "$(`,
		`command "/usr/local/bin/awss" "$@"`,
		`init|list|login|`,
		`[ ! -t 0 ]`,
		`output=$(command "/usr/local/bin/awss" "$@")`,
		`__complete|__completeNoDesc`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("bash output missing %q", want)
		}
	}
}

func TestRenderInit_Zsh(t *testing.T) {
	out, err := renderInit("zsh", "/usr/local/bin/awss")
	if err != nil {
		t.Fatalf("renderInit(zsh) error: %v", err)
	}
	for _, want := range []string{
		"awss()",
		"/usr/local/bin/awss",
		".zshrc",
		"init zsh",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("zsh output missing %q", want)
		}
	}
}

func TestRenderInit_Fish(t *testing.T) {
	out, err := renderInit("fish", "/usr/local/bin/awss")
	if err != nil {
		t.Fatalf("renderInit(fish) error: %v", err)
	}
	for _, want := range []string{
		"function awss",
		"/usr/local/bin/awss",
		"config.fish",
		"--shell fish",
		"init fish | source",
		"case init list login",
		"and not test -t 0",
		`set -l output (command "/usr/local/bin/awss" --shell fish $argv)`,
		`__complete __completeNoDesc`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("fish output missing %q", want)
		}
	}
}

// Execute the generated wrappers, not just template string assertions. In
// particular, a non-TTY profile list must be printed rather than evaluated.
func TestShellWrapperNonTTY(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX shells")
	}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed", shell)
			}
			dir := t.TempDir()
			binary := filepath.Join(dir, "mock-awss")
			// The no-argument output would change AWS_PROFILE if evaluated.
			mock := `#!/bin/sh
if [ "$#" -ne 0 ]; then
  if [ "$1" = --shell ]; then
    shift 2
    [ "$1" = broken ] && exit 7
    echo 'set -gx AWS_PROFILE selected'
    echo 'set -gx AWS_REGION eu-west-1'
  else
    [ "$1" = broken ] && exit 7
    echo 'export AWS_PROFILE=selected'
    echo 'export AWS_REGION=eu-west-1'
  fi
else
  echo 'AWS_PROFILE=must-not-eval'
fi
`
			if err := os.WriteFile(binary, []byte(mock), 0700); err != nil { // #nosec G306 -- executable test fixture in t.TempDir
				t.Fatal(err)
			}
			wrapper, err := renderInit(shell, binary)
			if err != nil {
				t.Fatal(err)
			}
			script := wrapper + `
export AWS_PROFILE=original
awss
printf 'after-list:%s\n' "$AWS_PROFILE"
awss production
printf 'after-select:%s:%s\n' "$AWS_PROFILE" "$AWS_REGION"
awss broken
printf 'error:%s profile:%s\n' "$?" "$AWS_PROFILE"
`
			args := []string{"--noprofile", "--norc", "-c", script}
			if shell == "zsh" {
				args = []string{"-f", "-c", script}
			}
			if shell == "fish" {
				script = wrapper + `
set -gx AWS_PROFILE original
awss
printf 'after-list:%s\n' "$AWS_PROFILE"
awss production
printf 'after-select:%s:%s\n' "$AWS_PROFILE" "$AWS_REGION"
awss broken
printf 'error:%s profile:%s\n' $status "$AWS_PROFILE"
`
				args = []string{"--no-config", "-c", script}
			}
			out, err := exec.Command(path, args...).CombinedOutput() // #nosec G204 -- known shells running a generated test fixture
			if err != nil {
				t.Fatalf("shell failed: %v\n%s", err, out)
			}
			want := "AWS_PROFILE=must-not-eval\nafter-list:original\nafter-select:selected:eu-west-1\nerror:7 profile:selected\n"
			if string(out) != want {
				t.Fatalf("output = %q, want %q", out, want)
			}
		})
	}
}

func TestRenderInit_InvalidShell(t *testing.T) {
	_, err := renderInit("powershell", "/usr/local/bin/awss")
	if err == nil {
		t.Fatal("expected error for invalid shell, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("error = %q, want it to contain 'unsupported shell'", err.Error())
	}
}
