package comments_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/DagDigg/unpaper/backend/comments"
)

func TestGetComments(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	dir := comments.NewDirectory(ws.Server.GetDB())
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	t.Cleanup(func() {
		err := dir.Close()
		if err != nil {
			t.Fatalf("error closing directory: %v", err)
		}
	})

	t.Run("When retrieving hierarchically ordered comments", func(t *testing.T) {
		t.Parallel()
		postID := "a"
		audio := &v1API.Audio{
			Id:    uuid.NewString(),
			Bytes: []byte("foo"),
		}
		audioRaw, err := json.Marshal(audio)
		assert.Nil(err)
		emptyAudio, err := json.Marshal(&v1API.Audio{})
		assert.Nil(err)
		authorUserID := uuid.NewString()
		cmtsToAdd := []comments.Comment{
			{
				ID:             "1",
				Message:        sql.NullString{String: "foo", Valid: true},
				Audio:          audioRaw,
				Author:         authorUserID,
				ParentID:       sql.NullString{String: "", Valid: false},
				Likes:          sql.NullInt32{Int32: 10, Valid: true},
				PostID:         "a",
				ThreadType:     "post",
				ThreadTargetID: sql.NullString{String: "a", Valid: true},
			},
			{
				ID:             "2",
				Message:        sql.NullString{String: "foo", Valid: true},
				Audio:          emptyAudio,
				Author:         authorUserID,
				ParentID:       sql.NullString{String: "1", Valid: true},
				Likes:          sql.NullInt32{Int32: 1, Valid: true},
				PostID:         "a",
				ThreadType:     "post",
				ThreadTargetID: sql.NullString{String: "a", Valid: true},
			},
			{
				ID:         "3",
				Message:    sql.NullString{String: "foo", Valid: true},
				Audio:      emptyAudio,
				Author:     authorUserID,
				ParentID:   sql.NullString{String: "1", Valid: true},
				Likes:      sql.NullInt32{Int32: 2, Valid: true},
				ThreadType: "none",
				PostID:     "a",
			},
			{
				ID:         "4",
				Message:    sql.NullString{String: "foo", Valid: true},
				Audio:      emptyAudio,
				Author:     authorUserID,
				ParentID:   sql.NullString{String: "3", Valid: true},
				Likes:      sql.NullInt32{Int32: 3, Valid: true},
				ThreadType: "none",
				PostID:     "a",
			},
			{
				ID:         "5",
				Message:    sql.NullString{String: "foo", Valid: true},
				Audio:      emptyAudio,
				Author:     authorUserID,
				ParentID:   sql.NullString{String: "4", Valid: true},
				Likes:      sql.NullInt32{Int32: 0, Valid: true},
				ThreadType: "none",
				PostID:     "a",
			},
			{
				ID:         "6",
				Message:    sql.NullString{String: "foo", Valid: true},
				Audio:      emptyAudio,
				Author:     authorUserID,
				ParentID:   sql.NullString{String: "", Valid: false},
				Likes:      sql.NullInt32{Int32: 9, Valid: true},
				ThreadType: "none",
				PostID:     "a",
			},
		}
		type orderedResult struct {
			id    string
			depth int32
			audio *v1API.Audio
		}
		expected := []orderedResult{
			{id: "1", audio: audio},
			{id: "2", audio: &v1API.Audio{}},
			{id: "6", audio: &v1API.Audio{}},
			{id: "4", audio: &v1API.Audio{}},
			{id: "3", audio: &v1API.Audio{}},
			{id: "5", audio: &v1API.Audio{}},
		}
		for _, c := range cmtsToAdd {
			q := `INSERT INTO COMMENTS (id, message, audio, author, parent_id, likes, post_id, thread_type)
				  values ($1, $2, $3, $4, $5, $6, $7, $8)`
			_, err := ws.Server.GetDB().Exec(q, c.ID, c.Message, c.Audio, c.Author, c.ParentID, c.Likes, c.PostID, c.ThreadType)
			if err != nil {
				t.Fatalf("error inserting comments: %v", err)
			}
		}

		res, err := dir.GetComments(ctx, authorUserID, postID)
		if err != nil {
			t.Errorf("error getting comments: %v", err)
		}

		assert.Equal(len(res), len(cmtsToAdd))
		for i, c := range res {
			assert.Equal(expected[i].id, c.Id, fmt.Sprintf("index: %d, expected: %s, got: %s", i, expected[i].id, c.Id))
			assert.Equal(expected[i].audio, c.Audio, fmt.Sprintf("index: %d, expected: %v, got: %v", i, expected[i].audio, c.Audio))
		}
	})
}

func TestLikeComment(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	dir := comments.NewDirectory(ws.Server.GetDB())
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	t.Cleanup(func() {
		err := dir.Close()
		if err != nil {
			t.Fatalf("error closing directory: %v", err)
		}
	})

	t.Run("When liking and removing like from a comment", func(t *testing.T) {
		t.Parallel()
		c := createTestComment(ctx, t, dir)
		userIDWhoLikes := uuid.NewString()

		// Like comment
		res, err := dir.LikeComment(ctx, comments.LikeCommentParams{
			ID:     c.Id,
			UserID: userIDWhoLikes,
		})
		assert.Nil(err)
		assert.True(res.HasAlreadyLiked)

		// Remove like comment
		res, err = dir.RemoveLikeFromComment(ctx, comments.RemoveLikeFromCommentParams{
			ID:     c.Id,
			UserID: userIDWhoLikes,
		})
		assert.Nil(err)
		assert.False(res.HasAlreadyLiked)
	})
}

func createTestComment(ctx context.Context, t *testing.T, dir *comments.Directory) *v1API.Comment {
	assert := assert.New(t)
	audio := &v1API.Audio{
		Id:    uuid.NewString(),
		Bytes: []byte("foo"),
	}
	audioRaw, err := json.Marshal(audio)
	assert.Nil(err)

	c, err := dir.CreateComment(ctx, comments.CreateCommentParams{
		ID:         uuid.NewString(),
		Message:    sql.NullString{String: "foo", Valid: true},
		Audio:      audioRaw,
		Author:     uuid.NewString(),
		PostID:     "foo",
		ThreadType: string(comments.ThreadTypeNone),
	})
	assert.Nil(err)

	return c
}
