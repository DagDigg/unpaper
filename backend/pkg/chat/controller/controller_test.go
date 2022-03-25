package controller_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	v1Helpers "github.com/DagDigg/unpaper/backend/helpers"
	"github.com/DagDigg/unpaper/backend/pkg/chat/controller"
	"github.com/DagDigg/unpaper/backend/pkg/chat/message"
	"github.com/DagDigg/unpaper/backend/pkg/chat/usecase"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSubscribe(t *testing.T) {
	cfg := v1Testing.InitConfig()
	rdbURL := v1Helpers.StartRedisDB(t, cfg.GetRDBConnURL())
	u := usecase.New(v1Helpers.GetRDBInstance(t, rdbURL))
	c := controller.New(u)
	assert := assert.New(t)

	t.Run("When subscribing and sending a single message", func(t *testing.T) {
		ctx := context.Background()
		ch := "test_channel"
		userID := uuid.NewString()
		msgToSend := &message.Message{
			ID:        "message_id_1",
			UserID:    userID,
			CreatedAt: time.Date(1971, time.April, 31, 00, 00, 00, 00, time.Local),
			Text:      message.Text{Content: "hello!"},
		}

		msgsRes, err := c.GetMessages(ctx, userID, ch, 0)
		assert.Nil(err)
		err = c.SendMessage(ctx, ch, msgToSend)
		assert.Nil(err)

		done := make(chan struct{})
		time.AfterFunc(1*time.Second, func() {
			close(done)
		})

		go func() {
			count := 0
			for msg := range msgsRes.Messages {
				count++
				if count > 1 {
					t.Errorf("expected 1 message, got: %d", count)
				}
				if !reflect.DeepEqual(msg, msgToSend) {
					t.Errorf("messages mismatch.\ngot: %+v\nwant: %+v", msg, msgToSend)
				}
			}
		}()

		<-done
	})
}
