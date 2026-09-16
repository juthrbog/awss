package sso

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

// loadINI opens path, creating it with mode 0600 if it does not exist.
func loadINI(path string) (*ini.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		return nil, err
	}
	return ini.Load(path)
}

func sectionExists(f *ini.File, name string) bool {
	_, err := f.GetSection(name)
	return err == nil
}

func isKeyTrue(s *ini.Section, key string) bool {
	return s.HasKey(key) && s.Key(key).Value() == "true"
}

func isManagedSection(f *ini.File, name string) bool {
	return isKeyTrue(f.Section(name), managedKey)
}

// managedSections returns every awss-managed section whose sso_start_url
// matches startURL, keyed by section name.
func managedSections(f *ini.File, startURL string) map[string]*ini.Section {
	ms := make(map[string]*ini.Section)
	for _, s := range f.Sections() {
		if isKeyTrue(s, managedKey) && s.Key("sso_start_url").Value() == startURL {
			ms[s.Name()] = s
		}
	}
	return ms
}
