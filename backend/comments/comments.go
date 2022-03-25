package comments

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
func (d Directory) Close() error {
	return d.db.Close()
}

// ThreadType refers to which kind of thread the comment belongs
type ThreadType string

const (
	// ThreadTypeNone refers to comments that are not threaded
	ThreadTypeNone ThreadType = "none"
	// ThreadTypeComment refers to comments that threaded to the comment
	ThreadTypeComment ThreadType = "comment"
	// ThreadTypePost refers to comments that threaded to the post
	ThreadTypePost ThreadType = "post"
)

// CreateComment INSERTs a comment in db
func (d *Directory) CreateComment(ctx context.Context, args CreateCommentParams) (*v1API.Comment, error) {
	res, err := d.querier.CreateComment(ctx, args)
	if err != nil {
		return nil, err
	}

	return pgCommentToPB(pgCommentToPBParams{
		c:               res,
		hasAlreadyLiked: false,
	})
}

// GetComments returns the database comments for a postID sorted hierarchically and by votes
func (d *Directory) GetComments(ctx context.Context, userID string, postID string) ([]*v1API.Comment, error) {
	res, err := d.querier.GetComments(ctx, postID)
	if err != nil {
		return nil, err
	}
	cmts := []*v1API.Comment{}
	for _, c := range res {
		pbCmt, err := pgCommentToPB(pgCommentToPBParams{
			c:               c,
			hasAlreadyLiked: hasAlreadyLiked(c.UserIdsWhoLikes, userID),
		})
		if err != nil {
			return nil, err
		}
		cmts = append(cmts, pbCmt)
	}

	return cmts, nil
}

// LikeComment increments the likes of the comment
func (d *Directory) LikeComment(ctx context.Context, params LikeCommentParams) (*v1API.Comment, error) {
	res, err := d.querier.LikeComment(ctx, params)
	if err != nil {
		return nil, err
	}

	return pgCommentToPB(pgCommentToPBParams{
		c:               res,
		hasAlreadyLiked: hasAlreadyLiked(res.UserIdsWhoLikes, params.UserID),
	})
}

// RemoveLikeFromComment decrements the likes of the comment
func (d *Directory) RemoveLikeFromComment(ctx context.Context, params RemoveLikeFromCommentParams) (*v1API.Comment, error) {
	res, err := d.querier.RemoveLikeFromComment(ctx, params)
	if err != nil {
		return nil, err
	}

	return pgCommentToPB(pgCommentToPBParams{
		c:               res,
		hasAlreadyLiked: hasAlreadyLiked(res.UserIdsWhoLikes, params.UserID),
	})
}

// HasUserLikedComment returns whether the user has already liked the comment
func (d *Directory) HasUserLikedComment(ctx context.Context, params HasUserLikedCommentParams) (bool, error) {
	ok, err := d.querier.HasUserLikedComment(ctx, params)
	if err != nil {
		return false, err
	}

	return ok, nil
}
