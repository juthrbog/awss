package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestListProfiles_BasicConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[default]
region = us-west-2

[profile production]
region = us-east-1

[profile staging]
region = eu-west-1
`)
	credsPath := filepath.Join(dir, "credentials") // doesn't exist — should be fine

	profiles, err := ListProfiles(configPath, credsPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"default", "production", "staging"}
	assertProfiles(t, profiles, want)
}

func TestListProfiles_CredentialsFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	credsPath := writeFile(t, dir, "credentials", `
[default]
aws_access_key_id = AKIA...

[production]
aws_access_key_id = AKIA...
`)

	profiles, err := ListProfiles(configPath, credsPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"default", "production"}
	assertProfiles(t, profiles, want)
}

func TestListProfiles_BothFiles_Deduplicated(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[default]
region = us-west-2

[profile production]
region = us-east-1
`)
	credsPath := writeFile(t, dir, "credentials", `
[default]
aws_access_key_id = AKIA...

[production]
aws_access_key_id = AKIA...

[dev-only]
aws_access_key_id = AKIA...
`)

	profiles, err := ListProfiles(configPath, credsPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"default", "dev-only", "production"}
	assertProfiles(t, profiles, want)
}

func TestListProfiles_SkipsSSOSessionAndServices(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[default]
region = us-west-2

[profile dev-sso]
sso_session = my-sso
sso_account_id = 111122223333
sso_role_name = ReadOnly

[sso-session my-sso]
sso_start_url = https://myorg.awsapps.com/start
sso_region = us-east-1

[services my-services]
endpoint_url = http://localhost:4566
`)
	credsPath := filepath.Join(dir, "credentials")

	profiles, err := ListProfiles(configPath, credsPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"default", "dev-sso"}
	assertProfiles(t, profiles, want)
}

func TestListProfiles_InlineComments(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[default] ; this is a comment
region = us-west-2

[profile production] # another comment
region = us-east-1
`)
	credsPath := filepath.Join(dir, "credentials")

	profiles, err := ListProfiles(configPath, credsPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"default", "production"}
	assertProfiles(t, profiles, want)
}

func TestListProfiles_AssumeRoleProfiles(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[profile base]
region = us-east-1

[profile cross-account]
role_arn = arn:aws:iam::123456789012:role/Admin
source_profile = base
region = eu-west-1
`)
	credsPath := writeFile(t, dir, "credentials", `
[base]
aws_access_key_id = AKIA...
aws_secret_access_key = secret...
`)

	profiles, err := ListProfiles(configPath, credsPath)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"base", "cross-account"}
	assertProfiles(t, profiles, want)
}

func TestListProfiles_BothFilesMissing(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	credsPath := filepath.Join(dir, "credentials")

	profiles, err := ListProfiles(configPath, credsPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected empty list, got %v", profiles)
	}
}

func TestListProfiles_CredentialsSectionsWithSpacesSkipped(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	credsPath := writeFile(t, dir, "credentials", `
[default]
aws_access_key_id = AKIA...

[profile invalid]
aws_access_key_id = AKIA...

[valid-name]
aws_access_key_id = AKIA...
`)

	profiles, err := ListProfiles(configPath, credsPath)
	if err != nil {
		t.Fatal(err)
	}
	// [profile invalid] should be skipped in credentials file
	want := []string{"default", "valid-name"}
	assertProfiles(t, profiles, want)
}

func TestDefaultPaths_EnvOverride(t *testing.T) {
	t.Setenv("AWS_CONFIG_FILE", "/tmp/custom-config")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/tmp/custom-creds")

	if got := DefaultConfigPath(); got != "/tmp/custom-config" {
		t.Errorf("DefaultConfigPath() = %q, want /tmp/custom-config", got)
	}
	if got := DefaultCredentialsPath(); got != "/tmp/custom-creds" {
		t.Errorf("DefaultCredentialsPath() = %q, want /tmp/custom-creds", got)
	}
}

func assertProfiles(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d profiles %v, want %d profiles %v", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("profile[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
