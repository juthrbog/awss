// Package sso discovers AWS IAM Identity Center (SSO) accounts and roles for a
// start URL and writes matching profiles into the AWS config file. It can also
// exchange the SSO token for short-lived STS credentials and store them in the
// AWS credentials file for tools that do not understand SSO profiles.
package sso

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	ssotypes "github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// DefaultRegion is the SSO region used when none is configured. It is the
// region of the Identity Center instance, not the region for API calls made
// with the resulting profiles.
const DefaultRegion = "us-east-1"

// DefaultNameTemplate is the profile name template used when none is configured.
const DefaultNameTemplate = "{{.Account.Name}}-{{.Role}}"

// Loader talks to Identity Center for one start URL.
type Loader struct {
	StartURL             string
	Region               string
	DisableBrowserLaunch bool

	SSOClient  SSOAPI
	OIDCClient OIDCAPI

	// tokenCacheDir overrides the SSO token cache location. Empty means
	// ~/.aws/sso/cache, which is where the AWS CLI and SDKs look.
	tokenCacheDir string
}

// SSOAPI is the subset of the sso client used by Loader.
type SSOAPI interface {
	sso.ListAccountsAPIClient
	sso.ListAccountRolesAPIClient
	GetRoleCredentials(ctx context.Context, params *sso.GetRoleCredentialsInput, optFns ...func(*sso.Options)) (*sso.GetRoleCredentialsOutput, error)
}

// OIDCAPI is the subset of the ssooidc client used by Loader.
type OIDCAPI interface {
	RegisterClient(ctx context.Context, params *ssooidc.RegisterClientInput, optFns ...func(*ssooidc.Options)) (*ssooidc.RegisterClientOutput, error)
	StartDeviceAuthorization(ctx context.Context, params *ssooidc.StartDeviceAuthorizationInput, optFns ...func(*ssooidc.Options)) (*ssooidc.StartDeviceAuthorizationOutput, error)
	CreateToken(ctx context.Context, params *ssooidc.CreateTokenInput, optFns ...func(*ssooidc.Options)) (*ssooidc.CreateTokenOutput, error)
}

// NewLoader returns a Loader for the given start URL and SSO region.
// An empty region falls back to DefaultRegion.
func NewLoader(startURL, region string) Loader {
	if region == "" {
		region = DefaultRegion
	}
	cfg := aws.NewConfig()
	cfg.Region = region
	return Loader{
		StartURL:   startURL,
		Region:     region,
		SSOClient:  sso.NewFromConfig(*cfg),
		OIDCClient: ssooidc.NewFromConfig(*cfg),
	}
}

// Account is an AWS account visible through Identity Center.
type Account struct {
	// ID is the 12-digit AWS account number.
	ID string
	// Name is the account name as shown in the SSO portal.
	Name string
}

// Role is a permission set assignment for one account. Its fields are the
// variables available to profile name templates.
type Role struct {
	// SSORegion is the Identity Center region.
	SSORegion string
	// SSOStartURL is the Identity Center start URL.
	SSOStartURL string
	// Account is the AWS account the role belongs to.
	Account Account
	// ShortAccountID is the last four digits of the account number.
	ShortAccountID string
	// Role is the permission set (role) name.
	Role string
}

var templateFuncMap = template.FuncMap{
	"trimPrefix": func(prefix, s string) string { return strings.TrimPrefix(s, prefix) },
	"replace":    func(old, new, s string) string { return strings.ReplaceAll(s, old, new) },
}

// profileName renders the profile name template for this role.
func (r Role) profileName(t string) (string, error) {
	if t == "" {
		t = DefaultNameTemplate
	}
	tmpl, err := template.New("").Funcs(templateFuncMap).Parse(t)
	if err != nil {
		return "", fmt.Errorf("invalid profile name template %q: %w", t, err)
	}
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, r); err != nil {
		return "", fmt.Errorf("rendering profile name template %q: %w", t, err)
	}
	return buf.String(), nil
}

// Roles lists every role the token grants, limited to the named accounts when
// accountFilter is non-empty.
func (loader Loader) Roles(ctx context.Context, accessToken string, accountFilter []string) ([]Role, error) {
	accounts, err := loader.accounts(ctx, accessToken, accountFilter)
	if err != nil {
		return nil, fmt.Errorf("listing SSO accounts: %w", err)
	}
	roleInfos, err := loader.rolesForAccounts(ctx, accessToken, accounts)
	if err != nil {
		return nil, fmt.Errorf("listing SSO roles: %w", err)
	}

	byID := make(map[string]Account, len(accounts))
	for _, a := range accounts {
		acct := Account{ID: aws.ToString(a.AccountId), Name: aws.ToString(a.AccountName)}
		byID[acct.ID] = acct
	}

	roles := make([]Role, 0, len(roleInfos))
	for _, ri := range roleInfos {
		id := aws.ToString(ri.AccountId)
		roles = append(roles, Role{
			SSOStartURL:    loader.StartURL,
			SSORegion:      loader.Region,
			Account:        byID[id],
			ShortAccountID: shortAccount(id),
			Role:           aws.ToString(ri.RoleName),
		})
	}
	return roles, nil
}

func (loader Loader) accounts(ctx context.Context, accessToken string, accountFilter []string) ([]ssotypes.AccountInfo, error) {
	var accounts []ssotypes.AccountInfo
	p := sso.NewListAccountsPaginator(loader.SSOClient, &sso.ListAccountsInput{
		AccessToken: aws.String(accessToken),
	})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, a := range page.AccountList {
			if inFilter(aws.ToString(a.AccountName), accountFilter) {
				accounts = append(accounts, a)
			}
		}
	}
	return accounts, nil
}

// rolesForAccounts fans out ListAccountRoles across accounts. Calls are rate
// limited because Identity Center throttles aggressively with many accounts.
func (loader Loader) rolesForAccounts(ctx context.Context, accessToken string, accounts []ssotypes.AccountInfo) ([]ssotypes.RoleInfo, error) {
	g, ctx := errgroup.WithContext(ctx)
	limiter := rate.NewLimiter(rate.Every(time.Second/4), 2)

	var mu sync.Mutex
	var roles []ssotypes.RoleInfo

	for _, account := range accounts {
		g.Go(func() error {
			if err := limiter.Wait(ctx); err != nil {
				return err
			}
			r, err := loader.rolesForAccount(ctx, accessToken, account)
			if err != nil {
				return fmt.Errorf("account %s: %w", aws.ToString(account.AccountName), err)
			}
			mu.Lock()
			roles = append(roles, r...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return roles, nil
}

func (loader Loader) rolesForAccount(ctx context.Context, accessToken string, account ssotypes.AccountInfo) ([]ssotypes.RoleInfo, error) {
	var roles []ssotypes.RoleInfo
	p := sso.NewListAccountRolesPaginator(loader.SSOClient, &sso.ListAccountRolesInput{
		AccessToken: aws.String(accessToken),
		AccountId:   account.AccountId,
	})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		roles = append(roles, page.RoleList...)
	}
	return roles, nil
}

// inFilter reports whether elem is in filter. An empty filter matches everything.
func inFilter(elem string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, e := range filter {
		if elem == e {
			return true
		}
	}
	return false
}

// shortAccount returns the last four characters of an account number, or the
// whole string when it is shorter than that.
func shortAccount(account string) string {
	if len(account) <= 4 {
		return account
	}
	return account[len(account)-4:]
}
