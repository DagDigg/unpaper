package follows_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DagDigg/unpaper/backend/follows"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/stretchr/testify/assert"
)

func TestFollowUser(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	dir := follows.NewDirectory(ws.Server.GetDB())
	assert := assert.New(t)

	t.Run("When following and unfollowing user", func(t *testing.T) {
		usrToBeFollowed, err := ws.AddUser(v1Testing.GetRandomPGUserParams())
		assert.Nil(err)
		usrWhoFollows, err := ws.AddUser(v1Testing.GetRandomPGUserParams())
		assert.Nil(err)

		// Follow user
		ctx := context.Background()
		_, err = dir.FollowUser(ctx, follows.FollowUserParams{
			FollowerUserID:  usrWhoFollows.Id,
			FollowingUserID: usrToBeFollowed.Id,
			FollowDate:      time.Now(),
		})
		assert.Nil(err)

		// Assert follow user
		ok, err := dir.IsFollowingUser(ctx, follows.IsFollowingUserParams{
			FollowerUserID:  usrWhoFollows.Id,
			FollowingUserID: usrToBeFollowed.Id,
		})
		assert.Nil(err)
		assert.True(ok)

		// Unfollow user
		_, err = dir.UnfollowUser(ctx, follows.UnfollowUserParams{
			FollowerUserID:  usrWhoFollows.Id,
			FollowingUserID: usrToBeFollowed.Id,
			UnfollowDate:    sql.NullTime{Time: time.Now(), Valid: true},
		})
		assert.Nil(err)

		// Assert unfollow user
		ok, err = dir.IsFollowingUser(ctx, follows.IsFollowingUserParams{
			FollowerUserID:  usrWhoFollows.Id,
			FollowingUserID: usrToBeFollowed.Id,
		})
		assert.Nil(err)
		assert.False(ok)

		// Re-follow user
		_, err = dir.FollowUser(ctx, follows.FollowUserParams{
			FollowerUserID:  usrWhoFollows.Id,
			FollowingUserID: usrToBeFollowed.Id,
			FollowDate:      time.Now(),
		})
		assert.Nil(err)
		// Assert follow user
		ok, err = dir.IsFollowingUser(ctx, follows.IsFollowingUserParams{
			FollowerUserID:  usrWhoFollows.Id,
			FollowingUserID: usrToBeFollowed.Id,
		})
		assert.Nil(err)
		assert.True(ok)
	})
}
