package v1_test

import (
	"context"
	"testing"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestGetPost(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When creating a post", func(t *testing.T) {
		t.Parallel()
		userID := uuid.NewString()
		ctx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs(
				"x-user-id", userID,
			),
		)

		// Create a post
		createPostReq := &v1API.CreatePostRequest{
			Message:    "msg",
			AudioBytes: []byte("foo"),
		}
		createPostRes, err := ws.Server.CreatePost(ctx, createPostReq)
		assert.Nil(err)
		assert.Equal(createPostReq.AudioBytes, createPostRes.Post.Audio.Bytes)
		assert.NotEmpty(createPostRes.Post.Audio.Id)
		assert.Equal(createPostReq.Message, createPostRes.Post.Message)
		assert.Equal(userID, createPostRes.Post.Author)
		assert.NotNil(userID, createPostRes.Post.Id)

		// Get posts
		getPostsReq := &v1API.GetPostsRequest{
			Category: "foo",
		}
		getPostsRes, err := ws.Server.GetPosts(ctx, getPostsReq)
		assert.Nil(err)
		assert.Equal(getPostsRes.Posts[0].Id, createPostRes.Post.Id)
	})
}

func TestCreateComment(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When successfully creating comments", func(t *testing.T) {
		t.Parallel()
		userID := uuid.NewString()
		ctx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs(
				"x-user-id", userID,
			),
		)

		// Create a post
		createPostReq := &v1API.CreatePostRequest{
			Message:    "msg",
			AudioBytes: []byte("foo"),
		}
		createPostRes, err := ws.Server.CreatePost(ctx, createPostReq)
		assert.Nil(err)

		tests := []struct {
			in   *v1API.CreateCommentRequest
			want *v1API.CreateCommentResponse
		}{
			{
				in: &v1API.CreateCommentRequest{
					PostId:     createPostRes.Post.Id,
					Message:    "",
					AudioBytes: []byte("foo"),
					ParentId:   "",
					Thread: &v1API.ThreadRequest{
						ThreadType: v1API.ThreadType_POST,
					},
				},
				want: &v1API.CreateCommentResponse{
					Comment: &v1API.Comment{
						Message: "",
						Audio: &v1API.Audio{
							Bytes: []byte("foo"),
						},
						Author:   userID,
						ParentId: "",
						Likes:    0,
						PostId:   createPostRes.Post.Id,
						Thread: &v1API.Thread{
							TargetId:   "",
							ThreadType: v1API.ThreadType_POST,
						},
					},
				},
			},
			{
				in: &v1API.CreateCommentRequest{
					PostId:     createPostRes.Post.Id,
					Message:    "foo",
					AudioBytes: []byte("foo"),
					ParentId:   "",
				},
				want: &v1API.CreateCommentResponse{
					Comment: &v1API.Comment{
						Message: "foo",
						Audio: &v1API.Audio{
							Bytes: []byte("foo"),
						},
						Author:   userID,
						ParentId: "",
						Likes:    0,
						PostId:   createPostRes.Post.Id,
						Thread: &v1API.Thread{
							TargetId:   "",
							ThreadType: v1API.ThreadType_NONE,
						},
					},
				},
			},
		}

		for _, tt := range tests {
			res, err := ws.Server.CreateComment(
				metadata.NewIncomingContext(
					context.Background(),
					metadata.Pairs(
						"x-user-id", userID, // We're using post author user id
					),
				), tt.in)
			assert.Nilf(err, "%s", err)

			assert.NotNil(res)
			assert.NotEmpty(res.Comment.Id)
			assert.Equal(tt.in.PostId, res.Comment.PostId)
			assert.Equal(tt.in.Message, res.Comment.Message)
			assert.Equal(tt.in.ParentId, res.Comment.ParentId)
			assert.Equal(tt.in.AudioBytes, res.Comment.Audio.Bytes)
		}
	})
}

func TestLikePost(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When creating a post, liking and disliking it", func(t *testing.T) {
		t.Parallel()
		userID := uuid.NewString()
		ctx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs(
				"x-user-id", userID,
			),
		)

		// Create a post
		createPostReq := &v1API.CreatePostRequest{
			Message:    "msg",
			AudioBytes: []byte("foo"),
		}
		createPostRes, err := ws.Server.CreatePost(ctx, createPostReq)
		assert.Nil(err)

		// Like post
		likePostReq := &v1API.LikePostRequest{
			PostId: createPostRes.Post.Id,
		}
		likePostRes, err := ws.Server.LikePost(ctx, likePostReq)
		assert.True(likePostRes.Post.HasAlreadyLiked)
		assert.Equal(int32(1), likePostRes.Post.Likes)

		// Dislike post
		dislikePostRes, err := ws.Server.LikePost(ctx, likePostReq)
		assert.False(dislikePostRes.Post.HasAlreadyLiked)
		assert.Equal(int32(0), dislikePostRes.Post.Likes)
	})
}
