package follows

import (
	"context"
	"database/sql"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/Masterminds/squirrel"
)

// Directory is the directory which operates on db table 'follows'
type Directory struct {
	// querier is an interface containing all of the
	// directory methods. Must be created with customers.NewDirectory(db)
	querier Querier
	db      *sql.DB
	sb      squirrel.StatementBuilderType
}

// NewDirectory creates a new users directory
func NewDirectory(db *sql.DB) *Directory {
	return &Directory{db: db, querier: New(db)}
}

// Close closes Directory database connection
func (d Directory) Close() error {
	return d.db.Close()
}

// FollowUser follows the specified user updating the `follows` table and returning an external user info
func (d Directory) FollowUser(ctx context.Context, params FollowUserParams) (*v1API.ExtUserInfo, error) {
	res, err := d.querier.FollowUser(ctx, params)
	if err != nil {
		return nil, err
	}

	return pgFollowUserRowToPB(res), nil
}

// UnfollowUser unfollows the specified user updating the `follows` table and returning an external user info
func (d Directory) UnfollowUser(ctx context.Context, params UnfollowUserParams) (*v1API.ExtUserInfo, error) {
	res, err := d.querier.UnfollowUser(ctx, params)
	if err != nil {
		return nil, err
	}

	return pgUnfollowUserRowToPB(res), nil
}

// IsFollowingUser returns whether the user follows the target
func (d Directory) IsFollowingUser(ctx context.Context, params IsFollowingUserParams) (bool, error) {
	return d.querier.IsFollowingUser(ctx, params)
}

// GetFollowers returns a list of ext user info who follows the passed param userID
func (d Directory) GetFollowers(ctx context.Context, userID string) ([]*v1API.ExtUserInfo, error) {
	res, err := d.querier.GetFollowers(ctx, userID)
	if err != nil {
		return nil, err
	}

	return pgFollowersListToPB(res), nil
}

// GetFollowing returns a list of ext user info the passed param userID is following
func (d Directory) GetFollowing(ctx context.Context, userID string) ([]*v1API.ExtUserInfo, error) {
	res, err := d.querier.GetFollowing(ctx, userID)
	if err != nil {
		return nil, err
	}

	return pgFollowingListToPB(res), nil
}

// GetFollowingCount returns the number of users the passed param userID is following
func (d Directory) GetFollowingCount(ctx context.Context, userID string) (int64, error) {
	return d.querier.GetFollowingCount(ctx, userID)
}

// GetFollowersCount returns the number of users the passed param userID has as followers
func (d Directory) GetFollowersCount(ctx context.Context, userID string) (int64, error) {
	return d.querier.GetFollowersCount(ctx, userID)
}
