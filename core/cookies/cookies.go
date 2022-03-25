package cookies

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// AccessTknExpiry refers to the access token cookie lifespan
	AccessTknExpiry = 24 * time.Hour * 30
	// VerificationTknExpiry refers to the verification voken cookie lifespan
	VerificationTknExpiry = 24 * time.Hour
)

// Manager represents the cookie manager that will interact with the domains
type Manager struct {
	// Domain to which the cookies will be managed
	Domain string
}

// GetValue returns an HTTP string used as header value in an HTTP response.
// It returns an error if the key or val is an empty string
func (m *Manager) GetValue(key, val string, expiry time.Duration) (string, error) {
	if key == "" || val == "" {
		return "", fmt.Errorf("missing cookie key or value")
	}
	// Get max age. Max-Age HTTP header defines the
	// token lifespan in seconds.
	maxAge := expiry.Seconds()
	maxAgeStr := strconv.Itoa(int(maxAge))

	return fmt.Sprintf("%s=%s; Domain=%s; Path=/; Max-Age=%s; Secure; HttpOnly", key, val, m.Domain, maxAgeStr), nil
}

// DeleteCookie returns an HTTP cookie value with the provided key and expiration set in the past.
// It returns an error if the key is an empty string
func (m *Manager) DeleteCookie(key string) (string, error) {
	// Key is mandatory
	if key == "" {
		return "", fmt.Errorf("missing cookie key")
	}

	return fmt.Sprintf("%s=deleted; Domain=%s; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT", key, m.Domain), nil
}

// FindAttribute extracts the cookie attribute value in 's' and returns it
func FindAttribute(cookie, key string) (string, bool) {
	if cookie == "" {
		return "", false
	}
	attributes := strings.Split(cookie, ";")

	for _, attr := range attributes {
		attribute := strings.TrimSpace(attr)
		if strings.Contains(strings.ToLower(attribute), strings.ToLower(key)) {
			// Cookie found, get the index of equal operator
			idx := strings.Index(attribute, "=")
			if idx == -1 {
				// Malformed cookie
				return "", false
			}

			attrVal := attribute[idx+1:]
			if attrVal == "" {
				// Scenario which a cookie is defined as: 'cookie=' .
				// In that case, a cookie is treated as not found
				return "", false
			}

			// Return cookie value, which is everything after '='
			return attrVal, true
		}
	}

	return "", false
}
