package sso

import (
	"crypto/sha1" //nolint:gosec // Not used for security. The AWS SDK names SSO cache files by SHA-1 of the start URL.
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type rfc3339 time.Time

func (r rfc3339) String() string {
	return time.Time(r).Format(time.RFC3339)
}

func (r rfc3339) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}

func (r *rfc3339) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("unmarshalling RFC3339 timestamp: %w", err)
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fmt.Errorf("expected RFC3339 timestamp: %w", err)
	}
	*r = rfc3339(t)
	return nil
}

// token mirrors the JSON the AWS CLI writes to ~/.aws/sso/cache/<sha1>.json.
type token struct {
	AccessToken string  `json:"accessToken"`
	ExpiresAt   rfc3339 `json:"expiresAt"`
	Region      string  `json:"region,omitempty"`
	StartURL    string  `json:"startUrl,omitempty"`
}

func (t token) expired() bool {
	return time.Now().After(time.Time(t.ExpiresAt))
}

// InvalidTokenError means the cached SSO token is missing, unreadable, or expired.
type InvalidTokenError struct {
	Err error
}

func (e *InvalidTokenError) Unwrap() error { return e.Err }

func (e *InvalidTokenError) Error() string {
	const msg = "the SSO session has expired or is invalid"
	if e.Err == nil {
		return msg
	}
	return msg + ": " + e.Err.Error()
}

// cacheFileName returns the file name the AWS CLI uses for a start URL's token.
func cacheFileName(startURL string) string {
	h := sha1.New() //nolint:gosec // See package comment on sha1 import.
	h.Write([]byte(startURL))
	return strings.ToLower(hex.EncodeToString(h.Sum(nil))) + ".json"
}

func (loader Loader) tokenPath() (string, error) {
	dir := loader.tokenCacheDir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("finding home directory: %w", err)
		}
		dir = filepath.Join(home, ".aws", "sso", "cache")
	}
	return filepath.Join(dir, cacheFileName(loader.StartURL)), nil
}

func (loader Loader) loadToken() (token, error) {
	path, err := loader.tokenPath()
	if err != nil {
		return token{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return token{}, &InvalidTokenError{Err: errors.New("no cached token")}
		}
		return token{}, &InvalidTokenError{Err: err}
	}

	var t token
	if err := json.Unmarshal(data, &t); err != nil {
		return token{}, &InvalidTokenError{Err: err}
	}
	if t.AccessToken == "" {
		return token{}, &InvalidTokenError{Err: errors.New("cached token is empty")}
	}
	if t.expired() {
		return token{}, &InvalidTokenError{Err: errors.New("cached token has expired")}
	}
	return t, nil
}

func (loader Loader) writeToken(t token) error {
	path, err := loader.tokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating token cache directory: %w", err)
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
