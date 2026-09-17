package sso

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCacheFileName_MatchesAWSCLI(t *testing.T) {
	// SHA-1 of the start URL, lowercase hex, .json. Same scheme as the AWS CLI.
	got := cacheFileName("https://example.awsapps.com/start")
	if len(got) != 40+len(".json") {
		t.Fatalf("unexpected cache file name %q", got)
	}
	if filepath.Ext(got) != ".json" {
		t.Errorf("expected .json extension, got %q", got)
	}
}

func TestToken_RoundTrip(t *testing.T) {
	l := Loader{StartURL: "https://example.awsapps.com/start", tokenCacheDir: t.TempDir()}
	want := token{
		AccessToken: "abc",
		ExpiresAt:   rfc3339(time.Now().Add(time.Hour).UTC().Truncate(time.Second)),
		Region:      "us-east-1",
		StartURL:    l.StartURL,
	}
	if err := l.writeToken(want); err != nil {
		t.Fatalf("writeToken: %v", err)
	}

	path, _ := l.tokenPath()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	// Windows uses ACLs, not POSIX permission bits; still test the round trip there.
	if perm := info.Mode().Perm(); runtime.GOOS != "windows" && perm != 0o600 {
		t.Errorf("token file mode = %o, want 600", perm)
	}

	got, err := l.loadToken()
	if err != nil {
		t.Fatalf("loadToken: %v", err)
	}
	if got.AccessToken != want.AccessToken || got.StartURL != want.StartURL {
		t.Errorf("loadToken = %+v, want %+v", got, want)
	}
}

func TestLoadToken_Missing(t *testing.T) {
	l := Loader{StartURL: "https://example.awsapps.com/start", tokenCacheDir: t.TempDir()}
	_, err := l.loadToken()
	var ite *InvalidTokenError
	if !errors.As(err, &ite) {
		t.Fatalf("expected InvalidTokenError, got %v", err)
	}
}

func TestLoadToken_Expired(t *testing.T) {
	l := Loader{StartURL: "https://example.awsapps.com/start", tokenCacheDir: t.TempDir()}
	expired := token{
		AccessToken: "abc",
		ExpiresAt:   rfc3339(time.Now().Add(-time.Minute)),
	}
	if err := l.writeToken(expired); err != nil {
		t.Fatalf("writeToken: %v", err)
	}
	_, err := l.loadToken()
	var ite *InvalidTokenError
	if !errors.As(err, &ite) {
		t.Fatalf("expected InvalidTokenError for expired token, got %v", err)
	}
}
