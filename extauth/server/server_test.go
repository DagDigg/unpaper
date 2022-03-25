package server_test

import (
	"testing"
)

func TestEmailSignup(t *testing.T) {
	// t.Parallel()
	// cfg := config.Get()
	// rdbURL := cfg.GetRDBConnURL()

	// authServer, err := server.NewAuthorizationServer(unpapertest.StartRedisDB(t, rdbURL), cfg)
	// if err != nil {
	// 	t.Fatalf("error creating authserver: %v", err)
	// }

	// reqdata := struct {
	// 	Email    string
	// 	Password string
	// }{
	// 	Email:    "mail@google.com",
	// 	Password: "very-long-password-so-bcrypt-doesnt-complain",
	// }
	// ctx := context.Background()
	// req := unpapertest.MockRequest(
	// 	"/v1.UnpaperService/EmailSignup",
	// 	map[string]string{"x-email": reqdata.Email, "x-passw": reqdata.Password},
	// )

	// checkResp, err := authServer.Check(ctx, req)
	// if err != nil {
	// 	t.Errorf("error calling ext auth check rpc: %q", err)
	// }
	// stmt := `
	// 	SELECT id, email, password, family_name
	// 	FROM users
	// 	WHERE email = $1
	// `
	// type user struct {
	// 	ID         string `json:"id"`
	// 	Email      string `json:"email"`
	// 	Password   string `json:"password"`
	// 	FamilyName string `json:"family_name"`
	// }
	// var expected user
	// _ = authServer.DB.QueryRowContext(ctx, stmt, reqdata.Email).Scan(&expected.ID, &expected.Email, &expected.Password, &expected.FamilyName)

	// if expected == (user{}) {
	// 	t.Error("failed to insert user")
	// }
	// if expected.ID == "" {
	// 	t.Error("failed to insert user ID")
	// }
	// if expected.Email != reqdata.Email {
	// 	t.Errorf("stored user email and request email does not match. DB email: '%v', req email: '%v'", expected.Email, reqdata.Email)
	// }
	// if expected.Password == "" {
	// 	t.Error("failed to create password")
	// }
	// // Decrypt password
	// err = bcrypt.CompareHashAndPassword([]byte(expected.Password), []byte(reqdata.Password))
	// if err != nil {
	// 	t.Errorf("stored password and request password does not match: '%v'", err)
	// }

	// // CheckResponse assertions
	// if checkResp.Status.Code != int32(codes.OK) {
	// 	t.Errorf("failed response status check. Got: '%v', want: '%v'", checkResp.Status.Code, int32(codes.OK))
	// }

	// resp := checkResp.HttpResponse.(*authenvoy.CheckResponse_OkResponse)
	// headers := resp.OkResponse.Headers
	// for _, v := range headers {
	// 	switch v.Header.Key {
	// 	case "Set-Cookie":
	// 		validateSessionCookie(t, v.Header.Value)
	// 	case "x-email":
	// 		if v.Header.Value != expected.Email {
	// 			t.Errorf("header mismatch for 'x-email'. got: '%v', want: '%v'", v.Header.Value, expected.Email)
	// 		}
	// 	case "x-user-id":
	// 		if v.Header.Value != expected.ID {
	// 			t.Errorf("header mismatch for 'x-user-id'. got: '%v', want: '%v'", v.Header.Value, expected.ID)
	// 		}
	// 	case "x-verification-token":
	// 		validateVerificationToken(t, cfg.UnpaperClientSecret, v.Header.Value)
	// 	default:
	// 		t.Errorf("unexpected header: %v", v)
	// 	}
	// }
}

func validateSessionCookie(t *testing.T, cookie string) {
	// 	// At the moment there's no validation on JWT
	// 	_, ok := cookies.FindAttribute(cookie, session.CookieName)
	// 	if !ok {
	// 		t.Errorf("could not find cookie attribute %q", session.CookieName)
	// 	}
	// 	domain, ok := cookies.FindAttribute(cookie, "Domain")
	// 	assert.True(t, ok)

	// 	if domain != "localhost" {
	// 		t.Errorf("unexpected Path. got: '%v', want: '%v'", domain, "localhost")
	// 	}

	// 	path, ok := cookies.FindAttribute(cookie, "Path")
	// 	assert.True(t, ok)
	// 	assert.Equal(t, path, "/")

	// 	maxAge, ok := cookies.FindAttribute(cookie, "Max-Age")
	// 	assert.True(t, ok)
	// 	sessionLifetimeStr := strconv.Itoa(int(session.Lifetime.Seconds()))
	// 	assert.Equal(t, maxAge, sessionLifetimeStr)
	// }

	// func validateVerificationToken(t *testing.T, secret, tokenStr string) {
	// 	token := &claims.Token{
	// 		TokenStr: tokenStr,
	// 		Secret:   secret,
	// 	}
	// 	_, err := token.ParseAndVerifyJWT(&claims.VerifyOptions{SkipExpiryCheck: false})
	// 	if err != nil {
	// 		t.Errorf("Error validating verification token: %q", err)
	// 	}
}
