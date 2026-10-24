package sso

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"golang.org/x/sync/errgroup"
	"gopkg.in/ini.v1"
)

// STSCredentials are short-lived credentials for one profile.
type STSCredentials struct {
	Name            string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

func (c STSCredentials) expired() bool {
	return time.Now().After(c.Expiration)
}

// updateINIFile replaces the credentials section for this profile. It refuses
// to touch a section that exists but is not managed by awss, or that awss
// manages for a different start URL.
func (c STSCredentials) updateINIFile(f *ini.File, startURL string) error {
	if sectionExists(f, c.Name) {
		if !isManagedSection(f, c.Name) {
			return fmt.Errorf("credentials for profile %q already exist and are not managed by awss", c.Name)
		}
		if other := otherStartURL(f, c.Name, startURL); other != "" {
			return fmt.Errorf("credentials for profile %q are managed by awss for start URL %s, not %s; use a --template that gives unique names, or delete the old credentials if you no longer use that start URL", c.Name, other, startURL)
		}
		f.DeleteSection(c.Name)
	}
	s := f.Section(c.Name)
	pairs := [][2]string{
		{managedKey, "true"},
		{"aws_access_key_id", c.AccessKeyID},
		{"aws_secret_access_key", c.SecretAccessKey},
		{"aws_session_token", c.SessionToken},
		{"expiration", c.Expiration.Format(time.RFC3339)},
		{"sso_start_url", startURL},
	}
	for _, kv := range pairs {
		if _, err := s.NewKey(kv[0], kv[1]); err != nil {
			return err
		}
	}
	return nil
}

func stsCredentialsFromINISection(s *ini.Section) STSCredentials {
	kv := s.KeysHash()
	exp, err := time.Parse(time.RFC3339, kv["expiration"])
	if err != nil {
		exp = time.Time{}
	}
	return STSCredentials{
		Name:            s.Name(),
		AccessKeyID:     kv["aws_access_key_id"],
		SecretAccessKey: kv["aws_secret_access_key"],
		SessionToken:    kv["aws_session_token"],
		Expiration:      exp,
	}
}

// updateCredentialsINI writes creds into the AWS credentials file and drops
// expired managed credentials for startURL. Unexpired ones are left alone.
func updateCredentialsINI(f *ini.File, creds []STSCredentials, startURL string) error {
	stale := managedSections(f, startURL)
	for _, c := range creds {
		if err := c.updateINIFile(f, startURL); err != nil {
			return err
		}
		delete(stale, c.Name)
	}
	for name, s := range stale {
		if stsCredentialsFromINISection(s).expired() {
			f.DeleteSection(name)
		}
	}
	return nil
}

func (loader Loader) fetchSTSCredentials(ctx context.Context, accessToken string, profiles []Profile) ([]STSCredentials, error) {
	g, ctx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	creds := make([]STSCredentials, 0, len(profiles))

	for _, p := range profiles {
		g.Go(func() error {
			out, err := loader.SSOClient.GetRoleCredentials(ctx, &sso.GetRoleCredentialsInput{
				AccessToken: aws.String(accessToken),
				AccountId:   aws.String(p.SSOAccountID),
				RoleName:    aws.String(p.SSORoleName),
			})
			if err != nil {
				return fmt.Errorf("fetching STS credentials for profile %q: %w", p.Name, err)
			}
			rc := out.RoleCredentials
			mu.Lock()
			creds = append(creds, STSCredentials{
				Name:            p.Name,
				AccessKeyID:     aws.ToString(rc.AccessKeyId),
				SecretAccessKey: aws.ToString(rc.SecretAccessKey),
				SessionToken:    aws.ToString(rc.SessionToken),
				Expiration:      time.UnixMilli(rc.Expiration),
			})
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return creds, nil
}
