package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
