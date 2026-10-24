package sso

import "gopkg.in/ini.v1"

// updateConfigINI writes profiles into the AWS config file, removing managed
// sections for the same start URL that no longer correspond to a role.
func updateConfigINI(f *ini.File, profiles []Profile) error {
	if len(profiles) == 0 {
		return nil
	}

	stale := managedSections(f, profiles[0].SSOStartURL)
	for _, p := range profiles {
		if err := p.updateINIFile(f); err != nil {
			return err
		}
		delete(stale, p.configSectionName())
	}
	for name := range stale {
		f.DeleteSection(name)
	}
	return nil
}
