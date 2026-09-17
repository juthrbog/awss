package sso

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssso "github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"gopkg.in/ini.v1"
)

type filePathSSOClient struct{}

func (filePathSSOClient) ListAccounts(context.Context, *awssso.ListAccountsInput, ...func(*awssso.Options)) (*awssso.ListAccountsOutput, error) {
	return &awssso.ListAccountsOutput{AccountList: []types.AccountInfo{{AccountId: aws.String("111122223333"), AccountName: aws.String("test")}}}, nil
}

func (filePathSSOClient) ListAccountRoles(context.Context, *awssso.ListAccountRolesInput, ...func(*awssso.Options)) (*awssso.ListAccountRolesOutput, error) {
	return &awssso.ListAccountRolesOutput{RoleList: []types.RoleInfo{{AccountId: aws.String("111122223333"), RoleName: aws.String("ReadOnly")}}}, nil
}

func (filePathSSOClient) GetRoleCredentials(context.Context, *awssso.GetRoleCredentialsInput, ...func(*awssso.Options)) (*awssso.GetRoleCredentialsOutput, error) {
	return &awssso.GetRoleCredentialsOutput{RoleCredentials: &types.RoleCredentials{
		AccessKeyId: aws.String("FAKE_KEY_TEST"), SecretAccessKey: aws.String("FAKE_SECRET_TEST"),
		SessionToken: aws.String("FAKE_TOKEN_TEST"), Expiration: time.Now().Add(time.Hour).UnixMilli(),
	}}, nil
}

func TestLoginOutputFileOverrides(t *testing.T) {
	for _, withSTS := range []bool{false, true} {
		name := "clear-expired"
		if withSTS {
			name = "write-sts"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			defaultConfig := filepath.Join(dir, "default-config")
			defaultCredentials := filepath.Join(dir, "default-credentials")
			const untouched = "# Must not be read or written when overrides are provided.\n"
			for _, path := range []string{defaultConfig, defaultCredentials} {
				if err := os.WriteFile(path, []byte(untouched), 0600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("AWS_CONFIG_FILE", defaultConfig)
			t.Setenv("AWS_SHARED_CREDENTIALS_FILE", defaultCredentials)
			configPath := filepath.Join(dir, "custom-config")
			credentialsPath := filepath.Join(dir, "custom-credentials")
			const startURL = "https://sso.example.invalid/start"
			stale := "[expired]\nawss_managed=true\nsso_start_url=" + startURL + "\nexpiration=2000-01-01T00:00:00Z\n"
			if err := os.WriteFile(credentialsPath, []byte(stale), 0600); err != nil {
				t.Fatal(err)
			}
			loader := Loader{StartURL: startURL, Region: "us-east-1", SSOClient: filePathSSOClient{}}
			err := loader.configureProfiles(context.Background(), "FAKE_TOKEN_TEST", LoginOptions{
				ConfigPath: configPath, CredentialsPath: credentialsPath,
				NameTemplate: "test-profile", WithSTS: withSTS,
			})
			if err != nil {
				t.Fatal(err)
			}
			configFile, err := ini.Load(configPath)
			if err != nil || !configFile.HasSection("profile test-profile") {
				t.Fatalf("custom config missing generated profile: %v", err)
			}
			credentialsFile, err := ini.Load(credentialsPath)
			if err != nil {
				t.Fatal(err)
			}
			if credentialsFile.HasSection("expired") {
				t.Fatal("expired credentials were not removed from the custom file")
			}
			if credentialsFile.HasSection("test-profile") != withSTS {
				t.Fatalf("custom credentials presence does not match WithSTS=%v", withSTS)
			}
			for _, path := range []string{defaultConfig, defaultCredentials} {
				data, err := os.ReadFile(filepath.Clean(path))
				if err != nil || string(data) != untouched {
					t.Errorf("default file was changed: %s, %v", path, err)
				}
			}
		})
	}
}
