package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/juthrbog/awss/internal/state"
	"github.com/spf13/cobra"
)

func executeRoot(args ...string) (string, error) {
	cmd := newRootCommand()
	var out, diagnostic bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&diagnostic)
	cmd.SilenceErrors = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestPreviousProfileToggle(t *testing.T) {
	profileFixture(t)
	out, err := executeRoot("production")
	if err != nil || out != "export AWS_PROFILE=production\nexport AWS_REGION=us-east-1\n" {
		t.Fatalf("select = %q, %v", out, err)
	}
	if got, err := state.Previous(); got != "default" || err != nil {
		t.Fatalf("history = %q, %v", got, err)
	}
	// Each invocation is a new command and reads the on-disk history.
	t.Setenv("AWS_PROFILE", "production")
	out, err = executeRoot("-")
	if err != nil || out != "export AWS_PROFILE=default\nexport AWS_REGION=us-west-2\n" {
		t.Fatalf("previous = %q, %v", out, err)
	}
	t.Setenv("AWS_PROFILE", "default")
	out, err = executeRoot("--shell", "fish", "-")
	if err != nil || out != "set -gx AWS_PROFILE production\nset -gx AWS_REGION us-east-1\n" {
		t.Fatalf("toggle back = %q, %v", out, err)
	}
}

func TestSelectSubcommandSavesHistory(t *testing.T) {
	profileFixture(t)
	out, err := executeRoot("select", "--shell", "fish", "production")
	if err != nil || !strings.HasPrefix(out, "set -gx AWS_PROFILE production\n") {
		t.Fatalf("select = %q, %v", out, err)
	}
	if got, err := state.Previous(); got != "default" || err != nil {
		t.Fatalf("history = %q, %v", got, err)
	}
}

func TestSwitchErrorsDoNotEmitExportsOrOverwriteHistory(t *testing.T) {
	profileFixture(t)
	if out, err := executeRoot("-"); err == nil || out != "" || !strings.Contains(err.Error(), "no previous profile") {
		t.Fatalf("missing history = %q, %v", out, err)
	}
	if err := state.SavePrevious("deleted-profile", "production"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"-"}, {"missing-profile"}, {"--shell", "invalid", "production"}} {
		if out, err := executeRoot(args...); err == nil || out != "" {
			t.Fatalf("%v = %q, %v", args, out, err)
		}
		if got, err := state.Previous(); got != "deleted-profile" || err != nil {
			t.Fatalf("error changed history: %q, %v", got, err)
		}
	}
	// A cache write failure must not emit code that changes the active profile.
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", file)
	if out, err := executeRoot("production"); err == nil || out != "" {
		t.Fatalf("cache write failure = %q, %v", out, err)
	}
}

func TestPickerSavesHistoryOnlyOnSelection(t *testing.T) {
	profileFixture(t)
	if err := state.SavePrevious("staging", "default"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "default", "production"} {
		var out, diagnostic bytes.Buffer
		err := defaultAWSFiles().runDefault(&out, &diagnostic, "", true, func([]string, string) (string, error) { return name, nil })
		if err != nil {
			t.Fatal(err)
		}
		want := "staging"
		if name == "production" {
			want = "default"
		}
		if got, err := state.Previous(); got != want || err != nil {
			t.Fatalf("selection %q saved %q, %v; want %q", name, got, err, want)
		}
	}
}

func TestCurrent(t *testing.T) {
	profileFixture(t)
	t.Setenv("AWS_REGION", "eu-west-2") // Report environment, not profile defaults.
	for _, flag := range []string{"-c", "--current"} {
		if got, err := executeRoot(flag); err != nil || got != "default (eu-west-2)\n" {
			t.Fatalf("current = %q, %v", got, err)
		}
	}
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_REGION", "")
	if got, err := executeRoot("-c"); err != nil || got != "<unset> (<unset>)\n" {
		t.Fatalf("unset current = %q, %v", got, err)
	}
	if _, err := state.Previous(); err == nil {
		t.Fatal("printing current state must not save history")
	}
}

