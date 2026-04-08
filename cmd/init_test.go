package cmd

import (
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
		`command "/usr/local/bin/awss" select "$@"`,
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
	} {
		if !strings.Contains(out, want) {
			t.Errorf("fish output missing %q", want)
		}
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
