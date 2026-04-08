package config

import (
	"path/filepath"
	"testing"
)

func TestLoadProfile_ValidProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[default]
region = us-west-2

[profile production]
output = json
`)
	credsPath := filepath.Join(dir, "credentials")

	p, err := LoadProfile(configPath, credsPath, "production")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "production" {
		t.Errorf("Name = %q, want %q", p.Name, "production")
	}
	if p.Region != "" {
		t.Errorf("Region = %q, want empty", p.Region)
	}
}

func TestLoadProfile_ValidProfileWithRegion(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[profile production]
region = us-east-1
output = json
`)
	credsPath := filepath.Join(dir, "credentials")

	p, err := LoadProfile(configPath, credsPath, "production")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "production" {
		t.Errorf("Name = %q, want %q", p.Name, "production")
	}
	if p.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", p.Region, "us-east-1")
	}
}

func TestLoadProfile_DefaultProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[default]
region = us-west-2
`)
	credsPath := writeFile(t, dir, "credentials", `
[default]
aws_access_key_id = AKIA...
`)

	p, err := LoadProfile(configPath, credsPath, "default")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "default" {
		t.Errorf("Name = %q, want %q", p.Name, "default")
	}
	if p.Region != "us-west-2" {
		t.Errorf("Region = %q, want %q", p.Region, "us-west-2")
	}
}

func TestLoadProfile_MissingProfile(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[default]
region = us-west-2
`)
	credsPath := filepath.Join(dir, "credentials")

	_, err := LoadProfile(configPath, credsPath, "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing profile, got nil")
	}
}

func TestLoadProfile_ConfigRegionTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[profile shared]
region = us-east-1
`)
	credsPath := writeFile(t, dir, "credentials", `
[shared]
region = eu-west-1
`)

	p, err := LoadProfile(configPath, credsPath, "shared")
	if err != nil {
		t.Fatal(err)
	}
	if p.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q (config should take precedence)", p.Region, "us-east-1")
	}
}

func TestLoadProfile_RegionFromCredentials(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[profile nocreds]
output = json
`)
	credsPath := writeFile(t, dir, "credentials", `
[nocreds]
region = ap-southeast-1
`)

	p, err := LoadProfile(configPath, credsPath, "nocreds")
	if err != nil {
		t.Fatal(err)
	}
	if p.Region != "ap-southeast-1" {
		t.Errorf("Region = %q, want %q", p.Region, "ap-southeast-1")
	}
}

func TestLoadProfile_InlineCommentInRegion(t *testing.T) {
	dir := t.TempDir()
	configPath := writeFile(t, dir, "config", `
[profile commented]
region = us-east-1 # primary region
`)
	credsPath := filepath.Join(dir, "credentials")

	p, err := LoadProfile(configPath, credsPath, "commented")
	if err != nil {
		t.Fatal(err)
	}
	if p.Region != "us-east-1" {
		t.Errorf("Region = %q, want %q", p.Region, "us-east-1")
	}
}

func TestLoadProfile_OnlyInCredentials(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config")
	credsPath := writeFile(t, dir, "credentials", `
[myprofile]
aws_access_key_id = AKIA...
region = us-west-1
`)

	p, err := LoadProfile(configPath, credsPath, "myprofile")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "myprofile" {
		t.Errorf("Name = %q, want %q", p.Name, "myprofile")
	}
	if p.Region != "us-west-1" {
		t.Errorf("Region = %q, want %q", p.Region, "us-west-1")
	}
}
