package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func overrideFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "files with spaces ' and $signs")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config")
	credentialsPath := filepath.Join(dir, "credentials")
	if err := os.WriteFile(configPath, []byte("[profile custom]\nregion = eu-west-3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credentialsPath, []byte("[custom-credentials]\nregion = ap-northeast-1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return configPath, credentialsPath
}

func TestFileFlagsListAndPrecedence(t *testing.T) {
	profileFixture(t)
	configPath, credentialsPath := overrideFixture(t)
	originalConfig, originalCredentials := os.Getenv("AWS_CONFIG_FILE"), os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	for _, args := range [][]string{
		{"--config-file", configPath, "--credentials-file", credentialsPath, "list"},
		{"list", "--config-file=" + configPath, "--credentials-file=" + credentialsPath},
		{"--config-file", configPath, "--credentials-file", credentialsPath},
	} {
		out, err := executeRoot(args...)
		if err != nil || out != "custom\ncustom-credentials\n" {
			t.Fatalf("%v: %q, %v", args, out, err)
		}
	}
	// Each flag overrides only its corresponding environment variable.
	out, err := executeRoot("list", "--config-file", configPath)
	if err != nil || out != "credentials-only\ncustom\nproduction\n" {
		t.Fatalf("config-only override: %q, %v", out, err)
	}
	out, err = executeRoot("list", "--credentials-file", credentialsPath)
	if err != nil || out != "custom-credentials\ndefault\nproduction\nstaging\n" {
		t.Fatalf("credentials-only override: %q, %v", out, err)
	}
	if os.Getenv("AWS_CONFIG_FILE") != originalConfig || os.Getenv("AWS_SHARED_CREDENTIALS_FILE") != originalCredentials {
		t.Fatal("flags must not mutate the process environment")
	}
}

func TestFileFlagsSelect(t *testing.T) {
	profileFixture(t)
	configPath, credentialsPath := overrideFixture(t)
	for _, shell := range []string{"bash", "zsh", "fish"} {
		for _, subcommand := range []bool{false, true} {
			args := []string{"--config-file", configPath, "--credentials-file", credentialsPath, "--shell", shell}
			if subcommand {
				args = append(args, "select")
			}
			args = append(args, "custom")
			out, err := executeRoot(args...)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{quoteShellValue(configPath, shell), quoteShellValue(credentialsPath, shell), "AWS_PROFILE", "custom", "eu-west-3"} {
				if !strings.Contains(out, want) {
					t.Errorf("output %q missing %q", out, want)
				}
			}
		}
	}
	out, err := executeRoot("--config-file", configPath, "custom")
	if err != nil || !strings.Contains(out, "AWS_CONFIG_FILE=") || strings.Contains(out, "AWS_SHARED_CREDENTIALS_FILE=") {
		t.Fatalf("only explicit paths should be exported: %q, %v", out, err)
	}
}

func TestFileFlagsPicker(t *testing.T) {
	profileFixture(t)
	configPath, credentialsPath := overrideFixture(t)
	cmd := newRootCommand()
	if err := cmd.ParseFlags([]string{"--config-file", configPath, "--credentials-file", credentialsPath}); err != nil {
		t.Fatal(err)
	}
	files, err := resolveAWSFiles(cmd)
	if err != nil {
		t.Fatal(err)
	}
	for _, selected := range []string{"custom", ""} {
		var out, diagnostic bytes.Buffer
		err := files.runDefault(&out, &diagnostic, "bash", true, func(names []string, _ string) (string, error) {
			if !reflect.DeepEqual(names, []string{"custom", "custom-credentials"}) {
				t.Fatalf("picker profiles = %v", names)
			}
			return selected, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if selected == "" && out.Len() != 0 {
			t.Fatalf("cancellation emitted exports: %q", out.String())
		}
		if selected != "" && !strings.Contains(out.String(), "export AWS_CONFIG_FILE=") {
			t.Fatalf("selection did not export file override: %q", out.String())
		}
	}
}

func TestFileFlagsCompletion(t *testing.T) {
	profileFixture(t)
	configPath, credentialsPath := overrideFixture(t)
	for _, args := range [][]string{
		{"__complete", "--config-file", configPath, "--credentials-file", credentialsPath, "custom"},
		{"--config-file", configPath, "--credentials-file", credentialsPath, "__complete", "custom"},
		{"__complete", "select", "--config-file", configPath, "--credentials-file", credentialsPath, "custom"},
	} {
		out, err := executeRoot(args...)
		if err != nil || out != "custom\ncustom-credentials\n:4\n" {
			t.Fatalf("%v: %q, %v", args, out, err)
		}
	}
	for _, flag := range []string{"--config-file", "--credentials-file"} {
		out, err := executeRoot("__complete", flag, "")
		if err != nil || out != ":0\n" {
			t.Fatalf("%s must allow filename completion: %q, %v", flag, out, err)
		}
	}
}

func TestFileFlagsValidationAndRelativePaths(t *testing.T) {
	profileFixture(t)
	for _, args := range [][]string{
		{"--config-file=", "production"}, {"list", "--credentials-file="},
		{"select", "production", "--config-file="},
	} {
		out, err := executeRoot(args...)
		if err == nil || out != "" || !strings.Contains(err.Error(), "non-empty path") {
			t.Fatalf("%v: %q, %v", args, out, err)
		}
	}
	out, err := executeRoot("--config-file", "../testdata/aws/config", "production")
	absolute, absErr := filepath.Abs("../testdata/aws/config")
	if err != nil || absErr != nil || !strings.Contains(out, quoteShellValue(absolute, "")) {
		t.Fatalf("relative paths must be exported as absolute: %q, %v, %v", out, err, absErr)
	}
}
