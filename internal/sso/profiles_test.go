package sso

import (
	"testing"

	"gopkg.in/ini.v1"
)

func testProfile() Profile {
	return Profile{
		Name:         "test-profile",
		SSOStartURL:  "https://example.awsapps.com/start",
		SSORegion:    "us-east-1",
		SSOAccountID: "123456789012",
		SSORoleName:  "AdminRole",
		Region:       "us-east-1",
	}
}

func TestUpdateINIFile_FreshSection(t *testing.T) {
	f := ini.Empty()
	p := testProfile()

	if err := p.updateINIFile(f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := f.Section("profile test-profile")
	if got := s.Key(managedKey).Value(); got != "true" {
		t.Errorf("%s = %q, want true", managedKey, got)
	}
	for k, want := range map[string]string{
		"sso_start_url":  p.SSOStartURL,
		"sso_region":     p.SSORegion,
		"sso_account_id": p.SSOAccountID,
		"sso_role_name":  p.SSORoleName,
		"region":         p.Region,
	} {
		if got := s.Key(k).Value(); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestUpdateINIFile_KeepsUserRegion(t *testing.T) {
	f := ini.Empty()
	s := f.Section("profile test-profile")
	mustKey(t, s, managedKey, "true")
	mustKey(t, s, "region", "eu-west-1")

	if err := testProfile().updateINIFile(f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := f.Section("profile test-profile").Key("region").Value(); got != "eu-west-1" {
		t.Errorf("region = %q, want user value eu-west-1 to be preserved", got)
	}
}

func TestUpdateINIFile_RejectsUnmanagedSection(t *testing.T) {
	f := ini.Empty()
	mustKey(t, f.Section("profile test-profile"), "sso_start_url", "https://example.awsapps.com/start")

	if err := testProfile().updateINIFile(f); err == nil {
		t.Fatal("expected error when updating unmanaged section")
	}
}

func TestUpdateINIFile_DefaultProfileSectionName(t *testing.T) {
	f := ini.Empty()
	p := testProfile()
	p.Name = "default"

	if err := p.updateINIFile(f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sectionExists(f, "default") {
		t.Error("expected [default] section, not [profile default]")
	}
}

func TestUpdateConfigINI_RemovesStaleManagedSections(t *testing.T) {
	f := ini.Empty()
	url := "https://example.awsapps.com/start"

	stale := f.Section("profile old-role")
	mustKey(t, stale, managedKey, "true")
	mustKey(t, stale, "sso_start_url", url)

	otherOrg := f.Section("profile other-org")
	mustKey(t, otherOrg, managedKey, "true")
	mustKey(t, otherOrg, "sso_start_url", "https://other.awsapps.com/start")

	mustKey(t, f.Section("profile mine"), "region", "us-west-2")

	if err := updateConfigINI(f, []Profile{testProfile()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sectionExists(f, "profile old-role") {
		t.Error("stale managed section for same start URL should be removed")
	}
	if !sectionExists(f, "profile other-org") {
		t.Error("managed section for a different start URL must be kept")
	}
	if !sectionExists(f, "profile mine") {
		t.Error("unmanaged section must be kept")
	}
	if !sectionExists(f, "profile test-profile") {
		t.Error("new profile should be written")
	}
}

func TestProfileName_Template(t *testing.T) {
	r := Role{
		Account:        Account{ID: "123456789012", Name: "example-prod"},
		ShortAccountID: "9012",
		Role:           "AdminRole",
	}
	cases := []struct {
		tmpl string
		want string
	}{
		{"", "example-prod-AdminRole"},
		{"{{.Account.Name}}-{{.Role}}", "example-prod-AdminRole"},
		{`{{.Account.Name | trimPrefix "example-"}}-{{.Role}}`, "prod-AdminRole"},
		{`{{.Account.Name | replace "-" "_"}}`, "example_prod"},
		{"{{.ShortAccountID}}", "9012"},
	}
	for _, c := range cases {
		got, err := r.profileName(c.tmpl)
		if err != nil {
			t.Errorf("profileName(%q) error: %v", c.tmpl, err)
			continue
		}
		if got != c.want {
			t.Errorf("profileName(%q) = %q, want %q", c.tmpl, got, c.want)
		}
	}
}

func TestProfileName_InvalidTemplate(t *testing.T) {
	if _, err := (Role{}).profileName("{{.Nope"); err == nil {
		t.Fatal("expected error for unparseable template")
	}
}

func TestShortAccount(t *testing.T) {
	cases := map[string]string{
		"123456789012": "9012",
		"12":           "12",
		"":             "",
	}
	for in, want := range cases {
		if got := shortAccount(in); got != want {
			t.Errorf("shortAccount(%q) = %q, want %q", in, got, want)
		}
	}
}
