package codeverifier

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const (
	// DefaultLength is the default code verifier length in bytes
	DefaultLength = 32
	// MinLength is the minimum code verifier length in bytes
	MinLength = 32
	// MaxLength is the maximum code verifier length in bytes
	MaxLength = 96
)

// CodeVerifier contains methods for
// creating a pkce code_verifier and code_challenge
type CodeVerifier struct {
	Value string
}

// CreateCodeVerifier creates a CodeVerifier
func CreateCodeVerifier() (*CodeVerifier, error) {
	return CreateCodeVerifierWithLength(DefaultLength)
}

// CreateCodeVerifierWithLength creates a CodeVerifier with a custom byte length
func CreateCodeVerifierWithLength(length int) (*CodeVerifier, error) {
	if length < MinLength || length > MaxLength {
		return nil, fmt.Errorf("invalid length: %v", length)
	}
	buf, err := randomBytes(length)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %v", err)
	}
	return CreateCodeVerifierFromBytes(buf)
}

// CreateCodeVerifierFromBytes encodes the bytes buffer into CodeVerifier Value field to base64
func CreateCodeVerifierFromBytes(b []byte) (*CodeVerifier, error) {
	return &CodeVerifier{
		Value: encode(b),
	}, nil
}

func (v *CodeVerifier) String() string {
	return v.Value
}

// CodeChallengePlain returns the CodeVerifier inner Value
func (v *CodeVerifier) CodeChallengePlain() string {
	return v.Value
}

// CodeChallengeS256 returns the sha256 value of CodeVerifier inner Value
func (v *CodeVerifier) CodeChallengeS256() string {
	h := sha256.New()
	h.Write([]byte(v.Value))
	return encode(h.Sum(nil))
}

func encode(msg []byte) string {
	encoded := base64.StdEncoding.EncodeToString(msg)
	encoded = strings.Replace(encoded, "+", "-", -1)
	encoded = strings.Replace(encoded, "/", "_", -1)
	encoded = strings.Replace(encoded, "=", "", -1)
	return encoded
}

// https://tools.ietf.org/html/rfc7636#section-4.1)
func randomBytes(length int) ([]byte, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	const csLen = byte(len(charset))
	output := make([]byte, 0, length)
	for {
		buf := make([]byte, length)
		if _, err := io.ReadFull(rand.Reader, buf); err != nil {
			return nil, fmt.Errorf("failed to read random bytes: %v", err)
		}
		for _, b := range buf {
			// Avoid bias by using a value range that's a multiple of 62
			if b < (csLen * 4) {
				output = append(output, charset[b%csLen])

				if len(output) == length {
					return output, nil
				}
			}
		}
	}
}
