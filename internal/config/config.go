package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Profile holds metadata for a single AWS profile.
type Profile struct {
	Name   string
	Region string
}

// DefaultConfigPath returns the path to the AWS config file,
// respecting the AWS_CONFIG_FILE environment variable.
func DefaultConfigPath() string {
	if v := os.Getenv("AWS_CONFIG_FILE"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "config")
}

// DefaultCredentialsPath returns the path to the AWS credentials file,
// respecting the AWS_SHARED_CREDENTIALS_FILE environment variable.
func DefaultCredentialsPath() string {
	if v := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aws", "credentials")
}

// ListProfiles returns a sorted, deduplicated list of AWS profile names
// found in the config and credentials files.
func ListProfiles(configPath, credentialsPath string) ([]string, error) {
	seen := make(map[string]struct{})

	if err := scanCredentialsProfiles(credentialsPath, seen); err != nil {
		return nil, err
	}
	if err := scanConfigProfiles(configPath, seen); err != nil {
		return nil, err
	}

	profiles := make([]string, 0, len(seen))
	for name := range seen {
		profiles = append(profiles, name)
	}
	sort.Strings(profiles)
	return profiles, nil
}

// scanCredentialsProfiles reads section headers from the credentials file.
// In credentials, every bare [name] section is a profile.
func scanCredentialsProfiles(path string, seen map[string]struct{}) error {
	return scanSections(path, func(header string) {
		// In credentials, sections with a space (like [profile X]) are invalid — skip them.
		if !strings.Contains(header, " ") && header != "" {
			seen[header] = struct{}{}
		}
	})
}

// scanConfigProfiles reads section headers from the config file.
// In config, [default] and [profile X] are profiles; [sso-session X] and [services X] are not.
func scanConfigProfiles(path string, seen map[string]struct{}) error {
	return scanSections(path, func(header string) {
		switch {
		case header == "default":
			seen["default"] = struct{}{}
		case strings.HasPrefix(header, "profile "):
			if name := strings.TrimSpace(header[len("profile "):]); name != "" {
				seen[name] = struct{}{}
			}
		}
	})
}

// scanSections reads an INI file and calls fn for each section header value
// (the text between [ and ], with inline comments stripped).
func scanSections(path string, fn func(string)) error {
	path = filepath.Clean(path)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // missing file is fine
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "[") {
			continue
		}
		// Strip inline comments before finding the closing bracket.
		// AWS config allows: [profile foo] ; some comment
		line = stripInlineComment(line)
		if idx := strings.Index(line, "]"); idx > 0 {
			fn(strings.TrimSpace(line[1:idx]))
		}
	}
	return scanner.Err()
}

// LoadProfile reads the config and credentials files and returns the
// named profile with its settings. Returns an error if the profile
// does not exist in either file.
func LoadProfile(configPath, credentialsPath, name string) (Profile, error) {
	p := Profile{Name: name}

	// Config file uses [default] or [profile X]; credentials uses [default] or [X].
	configSection := "profile " + name
	if name == "default" {
		configSection = "default"
	}
	credsSection := name

	foundConfig, region := scanProfileRegion(configPath, configSection)
	if region != "" {
		p.Region = region
	}

	foundCreds, credsRegion := scanProfileRegion(credentialsPath, credsSection)
	// Credentials region is used only if config didn't provide one.
	if p.Region == "" && credsRegion != "" {
		p.Region = credsRegion
	}

	if !foundConfig && !foundCreds {
		return Profile{}, fmt.Errorf("profile %q not found", name)
	}
	return p, nil
}

// scanProfileRegion reads an INI file and returns whether the target
// section exists and its region value (empty string if not set).
func scanProfileRegion(path, section string) (found bool, region string) {
	path = filepath.Clean(path)
	f, err := os.Open(path)
	if err != nil {
		return false, ""
	}
	defer f.Close()

	inSection := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// New section header — check if it's ours.
		if strings.HasPrefix(line, "[") {
			line = stripInlineComment(line)
			if idx := strings.Index(line, "]"); idx > 0 {
				header := strings.TrimSpace(line[1:idx])
				inSection = header == section
				if inSection {
					found = true
				}
			}
			continue
		}

		if !inSection {
			continue
		}

		// Skip comments and blank lines.
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}

		// Parse key = value.
		eqIdx := strings.Index(line, "=")
		if eqIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eqIdx])
		if key == "region" {
			val := strings.TrimSpace(line[eqIdx+1:])
			// Strip inline comments from the value.
			for _, ch := range []string{" #", " ;"} {
				if i := strings.Index(val, ch); i >= 0 {
					val = strings.TrimSpace(val[:i])
				}
			}
			region = val
		}
	}
	return found, region
}

// stripInlineComment removes inline comments (# or ;) that appear
// after the closing bracket of a section header.
func stripInlineComment(line string) string {
	// Find the closing bracket first — don't strip # or ; inside the brackets.
	if idx := strings.Index(line, "]"); idx >= 0 {
		return line[:idx+1]
	}
	// No closing bracket found; strip comments from the whole line
	// so the caller sees there's no valid section header.
	for _, ch := range []string{"#", ";"} {
		if i := strings.Index(line, ch); i >= 0 {
			line = line[:i]
		}
	}
	return line
}
