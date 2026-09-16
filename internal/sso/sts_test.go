package sso

import (
	"testing"
	"time"

	"gopkg.in/ini.v1"
)

func TestUpdateCredentialsINI(t *testing.T) {
	f := ini.Empty()
	url := "https://example.awsapps.com/start"

	// Expired managed credentials for this start URL: should be removed.
	old := f.Section("old")
	mustKey(t, old, managedKey, "true")
	mustKey(t, old, "sso_start_url", url)
	mustKey(t, old, "expiration", time.Now().Add(-time.Hour).Format(time.RFC3339))

	// Still-valid managed credentials not in this batch: should be kept.
	live := f.Section("live")
	mustKey(t, live, managedKey, "true")
	mustKey(t, live, "sso_start_url", url)
	mustKey(t, live, "expiration", time.Now().Add(time.Hour).Format(time.RFC3339))

	// User-owned credentials: never touched.
	mustKey(t, f.Section("mine"), "aws_access_key_id", "FAKE")

	creds := []STSCredentials{{
		Name:            "fresh",
		AccessKeyID:     "FAKE_ID",
		SecretAccessKey: "FAKE_SECRET",
		SessionToken:    "FAKE_TOKEN",
		Expiration:      time.Now().Add(time.Hour),
	}}
	if err := updateCredentialsINI(f, creds, url); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sectionExists(f, "old") {
		t.Error("expired managed credentials should be removed")
	}
	if !sectionExists(f, "live") {
		t.Error("unexpired managed credentials should be kept")
	}
	if !sectionExists(f, "mine") {
		t.Error("unmanaged credentials must be kept")
	}
	s := f.Section("fresh")
	if s.Key("aws_session_token").Value() != "FAKE_TOKEN" {
		t.Error("new credentials should be written")
	}
	if s.Key(managedKey).Value() != "true" {
		t.Error("new credentials should be marked managed")
	}
}

func TestSTSCredentials_RejectsUnmanaged(t *testing.T) {
	f := ini.Empty()
	mustKey(t, f.Section("mine"), "aws_access_key_id", "FAKE")

	c := STSCredentials{Name: "mine"}
	if err := c.updateINIFile(f, "https://example.awsapps.com/start"); err == nil {
		t.Fatal("expected error writing over unmanaged credentials")
	}
}
