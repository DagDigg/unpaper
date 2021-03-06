// Code generated by sqlc. DO NOT EDIT.

package follows

import (
	"context"
)

type Querier interface {
	FollowUser(ctx context.Context, arg FollowUserParams) (FollowUserRow, error)
	GetFollowers(ctx context.Context, followingUserID string) ([]GetFollowersRow, error)
	GetFollowersCount(ctx context.Context, followingUserID string) (int64, error)
	GetFollowing(ctx context.Context, followerUserID string) ([]GetFollowingRow, error)
	GetFollowingCount(ctx context.Context, followerUserID string) (int64, error)
	IsFollowingUser(ctx context.Context, arg IsFollowingUserParams) (bool, error)
	UnfollowUser(ctx context.Context, arg UnfollowUserParams) (UnfollowUserRow, error)
}

var _ Querier = (*Queries)(nil)
