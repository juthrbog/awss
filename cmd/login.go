package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/juthrbog/awss/internal/settings"
	"github.com/juthrbog/awss/internal/sso"
)

type loginFlags struct {
	org         string
	startURL    string
	ssoRegion   string
	accounts    []string
	roles       []string
	template    string
	sts         bool
	force       bool
	noBrowser   bool
	browserOnly bool
}

var loginOpts loginFlags

func init() {
	f := loginCmd.Flags()
	f.StringVarP(&loginOpts.org, "org", "o", "", "named org from the awss config file")
	f.StringVar(&loginOpts.startURL, "start-url", "", "IAM Identity Center start URL (overrides --org)")
	f.StringVar(&loginOpts.ssoRegion, "sso-region", "", "region of the Identity Center instance (default "+sso.DefaultRegion+")")
	f.StringSliceVarP(&loginOpts.accounts, "account", "a", nil, "only generate profiles for these account names (repeatable)")
	f.StringSliceVarP(&loginOpts.roles, "role", "r", nil, "only generate profiles for these role names (repeatable)")
	f.StringVar(&loginOpts.template, "template", "", "profile name template (default "+sso.DefaultNameTemplate+")")
	f.BoolVar(&loginOpts.sts, "sts", false, "also write STS credentials to the AWS credentials file")
	f.BoolVarP(&loginOpts.force, "force", "f", false, "log in again even if a valid SSO token is cached")
	f.BoolVar(&loginOpts.noBrowser, "no-browser", false, "print the login URL instead of opening a browser")
	f.BoolVar(&loginOpts.browserOnly, "browser", false, "only open the start URL in a browser, then exit")
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to IAM Identity Center and generate profiles",
	Long: `Log in to AWS IAM Identity Center (SSO) and write one profile per account
and role to your AWS config file.

Profiles written by awss carry an awss_managed=true key. Existing profiles
without that key are never modified. Managed profiles for the same start URL
that no longer match a role are removed.

The start URL comes from --start-url, or from a named org in the awss config
file (see --org). The config file lives at ~/.config/awss/config.yaml unless
AWSS_CONFIG or XDG_CONFIG_HOME is set:

  default_org: example
  orgs:
    example:
      start_url: https://example.awsapps.com/start
      sso_region: us-east-1
      template: '{{.Account.Name | trimPrefix "example-"}}-{{.Role}}'

Profile name templates use Go text/template syntax. Variables:

  {{.Account.ID}}      full AWS account number
  {{.Account.Name}}    account name as shown in the SSO portal
  {{.ShortAccountID}}  last four digits of the account number
  {{.Role}}            role (permission set) name

Functions:

  trimPrefix "p"       remove prefix p, e.g. {{.Account.Name | trimPrefix "example-"}}
  replace "a" "b"      replace all a with b, e.g. {{.Account.Name | replace "_" "-"}}`,
	Example: `  awss login
  awss login --org example
  awss login --start-url https://example.awsapps.com/start
  awss login --account prod --account dev --role AdministratorAccess
  awss login --template "{{.Account.Name}}-{{.Role}}"
  awss login --sts`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		files, err := resolveAWSFiles(cmd)
		if err != nil {
			return err
		}
		s, err := settings.Load(settings.Path())
		if err != nil {
			return err
		}
		org, err := s.Resolve(loginOpts.org, loginOpts.startURL)
		if err != nil {
			return err
		}

		region := firstNonEmpty(loginOpts.ssoRegion, org.SSORegion, sso.DefaultRegion)
		template := firstNonEmpty(loginOpts.template, org.Template, sso.DefaultNameTemplate)

		if loginOpts.browserOnly {
			return browser.OpenURL(org.StartURL)
		}

		loader := sso.NewLoader(org.StartURL, region)
		loader.DisableBrowserLaunch = loginOpts.noBrowser

		if err := loader.Login(cmd.Context(), sso.LoginOptions{
			ConfigPath:      files.configPath,
			CredentialsPath: files.credentialsPath,
			Accounts:        loginOpts.accounts,
			Roles:           loginOpts.roles,
			NameTemplate:    template,
			WithSTS:         loginOpts.sts,
			Force:           loginOpts.force,
		}); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Profiles updated. Run 'awss list' to see them.")
		return nil
	},
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
