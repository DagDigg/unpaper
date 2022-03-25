package server

import (
	"context"
	"fmt"

	"github.com/DagDigg/unpaper/core/cookies"
	"github.com/DagDigg/unpaper/core/session"
	"github.com/DagDigg/unpaper/extauth/pkg/response"
	authenvoy "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
)

// Default simply forwards the protected route headers to upstream
func (a *AuthorizationServer) Default(ctx context.Context, req *authenvoy.CheckRequest, headers map[string]string) (*authenvoy.CheckResponse, error) {
	return response.OK(response.GetHeaderValueOptions(headers)), nil
}

// StripeWebhook validates incoming stripe-signature. On success, a new header with this signature is passed to upstream
func (a *AuthorizationServer) StripeWebhook(req *authenvoy.CheckRequest) (*authenvoy.CheckResponse, error) {
	// Check if it's the stripe webook
	// If so, the signature should be propagated
	signature := req.Attributes.Request.Http.Headers["stripe-signature"]
	headers := map[string]string{"Grpc-Metadata-stripe-signature": signature}
	return response.OK(response.GetHeaderValueOptions(headers)), nil
}

// SignOut deletes the incoming user session from the in-memory db and cookies
func (a *AuthorizationServer) SignOut(ctx context.Context, req *authenvoy.CheckRequest, headers map[string]string) (*authenvoy.CheckResponse, error) {
	sid, ok := headers[session.HeaderName]
	if !ok {
		return response.KO("missing session"), nil
	}

	err := a.SM.Delete(ctx, sid)
	if err != nil {
		return response.KO(fmt.Sprintf("error deleting session from the in-memory db: %v", err)), nil
	}

	cookiesMngr := &cookies.Manager{
		Domain: a.Cfg.ClientDomain,
	}
	deletedCookieSID, err := cookiesMngr.DeleteCookie(session.CookieName)
	if err != nil {
		return response.KO(fmt.Sprintf("error deleting session cookie: %v", err)), nil
	}

	// Set header with deleted cookie
	headers["Set-Cookie"] = deletedCookieSID

	return response.OK(response.GetHeaderValueOptions(headers)), nil
}
