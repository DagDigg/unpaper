package usecase_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	v1Helpers "github.com/DagDigg/unpaper/backend/helpers"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/chat"
	"github.com/DagDigg/unpaper/backend/pkg/chat/message"
	chatUsecase "github.com/DagDigg/unpaper/backend/pkg/chat/usecase"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSubscribe(t *testing.T) {
	cfg := v1Testing.InitConfig()
	c := initChatUseCase(t, cfg.GetRDBConnURL())
	assert := assert.New(t)
	t.Parallel()

	t.Run("When subscribing, and sending a message to a channel", func(t *testing.T) {
		ctx := context.Background()
		ch := uuid.NewString()
		messages := c.Subscribe(ctx, ch)
		msgsToSend := []*message.Message{
			{
				ID:        "message_ID_1",
				UserID:    "bob",
				CreatedAt: time.Date(1971, time.April, 31, 00, 00, 00, 00, time.Local),
				Text:      message.Text{Content: "Hello"},
			},
			{
				ID:        "message_ID_2",
				UserID:    "bob",
				CreatedAt: time.Date(1971, time.April, 31, 00, 00, 00, 00, time.Local),
				Award:     message.Award{AwardID: "award_id_1"},
			},
		}

		done := make(chan struct{})
		time.AfterFunc(time.Second, func() {
			close(done)
		})

		for _, m := range msgsToSend {
			err := c.SendMessage(ctx, ch, m)
			if err != nil {
				t.Errorf("error sending msg with id %q to redis channel: %v", m.ID, err)
			}
		}

		go func() {
			i := 0
			for msg := range messages {
				assert.Equal(msg, msgsToSend[i])
				i++
			}
		}()
		<-done
	})
}

func TestGetMessages(t *testing.T) {
	t.Parallel()
	cfg := v1Testing.InitConfig()
	c := initChatUseCase(t, cfg.GetRDBConnURL())
	assert := assert.New(t)

	msgsToAdd := map[string]*message.Message{
		"msg_one": {
			ID:        "msg_one",
			Type:      message.TypeText,
			UserID:    "user_one",
			CreatedAt: time.Date(1971, time.April, 31, 00, 00, 00, 00, time.Local),
			Text:      message.Text{Content: "content one"},
		},
		"msg_two": {
			ID:        "msg_two",
			Type:      message.TypeText,
			UserID:    "user_two",
			CreatedAt: time.Date(1971, time.April, 31, 00, 00, 00, 00, time.Local),
			Text:      message.Text{Content: "content two"},
		},
		"msg_three": {
			ID:        "msg_three",
			Type:      message.TypeAward,
			UserID:    "user_three",
			CreatedAt: time.Date(1971, time.April, 31, 00, 00, 00, 00, time.Local),
			Award:     message.Award{AwardID: "zz_yy_xx"},
		},
		"msg_four": {
			ID:        "msg_four",
			Type:      message.TypeDonation,
			UserID:    "user_four",
			CreatedAt: time.Date(1971, time.April, 31, 00, 00, 00, 00, time.Local),
			Donation:  message.Donation{Amount: 10},
		},
	}

	t.Run("When sending and retrieving all messages", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		chanName := uuid.New().String()
		userID := uuid.NewString()
		for _, msg := range msgsToAdd {
			if err := c.SendMessage(ctx, chanName, msg); err != nil {
				t.Errorf("error sending message: %v", err)
			}
		}

		msgsCh, err := c.GetMessages(ctx, userID, chanName, 0)
		assert.Nil(err)
		for _, m := range msgsCh {
			msgFound := msgsToAdd[m.Id]
			protoMsgFound := msgFound.ToProtobuf()
			assert.NotNil(msgFound)
			assert.NotNil(m)

			if protoMsgFound.CreatedAt != m.CreatedAt {
				t.Errorf("message 'created at' mismatch. got: %v, want: %v", protoMsgFound.CreatedAt, m.CreatedAt)
			}

			if m.Type == v1API.MessageType_TEXT {
				assert.Equal(protoMsgFound.Text.Content, m.Text.Content)
			}

			if m.Type == v1API.MessageType_AWARD {
				assert.Equal(protoMsgFound.Award.AwardId, m.Award.AwardId)
			}

			if m.Type == v1API.MessageType_DONATION {
				assert.Equal(protoMsgFound.Donation.Amount, m.Donation.Amount)
			}
		}
	})
}

func createUUIDSlice(size int) []string {
	res := []string{}
	for i := 0; i < size; i++ {
		res = append(res, uuid.NewString())
	}
	return res
}

func initChatUseCase(t *testing.T, rdbConnURL *url.URL) chat.Usecase {
	rdbURL := v1Helpers.StartRedisDB(t, rdbConnURL)
	rdb := v1Helpers.GetRDBInstance(t, rdbURL)
	return chatUsecase.New(rdb)
}
