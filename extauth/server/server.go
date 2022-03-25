package server

import (
	"context"
	"fmt"
	"net/url"

	"github.com/DagDigg/unpaper/core/config"
	"github.com/DagDigg/unpaper/core/session"
	"github.com/DagDigg/unpaper/extauth/pkg/response"
	authenvoy "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/go-redis/redis/v8"

	// header_to_metadata driver
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/header_to_metadata/v3"

	// pgx driver
	_ "github.com/jackc/pgx/v4/stdlib"
)

// AuthorizationServer has methods to handle incoming request validation
type AuthorizationServer struct {
	// Redis instance
	RDB *redis.Client

	// Session Manager
	SM session.Session

	// Configuration variables
	Cfg *config.Config
}

// IncomingCheckRequest is the default envoy `Check` signature function
type IncomingCheckRequest func(ctx context.Context, req *authenvoy.CheckRequest) (*authenvoy.CheckResponse, error)

// EnrichedCheckRequest is the default envoy `Check` signature function plus a header map
type EnrichedCheckRequest func(ctx context.Context, req *authenvoy.CheckRequest, headers map[string]string) (*authenvoy.CheckResponse, error)

// NewAuthorizationServer opens a pgx DB with the provided URL,
// and returns an *Authorization server with the DB and *config.Config associated
func NewAuthorizationServer(rdbURL *url.URL, cfg *config.Config) (*AuthorizationServer, error) {
	// Create redis client
	opt, err := redis.ParseURL(rdbURL.String())
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)

	// Create session manager
	sm := session.NewManager(rdb)

	return &AuthorizationServer{RDB: rdb, SM: sm, Cfg: cfg}, nil
}

// TODO: not sure if it's safe to keep reflection
var unauthPaths = []string{
	"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
	"/v1.UnpaperService/EmailCheck", // TODO: not sure if its safe
	"/v1.UnpaperService/SendResetLink",
	"/v1.UnpaperService/ResetPassword",
	"/v1.UnpaperService/EmailSignup",
	"/v1.UnpaperService/EmailSignin",
	"/v1.UnpaperService/Ping",
	"/h1/v1/Ping",
}

// Check authorization
func (a *AuthorizationServer) Check(ctx context.Context, req *authenvoy.CheckRequest) (*authenvoy.CheckResponse, error) {
	// Get request target path. For GRPC calls this is usually in the form of: /api.Service/Method
	path := req.Attributes.Request.Http.Path

	fmt.Println("incoming request on path: ", path)

	// Unauthenticated routes
	if contains(unauthPaths, path) {
		return response.OK(nil), nil
	}

	switch path {
	case "/v1.UnpaperService/GoogleLogin":
		return a.GoogleLogin(ctx)
	case "/v1.UnpaperService/GoogleCallback":
		return a.GoogleCallback(ctx, req)
	case "/v1.UnpaperService/GoogleOneTap":
		return a.GoogleOneTap(ctx, req)
	case "/v1.UnpaperService/SignOut":
		return a.authorize(a.SignOut)(ctx, req)
	case "/h1/v1/webhook:Stripe", "/h1/v1/webhook:Stripe:Connect":
		return a.StripeWebhook(req)
	default:
		return a.authorize(a.Default)(ctx, req)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
