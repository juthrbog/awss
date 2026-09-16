package sso

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	ssooidctypes "github.com/aws/aws-sdk-go-v2/service/ssooidc/types"
	"github.com/pkg/browser"
)

// LoginOptions controls what Login writes.
type LoginOptions struct {
	// Accounts limits generated profiles to these account names. Empty means all.
	Accounts []string
	// Roles limits generated profiles to these role names. Empty means all.
	Roles []string
	// NameTemplate is the text/template used to build profile names.
	NameTemplate string
	// WithSTS also fetches STS credentials for each profile and writes them to
	// the AWS credentials file.
	WithSTS bool
	// Force runs the browser login even if a valid cached token exists.
	Force bool
}

// Login makes sure a valid SSO token exists, running the device authorization
// flow in a browser if needed, then writes one profile per discovered role.
func (loader Loader) Login(ctx context.Context, opts LoginOptions) error {
	var t token
	var err error
	if opts.Force {
		t, err = loader.browserLogin(ctx)
	} else {
		t, err = loader.tokenOrLogin(ctx)
	}
	if err != nil {
		return err
	}
	return loader.configureProfiles(ctx, t.AccessToken, opts)
}

// tokenOrLogin returns the cached token when it is still valid, otherwise it
// runs the browser login flow.
func (loader Loader) tokenOrLogin(ctx context.Context) (token, error) {
	t, err := loader.loadToken()
	if err == nil {
		return t, nil
	}
	var ite *InvalidTokenError
	if !errors.As(err, &ite) {
		return token{}, err
	}
	fmt.Fprintf(os.Stderr, "SSO token missing or expired, logging in: %v\n", err)
	return loader.browserLogin(ctx)
}

// configureProfiles writes discovered profiles into the AWS config file and
// either refreshes or clears STS credentials in the AWS credentials file.
//
// Every section this tool writes carries an awss_managed=true key so it never
// touches profiles the user maintains by hand. Writing over an unmanaged
// section is an error; the user has to remove or rename that section.
func (loader Loader) configureProfiles(ctx context.Context, accessToken string, opts LoginOptions) error {
	profiles, err := loader.generateProfiles(ctx, accessToken, opts)
	if err != nil {
		return err
	}
	if err := updateConfigFile(profiles); err != nil {
		return err
	}
	if opts.WithSTS {
		return loader.updateSTSCredentials(ctx, accessToken, profiles)
	}
	return clearSTSCredentials(loader.StartURL)
}

// browserLogin runs the OIDC device authorization flow and caches the
// resulting token where the AWS CLI and SDKs expect it.
func (loader Loader) browserLogin(ctx context.Context) (token, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return token{}, err
	}

	register, err := loader.OIDCClient.RegisterClient(ctx, &ssooidc.RegisterClientInput{
		ClientName: aws.String(hostname),
		ClientType: aws.String("public"),
		Scopes:     []string{"sso-portal:*"},
	})
	if err != nil {
		return token{}, fmt.Errorf("registering OIDC client: %w", err)
	}

	deviceAuth, err := loader.OIDCClient.StartDeviceAuthorization(ctx, &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     register.ClientId,
		ClientSecret: register.ClientSecret,
		StartUrl:     aws.String(loader.StartURL),
	})
	if err != nil {
		return token{}, fmt.Errorf("starting device authorization: %w", err)
	}
	expires := time.Now().Add(time.Duration(deviceAuth.ExpiresIn) * time.Second)

	loader.promptUser(aws.ToString(deviceAuth.VerificationUriComplete))

	out, err := loader.pollForToken(ctx, register.ClientId, register.ClientSecret, deviceAuth.DeviceCode, deviceAuth.Interval, expires)
	if err != nil {
		return token{}, err
	}

	t := token{
		AccessToken: aws.ToString(out.AccessToken),
		ExpiresAt:   rfc3339(time.Now().UTC().Add(time.Duration(out.ExpiresIn) * time.Second)),
		Region:      loader.Region,
		StartURL:    loader.StartURL,
	}
	if err := loader.writeToken(t); err != nil {
		return token{}, err
	}
	fmt.Fprintln(os.Stderr, "SSO login successful.")
	return t, nil
}

func (loader Loader) promptUser(url string) {
	if loader.DisableBrowserLaunch {
		fmt.Fprintf(os.Stderr, "Open this link to complete login:\n%s\n", url)
		return
	}
	if err := browser.OpenURL(url); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open a browser. Open this link to complete login:\n%s\n", url)
		return
	}
	fmt.Fprintf(os.Stderr, "If your browser did not open, visit:\n%s\n", url)
}

func (loader Loader) pollForToken(ctx context.Context, clientID, clientSecret, deviceCode *string, interval int32, expires time.Time) (*ssooidc.CreateTokenOutput, error) {
	if interval <= 0 {
		interval = 1
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
		if time.Now().After(expires) {
			return nil, errors.New("timed out waiting for SSO login")
		}
		out, err := loader.OIDCClient.CreateToken(ctx, &ssooidc.CreateTokenInput{
			ClientId:     clientID,
			ClientSecret: clientSecret,
			DeviceCode:   deviceCode,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
		})
		var pending *ssooidctypes.AuthorizationPendingException
		if errors.As(err, &pending) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("SSO authentication: %w", err)
		}
		return out, nil
	}
}
