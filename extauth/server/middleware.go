package server

import (
	"context"
	"fmt"

	"github.com/DagDigg/unpaper/core/cookies"
	"github.com/DagDigg/unpaper/core/session"
	"github.com/DagDigg/unpaper/extauth/pkg/response"
	authenvoy "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
)

// authorize middleware checks the validity of the incoming session.
// On success it extract the user id associated to it and returns the headers containing the session id and user id
func (a *AuthorizationServer) authorize(next EnrichedCheckRequest) IncomingCheckRequest {
	return func(ctx context.Context, req *authenvoy.CheckRequest) (*authenvoy.CheckResponse, error) {
		cookiesStr := req.Attributes.Request.Http.Headers["cookie"]
		sid, ok := cookies.FindAttribute(cookiesStr, session.CookieName)
		if !ok {
			return response.KO("missing session in cookie"), nil
		}
		refreshOccurred := false

		// Check if SID exists on rdb
		ok, err := a.SM.HasSession(ctx, sid)
		if err != nil {
			return response.KO("error retrieving session"), nil
		}
		if !ok {
			return response.KO("missing session in in-memory store"), nil
		}

		// Check if sid is about to exipre
		timeRemaining, err := a.RDB.TTL(ctx, sid).Result()
		if err != nil {
			return nil, err
		}
		if timeRemaining <= session.TTLRefresh {
			// Renew session
			newSID, err := a.SM.RenewSession(ctx, sid, session.Lifetime)
			if err != nil {
				return nil, fmt.Errorf("error renewing session: %v", err)
			}
			refreshOccurred = true
			sid = newSID
		}

		// Extract userID from session
		user, err := a.SM.GetUserBySID(ctx, sid)
		if err != nil {
			fmt.Printf("err extract, %v \n", err)
			return nil, fmt.Errorf("error extracting user from session: %v", err)
		}

		// Create headers to send to upstream
		headers := map[string]string{session.HeaderName: sid, "x-user-id": user.ID}

		if refreshOccurred {
			// In the scenario where session has been refreshed, we need to update the cookie
			cookiesMngr := &cookies.Manager{
				Domain: a.Cfg.ClientDomain,
			}
			cookieSIDStr, err := cookiesMngr.GetValue(session.CookieName, sid, session.Lifetime)
			if err != nil {
				return nil, err
			}
			headers["Set-Cookie"] = cookieSIDStr
		}

		return next(ctx, req, headers)
	}
}
