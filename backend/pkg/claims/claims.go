package claims

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Claims represent the unpaper JWT Claims structure
type Claims struct {
	jwt.StandardClaims
}

// Token represents the claims.Token structure
type Token struct {
	// JWT access token
	AccessToken string
	// Secret for validating the signature
	Secret string
}

const (
	// VerificationTknExpiry denotes the verification token JWT expiration
	VerificationTknExpiry = 24 * time.Hour * 7
)

// VerifyOptions to be passed to claims.ParseAndVerifyJWT
type VerifyOptions struct {
	// If SkipExpiryCheck is set to 'true', the validation ignore expired tokens
	SkipExpiryCheck bool
}

// ParseAndVerifyJWT parses the JWT opt.AccessToken into the *claims, verifying it with the provided opt.Secret
// and returning any error encountered
func (t *Token) ParseAndVerifyJWT(opt *VerifyOptions, c jwt.Claims) (jwt.Claims, error) {
	return t.parseAndVerifyJWT(opt, c)
}

func (t *Token) parseAndVerifyJWT(opt *VerifyOptions, c jwt.Claims) (jwt.Claims, error) {
	_, err := jwt.ParseWithClaims(t.AccessToken, c, func(token *jwt.Token) (interface{}, error) {
		// Validate alg
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(t.Secret), nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		v, _ := err.(*jwt.ValidationError)
		if v.Errors == jwt.ValidationErrorExpired {
			if opt.SkipExpiryCheck {
				// No error if expiry should be ignored
				return c, nil
			}
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return c, nil
}

// CreateJWTTokenStr creates a JWT token with a specified duration and returns its string value
func CreateJWTTokenStr(c jwt.Claims, secret string) (string, error) {
	// Declare token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	//Create JWT string
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		// Error during creation of JWT
		return "", err
	}

	return tokenStr, nil
}
