package v1_test

import (
	"context"
	"testing"
	"time"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/mdutils"
	v1Service "github.com/DagDigg/unpaper/backend/pkg/service/v1"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/DagDigg/unpaper/core/cookies"
	"github.com/DagDigg/unpaper/core/session"
	"github.com/Masterminds/squirrel"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestGoogleCallback(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)
	// Disable stripe feature so customers and subscriptions are not created
	ws.Cfg.EnableStripe = false

	t.Run("When signing up a google user", func(t *testing.T) {
		t.Parallel()
		// User data
		userID := uuid.NewString()
		givenName := uuid.NewString()
		email := uuid.NewString() + "@gmail.com"
		mdSent := metadata.MD{}
		mdCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", userID, "x-user-name", givenName, "x-user-email", email))
		ctx := grpc.NewContextWithServerTransportStream(mdCtx, &v1Testing.ServerTransportStreamMock{
			SendHeaderFunc: func(md metadata.MD) error {
				mdSent = md
				return nil
			},
		})

		req := &v1API.GoogleCallbackRequest{Api: "v1"}
		userRes, err := ws.Server.GoogleCallback(ctx, req)
		assert.Nilf(err, "%s", err)

		// Assert that a `Set-Cookie` is sent with a valid session
		checkSessionInMD(ctx, t, ws.Server.GetSM(), mdSent, userRes.Id)
		assert.Equal(userID, userRes.Id)
		assert.Equal(givenName, userRes.GivenName)
		assert.Equal(email, userRes.Email)
	})
}

func TestEmailSignup(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)
	// Disable stripe feature so customers and subscriptions are not created
	ws.Cfg.EnableStripe = false

	t.Run("When successfully signing up user", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		// Metadata sent back to downstream
		mdSent := metadata.MD{}
		transportStreamMock := v1Testing.ServerTransportStreamMock{
			SendHeaderFunc: func(md metadata.MD) error {
				mdSent = md
				return nil
			},
		}

		reqCtx := grpc.NewContextWithServerTransportStream(ctx, &transportStreamMock)
		req := &v1API.EmailSignupRequest{
			Api:      "v1",
			Username: uuid.NewString(),
			Email:    uuid.NewString() + "@gmail.com",
			Password: uuid.NewString(),
		}
		userRes, err := ws.Server.EmailSignup(reqCtx, req)
		assert.Nil(err)

		// Assert user response
		assert.Equal(userRes.Username, req.Username)
		assert.Equal(userRes.Email, req.Email)
		assert.NotEqual(userRes.Id, "")

		// Assert that a `Set-Cookie` is sent with a valid session
		checkSessionInMD(reqCtx, t, ws.Server.GetSM(), mdSent, userRes.Id)
	})

	t.Run("When invalid argument is provided", func(t *testing.T) {
		tests := []struct {
			req        *v1API.EmailSignupRequest
			wantErrStr string
		}{
			{
				req:        &v1API.EmailSignupRequest{Api: "v1", Email: "valid1@gmail.com", Password: "V4LidPasSw!!"},
				wantErrStr: "invalid username",
			},
			{
				req:        &v1API.EmailSignupRequest{Api: "v1", Username: "validUser1", Password: "V4LidPasSw!!"},
				wantErrStr: "invalid email",
			},
			{
				req:        &v1API.EmailSignupRequest{Api: "v1", Username: "validUser2", Password: "short"},
				wantErrStr: "invalid password",
			},
			{
				req:        &v1API.EmailSignupRequest{Api: "v1", Username: "validUser3", Password: "V4LidPasSw!!", Email: "invalid"},
				wantErrStr: "invalid email",
			},
		}
		for _, tt := range tests {
			// Mock context with fake sendheader
			ctx := grpc.NewContextWithServerTransportStream(context.Background(), &v1Testing.ServerTransportStreamMock{
				SendHeaderFunc: func(md metadata.MD) error { return nil },
			})
			_, err := ws.Server.EmailSignup(ctx, tt.req)
			assert.NotNilf(err, "request:\n%v", tt.req)
			assert.Contains(err.Error(), tt.wantErrStr)
		}
	})

	t.Run("Other errors", func(t *testing.T) {
		tests := []struct {
			req        *v1API.EmailSignupRequest
			wantErrStr string
		}{
			{
				req:        &v1API.EmailSignupRequest{Api: "v1", Email: "valid1@gmail.com", Password: "V4LidPasSw!!", Username: "validUsr"},
				wantErrStr: "",
			},
			// This insertion must fail because the email already exists
			{
				req:        &v1API.EmailSignupRequest{Api: "v1", Email: "valid1@gmail.com", Password: "V4LidPasSw!!", Username: "validUsr2"},
				wantErrStr: "email already exists",
			},
			// This insertion must fail because the username already exists
			{
				req:        &v1API.EmailSignupRequest{Api: "v1", Email: "super_valid@gmail.com", Password: "V4LidPasSw!!", Username: "validUsr"},
				wantErrStr: "username already exists",
			},
		}

		for _, tt := range tests {
			ctx := context.Background()
			// Mock context with fake sendheader
			reqCtx := grpc.NewContextWithServerTransportStream(ctx, &v1Testing.ServerTransportStreamMock{
				SendHeaderFunc: func(md metadata.MD) error { return nil },
			})
			_, err := ws.Server.EmailSignup(reqCtx, tt.req)
			if len(tt.wantErrStr) == 0 {
				// Error is not expected
				assert.Nilf(err, "req: %v\nerr: %s", tt.req, err)
				continue
			}

			// Error is expected
			assert.NotNilf(err, "request:\n%v", tt.req)
			assert.Contains(err.Error(), tt.wantErrStr)
		}
	})
}