func TestRegionFlags(t *testing.T) {
	profileFixture(t)
	for _, args := range [][]string{
		{"-r", "eu-west-1"}, {"--region", "eu-west-1"},
		{"--region=eu-west-1"}, {"-r=eu-west-1"},
		{"--shell", "fish", "-r", "eu-west-1"},
	} {
		out, err := executeRoot(args...)
		want := "export AWS_REGION=eu-west-1\n"
		if slices.Contains(args, "fish") {
			want = "set -gx AWS_REGION eu-west-1\n"
		}
		if err != nil || out != want {
			t.Fatalf("%v = %q, %v; want %q", args, out, err, want)
		}
	}
	if _, err := state.Previous(); err == nil {
		t.Fatal("region switching must not save profile history")
	}
}

func TestInvalidFlags(t *testing.T) {
	profileFixture(t)
	for _, args := range [][]string{
		{"-c", "production"}, {"-c", "-r"}, {"--region=eu-west-1", "us-east-1"},
		{"-r", "not-a-region"}, {"--shell=invalid", "-r", "eu-west-1"},
		{"production", "staging"},
	} {
		out, err := executeRoot(args...)
		if err == nil || out != "" {
			t.Fatalf("%v = %q, %v; want error and empty stdout", args, out, err)
		}
	}
}

func TestRegionPicker(t *testing.T) {
	t.Setenv("AWS_REGION", "eu-west-1")
	for _, selection := range []string{"us-east-1", ""} {
		var out bytes.Buffer
		err := runRegion(&out, "", "", true, func(names []string, current string) (string, error) {
			if !slices.Contains(names, "us-east-1") || current != "eu-west-1" {
				t.Fatalf("picker got %v, %q", names, current)
			}
			return selection, nil
		})
		want := ""
		if selection != "" {
			want = "export AWS_REGION=us-east-1\n"
		}
		if err != nil || out.String() != want {
			t.Fatalf("picker output = %q, %v", out.String(), err)
		}
	}
	var out bytes.Buffer
	if err := runRegion(&out, "", "", false, nil); err == nil || out.Len() != 0 {
		t.Fatalf("non-TTY picker = %q, %v", out.String(), err)
	}
	want := errors.New("picker failed")
	err := runRegion(&out, "", "", true, func([]string, string) (string, error) { return "", want })
	if !errors.Is(err, want) || out.Len() != 0 {
		t.Fatalf("picker error = %q, %v", out.String(), err)
	}
}

func TestCompletions(t *testing.T) {
	profileFixture(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"__complete", ""}, "production\n"},
		{[]string{"__complete", "pr"}, "production\n:4\n"},
		{[]string{"__complete", "select", "pr"}, "production\n:4\n"},
		{[]string{"__complete", "-r", "eu-w"}, "eu-west-1\neu-west-2\neu-west-3\n:4\n"},
		{[]string{"__complete", "--region=eu-w"}, "eu-west-1\neu-west-2\neu-west-3\n:4\n"},
		{[]string{"__complete", "production", ""}, ":4\n"},
		{[]string{"__complete", "-c", ""}, ":4\n"},
		{[]string{"__complete", "-r", "eu-west-1", ""}, ":4\n"},
	} {
		out, err := executeRoot(tc.args...)
		if err != nil || !strings.Contains(out, tc.want) {
			t.Fatalf("%v = %q, %v; want %q", tc.args, out, err, tc.want)
		}
	}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		out, err := executeRoot("completion", shell)
		if err != nil || !strings.Contains(out, "__complete") {
			t.Fatalf("%s completion generation failed: %v", shell, err)
		}
	}
	if _, err := state.Previous(); err == nil {
		t.Fatal("completion must not change profile history")
	}
	t.Setenv("AWS_CONFIG_FILE", t.TempDir())
	_, directive := completeProfiles(nil, nil, "")
	if directive&cobra.ShellCompDirectiveError == 0 {
		t.Fatal("unreadable config should report a completion error")
	}
}
