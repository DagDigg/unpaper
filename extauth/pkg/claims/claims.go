package claims

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	// JWT refresh token lifespan
	RefreshTknExpiry = 24 * time.Hour * 30
	// JWT access token lifespan
	AccessTknExpiry = 10 * time.Minute
)

// Claims represent the unpaper JWT Claims structure
type Claims struct {
	jwt.StandardClaims
}

// Token represents the claims.Token structure
type Token struct {
	// JWT token
	TokenStr string
	// Secret for validating the signature
	Secret string
	// Expiration of the token
	Expiry time.Time
	// Timestamp when the token has been issued
	IssuedAt time.Time
}

// VerifyOptions to be passed to claims.ParseAndVerifyJWT
type VerifyOptions struct {
	// If SkipExpiryCheck is set to 'true', the validation ignore expired tokens
	SkipExpiryCheck bool
}

// ParseAndVerifyJWT parses the JWT opt.AccessToken into the *claims, verifying it with the provided opt.Secret
// and returning any error encountered
func (t *Token) ParseAndVerifyJWT(opt *VerifyOptions) (*Claims, error) {
	return t.parseAndVerifyJWT(opt)
}

func (t *Token) parseAndVerifyJWT(opt *VerifyOptions) (*Claims, error) {
	c := &Claims{}
	_, err := jwt.ParseWithClaims(t.TokenStr, c, func(token *jwt.Token) (interface{}, error) {
		// Validate alg
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(t.Secret), nil
	})

	if err != nil {
		v, _ := err.(*jwt.ValidationError)
		if v.Errors == jwt.ValidationErrorExpired {
			// Token is expired
			if opt.SkipExpiryCheck {
				// No error if expiry should be ignored
				return c, nil
			}
			return c, &ErrTokenExpired{}
		}

		// Other error have occurred
		return nil, err
	}

	// Validation successful
	return c, nil
}

// GetUserID parses and verify the access token, ignoring its expiration, and returns its Subject
func (t *Token) GetUserID() (string, error) {
	c, err := t.parseAndVerifyJWT(&VerifyOptions{SkipExpiryCheck: true})
	if err != nil {
		return "", fmt.Errorf(err.Error())
	}
	return c.Subject, nil
}

// ErrTokenExpired is a custom error returned
// on JWT validation when the token is expired
type ErrTokenExpired struct{}

func (e *ErrTokenExpired) Error() string {
	return "Token is expired"
}

// CreateJWTTokenStr creates a JWT token with a specified duration and returns its string value
func CreateJWTTokenStr(expiry time.Duration, userID, secret, iss string) (string, error) {
	expTime := time.Now().Add(expiry)
	claims := &Claims{
		StandardClaims: jwt.StandardClaims{
			Subject: userID,
			// Expiration must be in unix milliseconds.
			ExpiresAt: expTime.Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    iss,
		},
	}

	// Declare token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	//Create JWT string
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		// Error during creation of JWT
		return "", err
	}

	return tokenStr, nil
}

// TokenPair contains access and refresh token.
// The tokens are custom generated ones
type TokenPair struct {
	// Complete signed access token
	AccessToken *Token
	// Complete signed refresh token
	RefreshToken *Token
}

// GetTokenPair creates an access token and a refresh token.
// These tokens are created internally, so the provided secret must be the custom one,
// and not an external one (e.g: google, facebook etc..)
func GetTokenPair(subject, secret, issuer string) (*TokenPair, error) {
	// Create tokens
	refreshTknStr, err := CreateJWTTokenStr(RefreshTknExpiry, subject, secret, issuer)
	if err != nil {
		return nil, err
	}
	accessTknStr, err := CreateJWTTokenStr(AccessTknExpiry, subject, secret, issuer)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  &Token{TokenStr: accessTknStr, Expiry: time.Now().Add(AccessTknExpiry).UTC()},
		RefreshToken: &Token{TokenStr: refreshTknStr, Expiry: time.Now().Add(RefreshTknExpiry).UTC()},
	}, nil
}
