package usersession

import (
	"context"

	"github.com/go-redis/redis/v8"
)

// Manager structure to interact with user session
type Manager struct {
	rdb *redis.Client
}

// NewManager returns a Manager instance
func NewManager(rdb *redis.Client) Sessioner {
	return &Manager{
		rdb: rdb,
	}
}

// Sessioner is interface for the behaviour of a Login-Logout-IsOnline redis mechanism
type Sessioner interface {
	Log(ctx context.Context, userID string) error
	Unlog(ctx context.Context, userID string) error
	IsOnline(ctx context.Context, userID string) (bool, error)
}

const (
	prefixActiveUsers = "user:online:"
)

// Log adds the specified userID to the underlying `users:` redis set
func (m *Manager) Log(ctx context.Context, userID string) error {
	return m.rdb.SAdd(ctx, prefixActiveUsers, userID).Err()
}

// Unlog removes the specified userID to the underlying `users:` redis set
func (m *Manager) Unlog(ctx context.Context, userID string) error {
	return m.rdb.SRem(ctx, prefixActiveUsers, userID).Err()
}

// IsOnline returns whether the user is available in the underlying redis set
func (m *Manager) IsOnline(ctx context.Context, userID string) (bool, error) {
	return m.rdb.SIsMember(ctx, prefixActiveUsers, userID).Result()
}
