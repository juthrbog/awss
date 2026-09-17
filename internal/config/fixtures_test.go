package config

import (
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/ini.v1"
)

func TestLocalAWSFixtures(t *testing.T) {
	configPath := filepath.Join("..", "..", "testdata", "aws", "config")
	credentialsPath := filepath.Join("..", "..", "testdata", "aws", "credentials")
	profiles, err := ListProfiles(configPath, credentialsPath)
	if err != nil {
		t.Fatal(err)
	}
	assertProfiles(t, profiles, []string{"cross-account", "default", "dev-only", "dev-sso", "production", "staging"})

	for name, region := range map[string]string{
		"default":       "us-west-2",
		"production":    "us-east-1",
		"staging":       "",
		"dev-only":      "ap-northeast-1",
		"dev-sso":       "eu-west-1",
		"cross-account": "ap-southeast-1",
	} {
		profile, err := LoadProfile(configPath, credentialsPath, name)
		if err != nil || profile.Region != region {
			t.Errorf("LoadProfile(%q) = %+v, %v; want region %q", name, profile, err, region)
		}
	}
}

// Both files must remain useful on their own, with multiple static profiles.
// Enforce exact, visibly fake values rather than exempting these files from
// secret scanning: an accidentally pasted real credential should still fail CI.
func TestLocalFixtureCredentialsArePlaceholders(t *testing.T) {
	for _, filename := range []string{"config", "credentials"} {
		t.Run(filename, func(t *testing.T) {
			path := filepath.Join("..", "..", "testdata", "aws", filename)
			file, err := ini.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			staticProfiles := 0
			for _, section := range file.Sections() {
				access := section.Key("aws_access_key_id").String()
				secret := section.Key("aws_secret_access_key").String()
				token := section.Key("aws_session_token").String()
				if access == "" && secret == "" && token == "" {
					continue
				}
				staticProfiles++
				name := strings.TrimPrefix(section.Name(), "profile ")
				suffix := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
				if access != "FAKE_KEY_"+suffix || secret != "FAKE_SECRET_"+suffix || token != "" {
					// Never echo a potentially real credential into CI logs.
					t.Errorf("section %q must use its documented FAKE_KEY_/FAKE_SECRET_ placeholders and no session token", section.Name())
				}
			}
			if staticProfiles < 2 {
				t.Errorf("found %d static profiles; need at least 2", staticProfiles)
			}
		})
	}
}
