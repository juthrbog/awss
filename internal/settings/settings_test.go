package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPath_Precedence(t *testing.T) {
	t.Setenv("AWSS_CONFIG", "/tmp/custom.yaml")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	if got := Path(); got != "/tmp/custom.yaml" {
		t.Errorf("AWSS_CONFIG should win, got %q", got)
	}

	t.Setenv("AWSS_CONFIG", "")
	if got := Path(); got != filepath.Join("/tmp/xdg", "awss", "config.yaml") {
		t.Errorf("XDG_CONFIG_HOME should be used, got %q", got)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	if got := Path(); !strings.HasSuffix(got, filepath.Join(".config", "awss", "config.yaml")) {
		t.Errorf("expected ~/.config/awss/config.yaml, got %q", got)
	}
}

func TestLoad_MissingFileIsEmpty(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.DefaultOrg != "" || len(s.Orgs) != 0 {
		t.Errorf("expected empty settings, got %+v", s)
	}
}

func TestLoad_ParsesOrgs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	body := `
default_org: example
orgs:
  example:
    start_url: https://example.awsapps.com/start
    sso_region: eu-west-1
    template: '{{.Account.Name}}-{{.Role}}'
  other:
    start_url: https://other.awsapps.com/start
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.DefaultOrg != "example" {
		t.Errorf("DefaultOrg = %q", s.DefaultOrg)
	}
	if got := s.Orgs["example"].SSORegion; got != "eu-west-1" {
		t.Errorf("sso_region = %q", got)
	}
	if got := s.Orgs["other"].StartURL; got != "https://other.awsapps.com/start" {
		t.Errorf("other start_url = %q", got)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("orgs: [not a map"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestResolve(t *testing.T) {
	two := Settings{Orgs: map[string]Org{
		"a": {StartURL: "https://a.awsapps.com/start"},
		"b": {StartURL: "https://b.awsapps.com/start", SSORegion: "us-west-2"},
	}}
	one := Settings{Orgs: map[string]Org{"only": {StartURL: "https://only.awsapps.com/start"}}}
	withDefault := Settings{DefaultOrg: "b", Orgs: two.Orgs}

	cases := []struct {
		name     string
		s        Settings
		org, url string
		want     string
		wantErr  bool
	}{
		{"start-url wins", two, "", "https://x.awsapps.com/start", "https://x.awsapps.com/start", false},
		{"start-url with org is an error", two, "a", "https://x.awsapps.com/start", "", true},
		{"named org", two, "a", "", "https://a.awsapps.com/start", false},
		{"unknown org", two, "zzz", "", "", true},
		{"default_org", withDefault, "", "", "https://b.awsapps.com/start", false},
		{"single org without default", one, "", "", "https://only.awsapps.com/start", false},
		{"two orgs without default", two, "", "", "", true},
		{"nothing configured", Settings{}, "", "", "", true},
		{"org without start_url", Settings{Orgs: map[string]Org{"x": {}}}, "x", "", "", true},
	}
	for _, c := range cases {
		got, err := c.s.Resolve(c.org, c.url)
		if c.wantErr {
			if err == nil {
				t.Errorf("%s: expected error, got %+v", c.name, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if got.StartURL != c.want {
			t.Errorf("%s: StartURL = %q, want %q", c.name, got.StartURL, c.want)
		}
	}
}
