package posts_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/dbentities"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/DagDigg/unpaper/backend/posts"
)

func TestCreatePost(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	dir := getPostsDir(t)
	audio := &v1API.Audio{
		Id:    "foo",
		Bytes: []byte("bar"),
	}
	rawJSONAudio, err := dbentities.NewAudioRawJSON(audio)
	assert.Nil(err)

	fmt.Printf("%v", rawJSONAudio)
	t.Run("When creating a post", func(t *testing.T) {
		params := posts.CreatePostParams{
			ID:      uuid.NewString(),
			Author:  uuid.NewString(),
			Message: uuid.NewString(),
			Audio:   rawJSONAudio,
		}

		post, err := dir.CreatePost(context.Background(), params)
		assert.Nil(err)

		prt, _ := json.MarshalIndent(post.Audio, "", " ")
		fmt.Println(string(prt))

		assert.Equal(params.ID, post.Id)
		assert.Equal(audio, post.Audio)
		assert.Equal(params.Message, post.Message)
	})
}

func TestLikePost(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	dir := getPostsDir(t)
	audio := &v1API.Audio{
		Id:    "foo",
		Bytes: []byte("bar"),
	}
	rawJSONAudio, err := dbentities.NewAudioRawJSON(audio)
	assert.Nil(err)

	t.Run("When creating, liking and disliking a post", func(t *testing.T) {
		params := posts.CreatePostParams{
			ID:      uuid.NewString(),
			Author:  uuid.NewString(),
			Message: uuid.NewString(),
			Audio:   rawJSONAudio,
		}

		post, err := dir.CreatePost(context.Background(), params)
		assert.Nil(err)

		userWhoLiked := uuid.NewString()
		likeParams := posts.LikePostParams{
			ID:      post.Id,
			Column1: userWhoLiked,
		}
		likedPost, err := dir.LikePost(context.Background(), likeParams)
		assert.Nil(err)

		assert.NotNil(likedPost)
		assert.Equal(int32(1), likedPost.Likes)

		ok, err := dir.HasUserLikedPost(context.Background(), posts.HasUserLikedPostParams{
			ID:      likedPost.Id,
			Column2: userWhoLiked,
		})
		assert.Nil(err)

		assert.True(ok)

		// Remove like
		dislikeParams := posts.RemoveLikeFromPostParams{
			ID:      post.Id,
			Column1: userWhoLiked,
		}
		updatedLikedPost, err := dir.RemoveLikeFromPost(context.Background(), dislikeParams)
		assert.Nil(err)

		assert.Equal(int32(0), updatedLikedPost.Likes)
		assert.Equal(false, updatedLikedPost.HasAlreadyLiked)
	})

}

// getPostsDir returns the *posts.Directory
func getPostsDir(t *testing.T) *posts.Directory {
	ws := v1Testing.GetWrappedServer(t)
	return posts.NewDirectory(ws.Server.GetDB())
}
