package sso

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	awssso "github.com/aws/aws-sdk-go-v2/service/sso"
)

type failingSTSClient struct{ filePathSSOClient }

func (failingSTSClient) GetRoleCredentials(context.Context, *awssso.GetRoleCredentialsInput, ...func(*awssso.Options)) (*awssso.GetRoleCredentialsOutput, error) {
	return nil, errors.New("sts unavailable")
}

func TestConfigureProfiles_ErrorsLeaveFilesUnchanged(t *testing.T) {
	const startURL = "https://sso.example.invalid/start"
	const managedConfig = "[profile old-profile]\nawss_managed = true\nsso_start_url = " + startURL + "\n"
	const expiredCreds = "[expired]\nawss_managed = true\nsso_start_url = " + startURL + "\nexpiration = 2000-01-01T00:00:00Z\n"
	tests := []struct {
		name        string
		config      string
		credentials string
		client      SSOAPI
		wantErr     string
	}{
		{
			name:        "credentials clash",
			config:      managedConfig,
			credentials: expiredCreds + "[test-profile]\naws_access_key_id = FAKE_KEY_HAND\n",
			client:      filePathSSOClient{},
			wantErr:     "not managed by awss",
		},
		{
			name:        "config clash",
			config:      managedConfig + "[profile test-profile]\nregion = us-west-2\n",
			credentials: expiredCreds,
			client:      filePathSSOClient{},
			wantErr:     "not managed by awss",
		},
		{
			name:        "config managed for another start URL",
			config:      managedConfig + "[profile test-profile]\nawss_managed = true\nsso_start_url = https://other.example.invalid/start\n",
			credentials: expiredCreds,
			client:      filePathSSOClient{},
			wantErr:     "managed by awss for start URL https://other.example.invalid/start",
		},
		{
			name:        "credentials managed for another start URL",
			config:      managedConfig,
			credentials: expiredCreds + "[test-profile]\nawss_managed = true\nsso_start_url = https://other.example.invalid/start\n",
			client:      filePathSSOClient{},
			wantErr:     "managed by awss for start URL https://other.example.invalid/start",
		},
		{
			name:        "sts fetch error",
			config:      managedConfig,
			credentials: expiredCreds,
			client:      failingSTSClient{},
			wantErr:     "sts unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config")
			credentialsPath := filepath.Join(dir, "credentials")
			if err := os.WriteFile(configPath, []byte(tt.config), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(credentialsPath, []byte(tt.credentials), 0600); err != nil {
				t.Fatal(err)
			}
			loader := Loader{StartURL: startURL, Region: "us-east-1", SSOClient: tt.client}
			err := loader.configureProfiles(context.Background(), "FAKE_TOKEN_TEST", LoginOptions{
				ConfigPath: configPath, CredentialsPath: credentialsPath,
				NameTemplate: "test-profile", WithSTS: true,
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("configureProfiles error = %v, want %q", err, tt.wantErr)
			}
			for path, want := range map[string]string{configPath: tt.config, credentialsPath: tt.credentials} {
				got, err := os.ReadFile(filepath.Clean(path))
				if err != nil {
					t.Fatal(err)
				}
				if string(got) != want {
					t.Errorf("%s changed after failed login:\n%s", filepath.Base(path), got)
				}
			}
		})
	}
}
