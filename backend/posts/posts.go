package posts

import (
	"context"
	"database/sql"
	"fmt"

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

// CreatePost INSERTs a new post into db
func (d *Directory) CreatePost(ctx context.Context, args CreatePostParams) (*v1API.Post, error) {
	res, err := d.querier.CreatePost(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("error creating postgres post: %v", err)
	}

	return pgPostToPB(res)
}

// GetPost returns the db post by ID
func (d *Directory) GetPost(ctx context.Context, postID string) (*v1API.Post, error) {
	res, err := d.querier.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}

	return pgPostToPB(res)
}

// GetPosts returns all the posts
func (d *Directory) GetPosts(ctx context.Context) ([]*v1API.Post, error) {
	res, err := d.querier.GetPosts(ctx)
	if err != nil {
		return nil, err
	}

	posts := []*v1API.Post{}

	for _, p := range res {
		post, err := pgPostToPB(p)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// LikePost increments the likes of the post and adds the userid who liked to the array of users who liked column
func (d *Directory) LikePost(ctx context.Context, params LikePostParams) (*v1API.Post, error) {
	res, err := d.querier.LikePost(ctx, params)
	if err != nil {
		return nil, err
	}

	return pgPostToPB(res)
}

// RemoveLikeFromPost decrements the likes of the post and removes the userid who liked from the array of users who liked column
func (d *Directory) RemoveLikeFromPost(ctx context.Context, params RemoveLikeFromPostParams) (*v1API.Post, error) {
	res, err := d.querier.RemoveLikeFromPost(ctx, params)
	if err != nil {
		return nil, err
	}

	return pgPostToPB(res)
}

// HasUserLikedPost returns whether users_who_liked_post contains the passed user_id
func (d *Directory) HasUserLikedPost(ctx context.Context, params HasUserLikedPostParams) (bool, error) {
	ok, err := d.querier.HasUserLikedPost(ctx, params)
	if err != nil {
		return false, err
	}

	return ok, nil
}

// GetTrendingTodayPosts returns today trending posts
func (d *Directory) GetTrendingTodayPosts(ctx context.Context) ([]*v1API.Post, error) {
	res, err := d.querier.GetTrendingTodayPosts(ctx)
	if err != nil {
		return nil, err
	}

	return trendingTodayPostsListToPB(res)
}

// GetTrendingTodayPostIDs returns today trending posts ids
func (d *Directory) GetTrendingTodayPostIDs(ctx context.Context) ([]string, error) {
	res, err := d.querier.GetTrendingTodayPostIDs(ctx)
	if err != nil {
		return nil, err
	}

	return res, nil
}
