package lists

import (
	"context"
	"database/sql"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"

	"github.com/Masterminds/squirrel"

	// pgx is a postgres driver
	_ "github.com/jackc/pgx/v4/stdlib"
)

// Directory is the directory which operates on db table 'users'
type Directory struct {
	// querier is an interface containing all of the
	// directory methods. Must be created with customers.NewDirectory(db)
	querier Querier
	db      *sql.DB
	sb      squirrel.StatementBuilderType
}

// NewDirectory creates a new lists directory
func NewDirectory(db *sql.DB) *Directory {
	return &Directory{db: db, querier: New(db)}
}

// Close closes Directory database connection
func (d Directory) Close() error {
	return d.db.Close()
}

// CreateList inserts a list into db
func (d Directory) CreateList(ctx context.Context, params *CreateListParams) (*v1API.List, error) {
	res, err := d.querier.CreateList(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgListToPB(res)
}

// UpdateAllowedUsers updates the allowed_users column for the specified list id
func (d Directory) UpdateAllowedUsers(ctx context.Context, params *UpdateAllowedUsersParams) (*v1API.List, error) {
	res, err := d.querier.UpdateAllowedUsers(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgListToPB(res)
}

// UpdateName updates the name column for the specified list id
func (d Directory) UpdateName(ctx context.Context, params *UpdateNameParams) (*v1API.List, error) {
	res, err := d.querier.UpdateName(ctx, *params)
	if err != nil {
		return nil, err
	}

	return pgListToPB(res)
}

// GetListsByOwnerID returns a slice of lists for the user ID
func (d Directory) GetListsByOwnerID(ctx context.Context, userID string) ([]*v1API.List, error) {
	res, err := d.querier.GetListsByOwnerID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var allLists []*v1API.List
	for _, v := range res {
		pbList, err := pgListToPB(v)
		if err != nil {
			return nil, err
		}
		allLists = append(allLists, pbList)
	}

	return allLists, nil
}

// GetListByID returns a lits by its ID
func (d Directory) GetListByID(ctx context.Context, listID string) (*v1API.List, error) {
	res, err := d.querier.GetListByID(ctx, listID)
	if err != nil {
		return nil, err
	}

	return pgListToPB(res)
}