func TestEmailSignIn(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When adding and getting an user", func(t *testing.T) {
		t.Parallel()
		// Insert fresh user
		userParams := v1Testing.GetRandomPGUserParams()
		user, err := ws.AddUser(userParams)
		assert.Nil(err)

		mdSent := metadata.MD{}
		// Create metadata with userID
		ctx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs("x-user-id", user.Id),
		)
		ctx = grpc.NewContextWithServerTransportStream(ctx, &v1Testing.ServerTransportStreamMock{
			SendHeaderFunc: func(md metadata.MD) error {
				mdSent = md
				return nil
			},
		})

		req := &v1API.EmailSigninRequest{Api: "v1", Email: userParams.Email, Password: userParams.Password.String}
		userRes, err := ws.Server.EmailSignin(ctx, req)
		assert.Nilf(err, "%s", err)

		checkSessionInMD(ctx, t, ws.Server.GetSM(), mdSent, userRes.Id)
		v1Testing.AssertUser(t, userRes, userParams)
	})
}

func TestEmailCheck(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When email is valid", func(t *testing.T) {
		t.Parallel()
		req := &v1API.EmailCheckRequest{Api: "v1", Email: "valid@gmail.com"}
		_, err := ws.Server.EmailCheck(context.Background(), req)
		assert.Nil(err)
	})
	t.Run("When email is invalid", func(t *testing.T) {
		t.Parallel()
		req := &v1API.EmailCheckRequest{Api: "v1", Email: "invalid@foooobarbaz.com"}
		_, err := ws.Server.EmailCheck(context.Background(), req)
		assert.NotNil(err)
	})
}

func TestEmailVerify(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When adding user and verifying email", func(t *testing.T) {
		t.Parallel()
		// Insert fresh user
		userParams := v1Testing.GetRandomPGUserParams()
		user, err := ws.AddUser(userParams)
		if err != nil {
			t.Errorf("error inserting user: '%v'", err)
		}
		if user.EmailVerified == true {
			t.Error("newly created user should not have email_verified set to true")
		}
		// Create verification token
		unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
			Subject:   user.Id,
			Issuer:    "unpaper",
			ExpiresAt: time.Now().Add(1 * time.Minute).Unix(),
		})
		verificationToken, err := unsignedToken.SignedString([]byte(ws.Cfg.UnpaperClientSecret))
		if err != nil {
			t.Errorf("error signing token: ''%v", err)
		}
		// Append verification token to the request
		req := &v1API.EmailVerifyRequest{VerificationToken: verificationToken}

		// Add metadata to context
		ctx := metadata.NewIncomingContext(
			ws.Ctx,
			metadata.Pairs("x-user-id", user.Id),
		)

		// Call the RPC
		_, err = ws.Server.EmailVerify(ctx, req)
		if err != nil {
			t.Errorf("error calling rpc EmailVerify: '%v'", err)
		}

		// Check that the user's email_verified column has been set to true
		var result bool
		q := ws.Server.GetSB().Select("email_verified").From("users").Where(squirrel.Eq{"id": user.Id})
		err = q.QueryRowContext(ctx).Scan(&result)
		assert.Nil(err)
		assert.True(result)
	})
}

