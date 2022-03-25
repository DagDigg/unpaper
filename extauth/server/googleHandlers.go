package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/DagDigg/unpaper/core/cookies"
	"github.com/DagDigg/unpaper/extauth/pkg/authgoogle"
	"github.com/DagDigg/unpaper/extauth/pkg/codeverifier"
	"github.com/DagDigg/unpaper/extauth/pkg/response"

	authenvoy "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"golang.org/x/oauth2"
)

// GoogleClaims represent google id token claims
type GoogleClaims struct {
	Subject    string `json:"sub"`
	Email      string `json:"email"`
	Verified   bool   `json:"email_verified"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

// GoogleLogin returns a CheckResponse with custom added consent screen url and state headers to be consumed in upstream
func (a *AuthorizationServer) GoogleLogin(ctx context.Context) (*authenvoy.CheckResponse, error) {
	oauth2Conf, err := authgoogle.GetGoogleOauth2Conf(ctx, a.Cfg.GoogleClientID, a.Cfg.GoogleClientSecret)
	if err != nil {
		return response.KO("Failed to get oauth2conf"), nil
	}

	// Create random state
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	// Create code verifier and code challenge
	verifier, err := codeverifier.CreateCodeVerifier()
	if err != nil {
		return response.KO("failed to create code verifier"), nil
	}
	challenge := verifier.CodeChallengeS256()

	// Create challenge and challenge method url param
	challengeParam := oauth2.SetAuthURLParam("code_challenge", challenge)
	challengeMethodParam := oauth2.SetAuthURLParam("code_challenge_method", "S256")

	// Get consent screen url with state.
	// AccessTypeOffline so the refresh token can be retrieved
	u := oauth2Conf.AuthCodeURL(state, oauth2.AccessTypeOffline, challengeParam, challengeMethodParam)

	headers := response.GetHeaderValueOptions(map[string]string{"ConsentURL": u, "State": state, "Code-Verifier": verifier.String()})

	return response.OK(headers), nil
}

// GoogleCallback handles the auth code exchange, gets a token, and use that newly generated token to get userinfo to be stored on DB.
// It also create a session, that will be set as httpOnly cookie and will be used for future validations
func (a *AuthorizationServer) GoogleCallback(ctx context.Context, req *authenvoy.CheckRequest) (*authenvoy.CheckResponse, error) {
	// Get oauth2 config
	oauth2Conf, err := authgoogle.GetGoogleOauth2Conf(ctx, a.Cfg.GoogleClientID, a.Cfg.GoogleClientSecret)
	if err != nil {
		return response.KO("Failed to get oauth2conf"), nil
	}
	// Exchange code for token
	oauth2Token, err := exchangeCodeForToken(ctx, oauth2Conf, req)
	if err != nil {
		return response.KO(err.Error()), nil
	}

	// Verify and parse token into claims
	gClaims, err := parseGoogleToken(oauth2Token, a.Cfg.GoogleClientID)

	// Pass to upstream data needed for creating an user
	headers := response.GetHeaderValueOptions(map[string]string{
		"x-user-id":    gClaims.Subject,
		"x-user-name":  gClaims.GivenName,
		"x-user-email": gClaims.Email,
	})

	return response.OK(headers), nil
}

// GoogleOneTap handles google one-tap login
func (a *AuthorizationServer) GoogleOneTap(ctx context.Context, req *authenvoy.CheckRequest) (*authenvoy.CheckResponse, error) {
	clientID, ok := req.Attributes.Request.Http.Headers["x-google-client-id"]
	if !ok {
		fmt.Println("missing client id")
		return response.KO("missing client id"), nil
	}
	rawIDToken, ok := req.Attributes.Request.Http.Headers["x-google-id-token"]
	if !ok {
		fmt.Println("missing id token")
		return response.KO("missing id token"), nil
	}

	// Parse and verify ID Token payload.
	idToken, err := authgoogle.VerifyRawIDToken(oauth2.NoContext, &authgoogle.VerifyConfig{ClientID: clientID, RawIDToken: rawIDToken})
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf(fmt.Sprintf("Failed to verify RawIDToken: %v", err))
	}

	// Extract custom claims
	gClaims := &GoogleClaims{}
	if err := idToken.Claims(&gClaims); err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("Failed to unmarshal claims")
	}

	// Pass to upstream data needed for creating an user
	headers := response.GetHeaderValueOptions(map[string]string{
		"x-user-id":    gClaims.Subject,
		"x-user-name":  gClaims.GivenName,
		"x-user-email": gClaims.Email,
	})

	return response.OK(headers), nil
}

func exchangeCodeForToken(ctx context.Context, oauth2Conf *oauth2.Config, req *authenvoy.CheckRequest) (*oauth2.Token, error) {
	code := req.Attributes.Request.Http.Headers["x-code"]
	cookiesStr := req.Attributes.Request.Http.Headers["cookie"]
	codeVerifier, _ := cookies.FindAttribute(cookiesStr, "x-code-verifier")

	if code == "" {
		return nil, fmt.Errorf("Failed to get code from request")
	}
	if codeVerifier == "" {
		return nil, fmt.Errorf("Failed to get codeVerifier from request")
	}

	// Create code verifier param
	codeVerifierParam := oauth2.SetAuthURLParam("code_verifier", codeVerifier)

	// Exchange code for token
	oauth2Token, err := oauth2Conf.Exchange(ctx, code, codeVerifierParam)
	if err != nil {
		return nil, fmt.Errorf("Failed to exchange code for token")
	}

	return oauth2Token, nil
}

// Parses google token into custom Google Claims
func parseGoogleToken(tok *oauth2.Token, googleClientID string) (*GoogleClaims, error) {
	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := tok.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("Failed to get id_token")
	}

	// Parse and verify ID Token payload.
	idToken, err := authgoogle.VerifyRawIDToken(oauth2.NoContext, &authgoogle.VerifyConfig{ClientID: googleClientID, RawIDToken: rawIDToken})
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("Failed to verify RawIDToken: %v", err))
	}

	// Extract custom claims
	gClaims := &GoogleClaims{}
	if err := idToken.Claims(&gClaims); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal claims")
	}

	return gClaims, nil
}
