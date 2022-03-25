// Code generated by sqlc. DO NOT EDIT.

package comments

import (
	"context"
)

type Querier interface {
	CreateComment(ctx context.Context, arg CreateCommentParams) (Comment, error)
	GetComments(ctx context.Context, postID string) ([]Comment, error)
	HasUserLikedComment(ctx context.Context, arg HasUserLikedCommentParams) (bool, error)
	LikeComment(ctx context.Context, arg LikeCommentParams) (Comment, error)
	RemoveLikeFromComment(ctx context.Context, arg RemoveLikeFromCommentParams) (Comment, error)
}

var _ Querier = (*Queries)(nil)