func TestResetPassword(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When resetting an user password", func(t *testing.T) {
		t.Parallel()
		userParams := v1Testing.GetRandomPGUserParams()
		user, err := ws.AddUser(userParams)
		assert.Nil(err)

		unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
			Subject:   user.Id,
			Issuer:    "unpaper",
			ExpiresAt: time.Now().Add(60 * time.Minute * 14).Unix(), // 14 days lifespan
		})
		verificationToken, err := unsignedToken.SignedString([]byte(ws.Cfg.UnpaperClientSecret))
		assert.Nil(err)

		req := &v1API.ResetPasswordRequest{
			VerificationToken: verificationToken,
			NewPassword:       "obla-di-obla-da",
			Repeat:            "obla-di-obla-da",
		}

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", userParams.ID))
		_, err = ws.Server.ResetPassword(ctx, req)
		assert.Nil(err)
	})

	t.Run("When verification token is expired", func(t *testing.T) {
		t.Parallel()
		userParams := v1Testing.GetRandomPGUserParams()
		user, err := ws.AddUser(userParams)
		assert.Nil(err)

		unsignedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
			Subject:   user.Id,
			Issuer:    "unpaper",
			ExpiresAt: time.Now().AddDate(0, 0, -1).Unix(), // 1 day ago
		})
		verificationToken, err := unsignedToken.SignedString([]byte(ws.Cfg.UnpaperClientSecret))
		assert.Nil(err)

		req := &v1API.ResetPasswordRequest{
			VerificationToken: verificationToken,
			NewPassword:       "obla-di-obla-da",
			Repeat:            "obla-di-obla-da",
		}

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", user.Id))
		_, err = ws.Server.ResetPassword(ctx, req)
		assert.NotNil(err)

		assert.Equal(err, v1Service.TokenError)
	})

	t.Run("When passwords mismatch", func(t *testing.T) {
		t.Parallel()
		userParams := v1Testing.GetRandomPGUserParams()
		user, err := ws.AddUser(userParams)
		assert.Nil(err)

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", user.Id))
		req := &v1API.ResetPasswordRequest{
			VerificationToken: "verificationToken",
			NewPassword:       "obla-di-obla-da",
			Repeat:            "DIFFERENT",
		}

		_, err = ws.Server.ResetPassword(ctx, req)
		assert.NotNil(err)
	})

	t.Run("When password isnt valid", func(t *testing.T) {
		t.Parallel()
		userParams := v1Testing.GetRandomPGUserParams()
		user, err := ws.AddUser(userParams)
		assert.Nil(err)

		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", user.Id))
		req := &v1API.ResetPasswordRequest{
			VerificationToken: "verificationToken",
			NewPassword:       "small",
			Repeat:            "small",
		}

		_, err = ws.Server.ResetPassword(ctx, req)
		assert.NotNil(err)
	})
}

// checkSessionInMD returns whether metadata contains a `Set-Cookie` value
// with a valid session corresponding to the user id
func checkSessionInMD(ctx context.Context, t *testing.T, sm *session.Manager, md metadata.MD, userID string) {
	// Retrieve metadata `Set-Cookie`. A session must have been submitted
	cookiesSent, ok := mdutils.GetFirstMDValue(md, "Set-Cookie")
	assert.True(t, ok)
	sid, ok := cookies.FindAttribute(cookiesSent, session.CookieName)
	assert.True(t, ok)

	// A session must have been created on the in-memory-db
	sessionUser, err := sm.GetUserBySID(ctx, sid)
	assert.Nil(t, err, "%s", err)

	assert.Equal(t, sessionUser.ID, userID)
}
