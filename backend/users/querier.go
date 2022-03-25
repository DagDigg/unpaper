// Code generated by sqlc. DO NOT EDIT.

package users

import (
	"context"
	"database/sql"
)

type Querier interface {
	CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
	DeleteUser(ctx context.Context, id string) error
	EmailExists(ctx context.Context, email string) (bool, error)
	GetEmail(ctx context.Context, id string) (string, error)
	GetEmailVerified(ctx context.Context, id string) (sql.NullBool, error)
	GetPassword(ctx context.Context, id string) (sql.NullString, error)
	GetUser(ctx context.Context, id string) (User, error)
	GetUserByUsername(ctx context.Context, username sql.NullString) (User, error)
	GetUserIDFromEmail(ctx context.Context, email string) (string, error)
	GetUserSuggestions(ctx context.Context, lower string) ([]User, error)
	UpdatePassword(ctx context.Context, arg UpdatePasswordParams) error
	UpdatePasswordChangedAt(ctx context.Context, arg UpdatePasswordChangedAtParams) error
	UpdateUsername(ctx context.Context, arg UpdateUsernameParams) (User, error)
	UserIDExists(ctx context.Context, id string) (bool, error)
	UsernameExists(ctx context.Context, username sql.NullString) (bool, error)
	VerifyEmail(ctx context.Context, id string) error
}

var _ Querier = (*Queries)(nil)
