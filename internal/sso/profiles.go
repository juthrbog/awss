package sso

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/ini.v1"
)

// managedKey marks INI sections written by awss. Sections without it are
// never modified or deleted.
const managedKey = "awss_managed"

// Profile is one SSO profile as written to the AWS config file.
type Profile struct {
	Name         string
	SSOStartURL  string
	SSORegion    string
	SSOAccountID string
	SSORoleName  string
	Region       string
}

// configSectionName returns the INI section name for this profile. The default
// profile is [default]; every other profile is [profile <name>].
func (p Profile) configSectionName() string {
	if p.Name == "default" {
		return p.Name
	}
	return "profile " + p.Name
}

// updateINIFile creates or refreshes the section for this profile. It refuses
// to touch a section that exists but is not managed by awss.
func (p Profile) updateINIFile(f *ini.File) error {
	n := p.configSectionName()
	if sectionExists(f, n) && !isManagedSection(f, n) {
		return fmt.Errorf("profile %q already exists and is not managed by awss", p.Name)
	}

	s := f.Section(n)
	if err := setKey(s, managedKey, "true"); err != nil {
		return err
	}
	if err := setKey(s, "sso_start_url", p.SSOStartURL); err != nil {
		return err
	}
	if err := setKey(s, "sso_region", p.SSORegion); err != nil {
		return err
	}
	if err := setKey(s, "sso_account_id", p.SSOAccountID); err != nil {
		return err
	}
	if err := setKey(s, "sso_role_name", p.SSORoleName); err != nil {
		return err
	}
	// region is only seeded, never overwritten, so users can pin their own.
	return addKey(s, "region", p.Region)
}

// addKey creates key only if it does not exist yet.
func addKey(s *ini.Section, key, value string) error {
	if s.HasKey(key) {
		return nil
	}
	_, err := s.NewKey(key, value)
	return err
}

// setKey creates or overwrites key.
func setKey(s *ini.Section, key, value string) error {
	if s.HasKey(key) {
		s.Key(key).SetValue(value)
		return nil
	}
	_, err := s.NewKey(key, value)
	return err
}

func profileFromINISection(s *ini.Section) Profile {
	kv := s.KeysHash()
	return Profile{
		Name:         strings.TrimPrefix(s.Name(), "profile "),
		SSOStartURL:  kv["sso_start_url"],
		SSORegion:    kv["sso_region"],
		SSOAccountID: kv["sso_account_id"],
		SSORoleName:  kv["sso_role_name"],
		Region:       kv["region"],
	}
}

func (loader Loader) newProfile(name string, r Role) Profile {
	return Profile{
		Name:         name,
		SSOStartURL:  loader.StartURL,
		SSORegion:    loader.Region,
		SSOAccountID: r.Account.ID,
		SSORoleName:  r.Role,
		Region:       loader.Region,
	}
}

func (loader Loader) generateProfiles(ctx context.Context, accessToken string, opts LoginOptions) ([]Profile, error) {
	roles, err := loader.Roles(ctx, accessToken, opts.Accounts)
	if err != nil {
		return nil, err
	}

	var profiles []Profile
	for _, r := range roles {
		if !inFilter(r.Role, opts.Roles) {
			continue
		}
		name, err := r.profileName(opts.NameTemplate)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, loader.newProfile(name, r))
	}
	return profiles, nil
}
