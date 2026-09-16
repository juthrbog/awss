// Package settings reads the awss config file, which holds named Identity
// Center organizations so users do not have to pass a start URL every time.
package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Org is one Identity Center instance a user can log in to.
type Org struct {
	// StartURL is the Identity Center portal URL, e.g. https://example.awsapps.com/start.
	StartURL string `yaml:"start_url"`
	// SSORegion is the region of the Identity Center instance.
	SSORegion string `yaml:"sso_region,omitempty"`
	// Template is the profile name template for this org.
	Template string `yaml:"template,omitempty"`
}

// Settings is the on-disk awss config.
type Settings struct {
	// DefaultOrg names the entry in Orgs used when --org is not given.
	DefaultOrg string `yaml:"default_org,omitempty"`
	// Orgs maps a short name to an Org.
	Orgs map[string]Org `yaml:"orgs,omitempty"`
}

// Path returns the awss config file location. AWSS_CONFIG wins, then
// $XDG_CONFIG_HOME/awss/config.yaml, then ~/.config/awss/config.yaml.
func Path() string {
	if v := os.Getenv("AWSS_CONFIG"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "awss", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "awss", "config.yaml")
}

// Load reads the settings file at path. A missing file yields empty settings.
func Load(path string) (Settings, error) {
	var s Settings
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return s, fmt.Errorf("reading %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("parsing %s: %w", path, err)
	}
	return s, nil
}

// Resolve picks the Org to log in to.
//
// startURL, when set, wins and describes an ad-hoc org. Otherwise orgName is
// looked up in s.Orgs. With neither, DefaultOrg is used, and if that is unset
// and exactly one org is configured, that one is used.
func (s Settings) Resolve(orgName, startURL string) (Org, error) {
	if startURL != "" {
		if orgName != "" {
			return Org{}, errors.New("--start-url and --org cannot be combined")
		}
		return Org{StartURL: startURL}, nil
	}

	if orgName == "" {
		orgName = s.DefaultOrg
	}
	if orgName == "" && len(s.Orgs) == 1 {
		for name := range s.Orgs {
			orgName = name
		}
	}
	if orgName == "" {
		return Org{}, fmt.Errorf("no SSO start URL: pass --start-url or --org, or set default_org in %s", Path())
	}

	org, ok := s.Orgs[orgName]
	if !ok {
		return Org{}, fmt.Errorf("org %q is not defined in %s", orgName, Path())
	}
	if org.StartURL == "" {
		return Org{}, fmt.Errorf("org %q has no start_url in %s", orgName, Path())
	}
	return org, nil
}
