package mixes

import (
	"context"
	"database/sql"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/Masterminds/squirrel"
)

// Directory is the directory which operates on db table 'users'
type Directory struct {
	// querier is an interface containing all of the
	// directory methods. Must be created with users.New(db)
	querier Querier
	db      *sql.DB
	sb      squirrel.StatementBuilderType
}

// NewDirectory creates a new users directory
func NewDirectory(db *sql.DB) *Directory {
	return &Directory{db: db, querier: New(db)}
}

// Close closes Directory database connection
func (d *Directory) Close() error {
	return d.db.Close()
}

// CreateUserMix creates a new Mix for the user
func (d *Directory) CreateUserMix(ctx context.Context, params CreateUserMixParams) (*v1API.Mix, error) {
	res, err := d.querier.CreateUserMix(ctx, params)
	if err != nil {
		return nil, err
	}

	return pgMixToPB(res)
}

// GetUserMixes returns the user's mixes
func (d *Directory) GetUserMixes(ctx context.Context, userID string) ([]*v1API.Mix, error) {
	res, err := d.querier.GetUserMixes(ctx, userID)
	if err != nil {
		return nil, err
	}

	return pgMixListToPB(res)
}

// DeleteUserMixes deletes all the mixes associated with the user id
func (d *Directory) DeleteUserMixes(ctx context.Context, userID string) error {
	return d.querier.DeleteUserMixes(ctx, userID)
}
