package session

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"time"
)

const (
	// KeyByUserID in-memory store dictionary
	// key used for grouping sessions by userID
	KeyByUserID string = "by_user_id"
	// KeyBySessionID in-memory store dictionary
	// key used for setting a user to a sessionID
	KeyBySessionID string = "by_session_id"

	// CookieName represents the name of the cookie related to the current session
	CookieName = "__unpaperSID"

	// HeaderName denotes the http header name associated to the session ID
	HeaderName = "x-unpaper-sid"

	// Lifetime denotes the session duration.
	// After the expiry, if not refreshed, the session is removed
	// from the in-memory store and the user needs to login again
	Lifetime = 24 * time.Hour

	// TTLRefresh denotes the Time To Live for a key
	// in order to be able to be refreshed. Eg: with a SessionTTLRefresh of 10 seconds,
	// if a key has a TTL of 5 seconds, it's eligible for refresh. If it has a TTL of 15 seconds, it is not
	TTLRefresh = 30 * time.Second
)

// Session describes the behaviour
// of a in-memory session management
type Session interface {
	SetNew(ctx context.Context, u *User, expiry time.Duration) (string, error)
	Delete(ctx context.Context, sid string) error
	GetUserBySID(ctx context.Context, sessionID string) (*User, error)
	Sync(ctx context.Context, userID string) error
	RenewSession(ctx context.Context, sid string, expiry time.Duration) (string, error)
	HasSession(ctx context.Context, sid string) (bool, error)
}

// User refers to a user session
type User struct {
	ID string `json:"id"`
}

// EncodeBinary encodes the `User` data structure in a base64 encoded string
func (u *User) EncodeBinary() (string, error) {
	b := &bytes.Buffer{}
	e := gob.NewEncoder(b)
	if err := e.Encode(u); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

// DecodeBinary decodes base64 encoded `User` string into `User` data structure
func (u *User) DecodeBinary(str string) error {
	// Decode base64 string
	p, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return err
	}

	// Create a buffer and write the decoded base64 string bytes
	b := &bytes.Buffer{}
	_, err = b.Write(p)
	if err != nil {
		return err
	}

	// Decode buffer into `User`
	d := gob.NewDecoder(b)
	if err := d.Decode(u); err != nil {
		return err
	}

	return nil
}
