package notifications_test

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	v1Helpers "github.com/DagDigg/unpaper/backend/helpers"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/notifications"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSubscribe(t *testing.T) {
	cfg := v1Testing.InitConfig()
	push := initPushManager(t, cfg.GetRDBConnURL())
	assert := assert.New(t)
	t.Parallel()

	t.Run("When subscribing to notifications channel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		userIDToSubscribe := uuid.NewString()
		wg := &sync.WaitGroup{}

		notificationsToSend := []*v1API.Notification{
			{
				Id:        "1",
				Date:      timestamppb.New(time.Now()),
				TriggerId: "trigger",
				Event: &v1API.Event{
					Id:   v1API.EventID_COMMENT,
					Text: "foo",
				},
			},
			{
				Id:        "2",
				Date:      timestamppb.New(time.Now()),
				TriggerId: "trigger2",
				Event: &v1API.Event{
					Id:   v1API.EventID_FOLLOW,
					Text: "bar",
				},
			},
		}

		wg.Add(len(notificationsToSend))
		notificationsCh := push.Subscribe(ctx, userIDToSubscribe)
		go func() {
			count := 0

			for n := range notificationsCh {
				fmt.Printf("received notification: %v\n", n.Id)
				assert.Equal(notificationsToSend[count], n)
				count++
				wg.Done()
			}
		}()

		for _, n := range notificationsToSend {
			fmt.Printf("sending notification: %v\n", n.Id)
			err := push.Push(ctx, userIDToSubscribe, n)
			assert.Nil(err)
		}

		wg.Wait()
		cancel() // Cancel context to gracefully close redis conn
	})
}

func initPushManager(t *testing.T, rdbConnURL *url.URL) notifications.SubscribePusher {
	rdbURL := v1Helpers.StartRedisDB(t, rdbConnURL)
	rdb := v1Helpers.GetRDBInstance(t, rdbURL)
	return notifications.NewPush(rdb)
}
