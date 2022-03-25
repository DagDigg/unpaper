package session

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/ksuid"
)

// Assert that implements interface
var _ Session = &Manager{}

// Manager implements `Session` interface.
// It deals with the underlying redis client
type Manager struct {
	rdb *redis.Client
}

// NewManager returns a new `Manager` instance
func NewManager(rdb *redis.Client) *Manager {
	return &Manager{
		rdb: rdb,
	}
}

// SetNew sets a new user session into the in-memory store.
// It returns the generated session id
func (m *Manager) SetNew(ctx context.Context, u *User, expiry time.Duration) (string, error) {
	sid := ksuid.New().String()
	encodedUser, err := u.EncodeBinary()
	if err != nil {
		return "", err
	}

	// Update sessions by userID
	_, err = m.rdb.SAdd(ctx, u.ID, sid).Result()
	if err != nil {
		return "", err
	}

	// Set session for the user with expiry
	_, err = m.rdb.Set(ctx, sid, encodedUser, expiry).Result()
	if err != nil {
		return "", err
	}

	return sid, nil
}

// Delete removes the sid from the in-memory db. It then syncs the user sessions.
func (m *Manager) Delete(ctx context.Context, sid string) error {
	// Retrieve user by SID
	u, err := m.GetUserBySID(ctx, sid)
	if err != nil {
		return err
	}

	// Delete session
	_, err = m.rdb.Del(ctx, sid).Result()
	if err != nil {
		return err
	}

	// Sync user session. This will delete the recently deleted
	// sid from the user sessions list
	return m.Sync(ctx, u.ID)
}

// HasSession checks if a session exists on the in-memory db
func (m *Manager) HasSession(ctx context.Context, sid string) (bool, error) {
	_, err := m.rdb.Get(ctx, sid).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetUserBySID returns an `User` session instance from the in-memory store associated with the session ID
func (m *Manager) GetUserBySID(ctx context.Context, sid string) (*User, error) {
	// Get encoded user by session id
	encodedUser, err := m.rdb.Get(ctx, sid).Result()
	if err != nil {
		return nil, err
	}

	// Decode encoded user
	u := &User{}
	if err := u.DecodeBinary(encodedUser); err != nil {
		return nil, err
	}

	return u, nil
}

// Sync searches for expired sessions for the userID and updates the in-memory store accordingly.
// Since only singular sessions have an expiry, the list of session for the user id must be synced periodically,
// to keep it updated
func (m *Manager) Sync(ctx context.Context, userID string) error {
	sids, err := m.rdb.SMembers(ctx, userID).Result()
	if err != nil {
		return err
	}

	// Iterate over user session ids and remove non-existent ones.
	for _, sid := range sids {
		_, err := m.rdb.Get(ctx, sid).Result()
		if err != nil {
			if err == redis.Nil {
				// Session doesn't exist anymore. It should be removed
				m.rdb.SRem(ctx, userID, sid)
				continue
			}
			return err
		}
	}

	return nil
}

// RenewSession deletes the current session id from the in-memory store, and creates a new one
func (m *Manager) RenewSession(ctx context.Context, sid string, expiry time.Duration) (string, error) {
	// Retrieve user
	u, err := m.GetUserBySID(ctx, sid)
	if err != nil {
		return "", err
	}

	// Remove session id
	_, err = m.rdb.Del(ctx, sid).Result()
	if err != nil {
		return "", err
	}

	// Remove session id from user session id list
	_, err = m.rdb.SRem(ctx, u.ID, sid).Result()
	if err != nil {
		return "", err
	}

	// Recreate and return a new session
	return m.SetNew(ctx, &User{ID: u.ID}, expiry)
}
