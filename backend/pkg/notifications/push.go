package notifications

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/go-redis/redis/v8"
)

type PushManager struct {
	rdb *redis.Client
}

func NewPush(rdb *redis.Client) SubscribePusher {
	return &PushManager{
		rdb: rdb,
	}
}

type SubscribePusher interface {
	Subscribe(ctx context.Context, userID string) <-chan *v1API.Notification
	Push(ctx context.Context, userID string, notification *v1API.Notification) error
}

const (
	channelPushSubscription = "push:user:"
)

func (p *PushManager) Subscribe(ctx context.Context, userID string) <-chan *v1API.Notification {
	notifications := make(chan *v1API.Notification)
	pubsub := p.rdb.Subscribe(ctx, getPushChannel(userID))
	// Wait for confirmation that subscription is created before publishing anything.
	_, err := pubsub.Receive(ctx)
	if err != nil {
		close(notifications)
	}

	go func() {
		defer close(notifications)
		defer pubsub.Close()

		go func() {
			<-ctx.Done()
			pubsub.Close()
		}()

		for m := range pubsub.Channel() {
			n, err := DecodeBinaryNotification(m.Payload)
			if err != nil {
				return
			}
			notifications <- n
		}
	}()

	return notifications
}

func (p *PushManager) Push(ctx context.Context, userID string, n *v1API.Notification) error {
	notification, err := EncodeBinaryNotification(n)
	if err != nil {
		return err
	}

	return p.rdb.Publish(ctx, getPushChannel(userID), notification).Err()
}

func EncodeBinaryNotification(n *v1API.Notification) (string, error) {
	b := &bytes.Buffer{}
	e := gob.NewEncoder(b)
	if err := e.Encode(n); err != nil {
		return "", nil
	}

	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

func DecodeBinaryNotification(n string) (*v1API.Notification, error) {
	p, err := base64.StdEncoding.DecodeString(n)
	if err != nil {
		return nil, err
	}
	b := &bytes.Buffer{}

	if _, err = b.Write(p); err != nil {
		return nil, err
	}

	d := gob.NewDecoder(b)
	pbNotification := &v1API.Notification{}
	if err := d.Decode(pbNotification); err != nil {
		return nil, err
	}

	return pbNotification, nil
}

func getPushChannel(userID string) string {
	return channelPushSubscription + userID
}
