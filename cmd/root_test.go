package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func profileFixture(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	if err := os.WriteFile(configPath, []byte("[default]\nregion = us-west-2\n[profile production]\nregion = us-east-1\n[profile staging]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	credentialsPath := filepath.Join(dir, "credentials")
	if err := os.WriteFile(credentialsPath, []byte("[credentials-only]\n[production]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_CONFIG_FILE", configPath)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialsPath)
	t.Setenv("AWS_PROFILE", "default")
}

func TestDefaultNonTTY(t *testing.T) {
	profileFixture(t)
	var out bytes.Buffer
	choose := func([]string, string) (string, error) {
		t.Fatal("non-TTY input must not start the picker")
		return "", nil
	}
	if err := runDefault(&out, io.Discard, "", false, choose); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "credentials-only\ndefault\nproduction\nstaging\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestDefaultSelection(t *testing.T) {
	for _, tc := range []struct {
		name, shell, want string
	}{
		{"production", "", "export AWS_PROFILE=production\nexport AWS_REGION=us-east-1\n"},
		{"staging", "zsh", "export AWS_PROFILE=staging\nunset AWS_REGION\n"},
		{"production", "fish", "set -gx AWS_PROFILE production\nset -gx AWS_REGION us-east-1\n"},
		{"credentials-only", "fish", "set -gx AWS_PROFILE credentials-only\nset -e AWS_REGION\n"},
		{"", "", ""}, // Cancellation is a successful no-op.
	} {
		t.Run(tc.name+"/"+tc.shell, func(t *testing.T) {
			profileFixture(t)
			var out bytes.Buffer
			choose := func(names []string, current string) (string, error) {
				if !reflect.DeepEqual(names, []string{"credentials-only", "default", "production", "staging"}) || current != "default" {
					t.Fatalf("picker got names=%v current=%q", names, current)
				}
				return tc.name, nil
			}
			if err := runDefault(&out, io.Discard, tc.shell, true, choose); err != nil {
				t.Fatal(err)
			}
			if out.String() != tc.want {
				t.Fatalf("output = %q, want %q", out.String(), tc.want)
			}
		})
	}
}

func TestDefaultPickerError(t *testing.T) {
	profileFixture(t)
	var out bytes.Buffer
	want := errors.New("terminal failed")
	err := runDefault(&out, io.Discard, "", true, func([]string, string) (string, error) { return "", want })
	if !errors.Is(err, want) || out.Len() != 0 {
		t.Fatalf("error = %v, stdout = %q", err, out.String())
	}
}

func TestDefaultMissingProfiles(t *testing.T) {
	t.Setenv("AWS_CONFIG_FILE", filepath.Join(t.TempDir(), "missing"))
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", filepath.Join(t.TempDir(), "missing"))
	for _, interactive := range []bool{false, true} {
		var out, diagnostic bytes.Buffer
		if err := runDefault(&out, &diagnostic, "", interactive, nil); err != nil || out.Len() != 0 {
			t.Fatalf("error = %v, stdout = %q", err, out.String())
		}
		if interactive {
			for _, want := range []string{"No AWS profiles found", os.Getenv("AWS_CONFIG_FILE"), os.Getenv("AWS_SHARED_CREDENTIALS_FILE"), "awss login"} {
				if !strings.Contains(diagnostic.String(), want) {
					t.Errorf("diagnostic %q missing %q", diagnostic.String(), want)
				}
			}
		} else if diagnostic.Len() != 0 {
			t.Fatalf("non-TTY listing should stay quiet: %q", diagnostic.String())
		}
	}
}

func TestDefaultConfigError(t *testing.T) {
	profileFixture(t)
	t.Setenv("AWS_CONFIG_FILE", t.TempDir()) // Cannot read a directory as an INI file.
	var out bytes.Buffer
	if err := runDefault(&out, io.Discard, "", true, nil); err == nil || out.Len() != 0 {
		t.Fatalf("error = %v, stdout = %q", err, out.String())
	}
}
