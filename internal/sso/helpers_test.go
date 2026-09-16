package sso

import (
	"testing"

	"gopkg.in/ini.v1"
)

// mustKey sets key=value on s and fails the test if ini rejects it.
func mustKey(t *testing.T, s *ini.Section, key, value string) {
	t.Helper()
	if _, err := s.NewKey(key, value); err != nil {
		t.Fatalf("NewKey(%q, %q): %v", key, value, err)
	}
}
