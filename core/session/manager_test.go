package session

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	"github.com/DagDigg/unpaper/core/config"
// 	// "github.com/DagDigg/unpaper/extauth/pkg/unpapertest"
// 	"github.com/go-redis/redis/v8"
// 	"github.com/google/uuid"
// 	"github.com/stretchr/testify/assert"
// )

// func TestSetGetAndSync(t *testing.T) {
// 	t.Parallel()
// 	assert := assert.New(t)
// 	m := NewManager(getRDBInstance(t))
// 	ctx := context.Background()

// 	var (
// 		uniqueUserID     = uuid.NewString()
// 		duplicateUserID  = "dupe"
// 		userIDNotExpired = uuid.NewString()
// 	)

// 	sidsByUserID := map[string][]string{}
// 	expiry := 3 * time.Second

// 	// Create new sessions
// 	for _, tt := range []struct {
// 		user *User
// 	}{
// 		{&User{ID: uniqueUserID}},
// 		{&User{ID: duplicateUserID}},
// 		{&User{ID: duplicateUserID}},
// 	} {
// 		sid, err := m.SetNew(ctx, tt.user, expiry)
// 		assert.Nil(err)

// 		sidsByUserID[tt.user.ID] = append(sidsByUserID[tt.user.ID], sid)
// 	}

// 	// Retrieve set from sid
// 	for userID, sids := range sidsByUserID {
// 		res, err := m.rdb.SMembers(ctx, userID).Result()
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		assert.ElementsMatch(res, sids)
// 	}

// 	// Create a session that should not expire immediately
// 	notExpiredSid, err := m.SetNew(ctx, &User{ID: userIDNotExpired}, 7*time.Second)
// 	assert.Nil(err)

// 	// Wait until keys are expired, sync and check new values
// 	time.Sleep(expiry)

// 	// The session with longer expiry should continue to live
// 	err = m.Sync(ctx, userIDNotExpired)
// 	assert.Nil(err)
// 	sid, err := m.rdb.Get(ctx, notExpiredSid).Result()
// 	assert.Nil(err)
// 	assert.NotEqual(sid, "")

// 	// The other sessions had shorted expiry, they should no longer exist
// 	for userID, sids := range sidsByUserID {
// 		// Sync user sessions
// 		err := m.Sync(ctx, userID)
// 		assert.Nil(err)

// 		// Check each user session to be empty
// 		for _, sid := range sids {
// 			res, err := m.rdb.Get(ctx, sid).Result()
// 			assert.Equal(err, redis.Nil)
// 			assert.Equal(res, "")
// 		}

// 	}

// }

// func getRDBInstance(t *testing.T) *redis.Client {
// 	u := unpapertest.StartRedisDB(t, config.Get().GetRDBConnURL())

// 	opt, err := redis.ParseURL(u.String())
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	return redis.NewClient(opt)
// }
