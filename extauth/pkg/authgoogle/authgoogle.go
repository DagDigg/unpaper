package authgoogle

import (
	"context"
	"errors"
	"strings"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

// GetGoogleOauth2Conf returns oauth2 Google openid connect config
func GetGoogleOauth2Conf(ctx context.Context, googleClientID, googleClientSecret string) (*oauth2.Config, error) {
	provider, err := getProvider(ctx)
	if err != nil {
		return nil, err
	}
	oauth2Config := &oauth2.Config{
		ClientID:     googleClientID,
		ClientSecret: googleClientSecret,
		RedirectURL:  "https://localhost:3000/login/google/callback",
		// Discovery returns the OAuth2 endpoints.
		Endpoint: provider.Endpoint(),
		// "openid" is a required scope for OpenID Connect flows.
		Scopes: []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return oauth2Config, nil
}

// GetProvider returns an OIDC Provider
func GetProvider(ctx context.Context) (*oidc.Provider, error) {
	return getProvider(ctx)
}

func getProvider(ctx context.Context) (*oidc.Provider, error) {
	return oidc.NewProvider(ctx, "https://accounts.google.com")
}

// ErrTokenExpired is used when validating an id token
var ErrTokenExpired = errors.New("token is expired")

// VerifyConfig configuration needed to verify id token
type VerifyConfig struct {
	ClientID        string
	RawIDToken      string
	SkipExpiryCheck bool
}

// VerifyRawIDToken performs id token validation. While validating, it can return an ErrTokenExpired
// on an expired token, otherwise it returns any encountered error.
// On validation succeeded, the id token payload is returned
func VerifyRawIDToken(ctx context.Context, cfg *VerifyConfig) (*oidc.IDToken, error) {
	provider, err := getProvider(ctx)
	if err != nil {
		return nil, err
	}

	verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID, SkipExpiryCheck: cfg.SkipExpiryCheck})
	idToken, err := verifier.Verify(ctx, cfg.RawIDToken)
	if err != nil {
		if strings.Contains(err.Error(), "token is expired") {
			return nil, ErrTokenExpired
		}
		return nil, err
	}
	return idToken, nil
}
