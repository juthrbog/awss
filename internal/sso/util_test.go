package sso

import (
	"path/filepath"
	"testing"

	"gopkg.in/ini.v1"
)

func TestIsKeyTrue(t *testing.T) {
	f := ini.Empty()
	s := f.Section("test")
	s.NewKey("enabled", "true")
	s.NewKey("disabled", "false")

	if !isKeyTrue(s, "enabled") {
		t.Error("enabled=true should be true")
	}
	if isKeyTrue(s, "disabled") {
		t.Error("disabled=false should be false")
	}
	if isKeyTrue(s, "missing") {
		t.Error("missing key should be false")
	}
}

func TestIsManagedSection(t *testing.T) {
	f := ini.Empty()
	f.Section("profile managed").NewKey(managedKey, "true")
	f.Section("profile plain").NewKey("sso_start_url", "https://example.awsapps.com/start")

	if !isManagedSection(f, "profile managed") {
		t.Error("section with awss_managed=true should be managed")
	}
	if isManagedSection(f, "profile plain") {
		t.Error("section without awss_managed should not be managed")
	}
}

func TestManagedSections_FiltersByStartURL(t *testing.T) {
	f := ini.Empty()
	url := "https://example.awsapps.com/start"

	s1 := f.Section("profile managed")
	s1.NewKey(managedKey, "true")
	s1.NewKey("sso_start_url", url)

	s2 := f.Section("profile other-url")
	s2.NewKey(managedKey, "true")
	s2.NewKey("sso_start_url", "https://other.awsapps.com/start")

	f.Section("profile unmanaged").NewKey("sso_start_url", url)

	got := managedSections(f, url)
	if len(got) != 1 {
		t.Fatalf("got %d managed sections, want 1", len(got))
	}
	if _, ok := got["profile managed"]; !ok {
		t.Error("expected 'profile managed' in result")
	}
}

func TestLoadINI_CreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config")
	f, err := loadINI(path)
	if err != nil {
		t.Fatalf("loadINI: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil ini file")
	}
}
