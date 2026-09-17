package state

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPreviousRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", dir)
	if _, err := Previous(); err == nil {
		t.Fatal("missing history should return an actionable error")
	}
	for _, name := range []string{"default", "production", "name with spaces", "trailing space "} {
		if err := SavePrevious(name, "next"); err != nil {
			t.Fatal(err)
		}
		got, err := Previous()
		if err != nil || got != name {
			t.Fatalf("Previous() = %q, %v; want %q", got, err, name)
		}
	}
	path := filepath.Join(dir, "awss", "previous")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Errorf("history permissions = %o", info.Mode().Perm())
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 || entries[0].Name() != "previous" {
		t.Fatalf("temporary files left behind: %v, %v", entries, err)
	}
}

func TestPreviousHomeFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CACHE_HOME", "")
	if err := SavePrevious("default", "production"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Clean(filepath.Join(home, ".cache", "awss", "previous")))
	if err != nil || string(data) != "default\n" {
		t.Fatalf("fallback history = %q, %v", data, err)
	}
}

func TestNoOpPreservesHistory(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if err := SavePrevious("default", "production"); err != nil {
		t.Fatal(err)
	}
	for _, current := range []string{"", "production"} {
		if err := SavePrevious(current, "production"); err != nil {
			t.Fatal(err)
		}
		if got, err := Previous(); got != "default" || err != nil {
			t.Fatalf("no-op destroyed history: %q, %v", got, err)
		}
	}
}

func TestStateErrors(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", file)
	if err := SavePrevious("default", "production"); err == nil {
		t.Fatal("expected write error")
	}
	if _, err := Previous(); err == nil {
		t.Fatal("expected read error")
	}
}
