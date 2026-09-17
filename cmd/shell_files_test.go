package cmd

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestShellFileOverrides(t *testing.T) {
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
			configPath, credentialsPath := overrideFixture(t)
			t.Setenv("AWSS_TEST_CONFIG", configPath)
			t.Setenv("AWSS_TEST_CREDENTIALS", credentialsPath)
			wrapper, err := renderInit(shell, binary)
			if err != nil {
				t.Fatal(err)
			}
			// These invocations must print (not eval) lists and completion
			// responses, even when file flags precede the subcommand.
			script := wrapper + `
awss --config-file "$AWSS_TEST_CONFIG" --credentials-file "$AWSS_TEST_CREDENTIALS"
awss --config-file="$AWSS_TEST_CONFIG" list --credentials-file="$AWSS_TEST_CREDENTIALS"
awss --config-file "$AWSS_TEST_CONFIG" --credentials-file "$AWSS_TEST_CREDENTIALS" __complete custom 2>/dev/null
awss -c
awss --config-file "$AWSS_TEST_CONFIG" --credentials-file "$AWSS_TEST_CREDENTIALS" custom
awss -c
printf '%s\n' "$AWS_CONFIG_FILE" "$AWS_SHARED_CREDENTIALS_FILE"
awss custom-credentials
awss -c
awss -
awss -c
`
			args := []string{"--noprofile", "--norc", "-c", script}
			switch shell {
			case "zsh":
				args = []string{"-f", "-c", script}
			case "fish":
				args = []string{"--no-config", "-c", script}
			}
			out, err := exec.Command(path, args...).CombinedOutput() // #nosec G204 -- known shells running generated test commands
			if err != nil {
				t.Fatalf("shell failed: %v\n%s", err, out)
			}
			want := strings.Join([]string{
				"custom", "custom-credentials",
				"custom", "custom-credentials",
				"custom", "custom-credentials", ":4",
				"default (us-west-2)",
				"custom (eu-west-3)", configPath, credentialsPath,
				"custom-credentials (ap-northeast-1)",
				"custom (eu-west-3)",
			}, "\n") + "\n"
			if string(out) != want {
				t.Fatalf("output = %q\nwant = %q", out, want)
			}
		})
	}
}
