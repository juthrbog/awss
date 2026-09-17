// Package state stores the previously selected profile across shell sessions.
package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func previousPath() (string, error) {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		// Use the XDG fallback even on macOS, not ~/Library/Caches.
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "awss", "previous"), nil
}

// Previous returns the last active profile recorded before a switch.
func Previous() (string, error) {
	path, err := previousPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("no previous profile saved; switch profiles first")
	}
	if err != nil {
		return "", fmt.Errorf("reading previous profile: %w", err)
	}
	name := strings.TrimSuffix(string(data), "\n")
	if name == "" {
		return "", errors.New("no previous profile saved; switch profiles first")
	}
	return name, nil
}

// SavePrevious atomically replaces the saved profile. An unset AWS_PROFILE or
// reselecting the same profile must not destroy useful history.
func SavePrevious(current, next string) error {
	if current == "" || current == next {
		return nil
	}
	path, err := previousPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating profile cache: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".previous-*")
	if err != nil {
		return fmt.Errorf("creating previous-profile file: %w", err)
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString(current + "\n"); err != nil {
		_ = file.Close() // Preserve the write error.
		return fmt.Errorf("writing previous profile: %w", err)
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return fmt.Errorf("saving previous profile: %w", err)
	}
	return nil
}
