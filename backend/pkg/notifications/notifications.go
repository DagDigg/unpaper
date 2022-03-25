package notifications

import (
	"context"
	"database/sql"
	"time"

	dbNotifications "github.com/DagDigg/unpaper/backend/notifications"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/usersession"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// Manager structure has methods for interacting with notifications
type Manager struct {
	dir         *dbNotifications.Directory
	usersession usersession.Sessioner
	pushmanager SubscribePusher
}

// NewManager returns a new notifications.Manager with an underlying database directory
func NewManager(db *sql.DB, rdb *redis.Client) SendListenReceiver {
	dir := dbNotifications.NewDirectory(db)
	pushmanager := NewPush(rdb)
	usrsession := usersession.NewManager(rdb)
	return &Manager{
		dir:         dir,
		pushmanager: pushmanager,
		usersession: usrsession,
	}
}

// SendListenReceiver interface contains method for interacting with notifications
type SendListenReceiver interface {
	Send(SendNotificationParams) (*v1API.Notification, error)
	Listen(ctx context.Context, userID string) <-chan *v1API.Notification
}

// ResendCondition is used when a notification already exists,
// it denotes the condition to which should be resent
type ResendCondition func(n *v1API.Notification) bool

// ResendConditionNever never resends the notification if already exists
func ResendConditionNever(n *v1API.Notification) bool {
	return false
}

// ResendConditionAfter resends the notification if the elapsed time since the previous is greater than `d`
func ResendConditionAfter(d time.Duration) ResendCondition {
	return func(n *v1API.Notification) bool {
		return time.Now().Sub(n.Date.AsTime()) > d
	}
}

// SendNotificationParams parameters for sending a notification
type SendNotificationParams struct {
	Ctx             context.Context
	ResendCondition ResendCondition
	SenderUserID    string
	ReceiverUserID  string
	TriggerID       string
	EventID         string
	Content         string
}

// Send creates a notification in db and sends it to the user
func (m *Manager) Send(p SendNotificationParams) (*v1API.Notification, error) {
	ok, err := m.shouldSendNotification(p)
	if err != nil {
		return nil, err
	}
	if !ok {
		// Should not send notification
		return nil, nil
	}

	// Store notification on db
	notification, err := m.createNotification(p)

	isOnline, err := m.usersession.IsOnline(p.Ctx, p.ReceiverUserID)
	if err != nil {
		return nil, err
	}

	if isOnline {
		// User is online. Send 'push' notification
		if err := m.pushmanager.Push(p.Ctx, p.ReceiverUserID, notification); err != nil {
			return nil, err
		}

		return notification, nil
	}

	// User is NOT online. Send an email
	// TODO...
	return notification, nil
}

func (m *Manager) shouldSendNotification(p SendNotificationParams) (bool, error) {
	prevNotification, err := m.dir.GetNotification(p.Ctx, dbNotifications.GetNotificationParams{
		UserIDToNotify:      p.ReceiverUserID,
		UserIDWhoFiredEvent: p.SenderUserID,
		TriggerID:           sql.NullString{String: p.TriggerID, Valid: p.TriggerID != ""},
		EventID:             string(dbNotifications.EventIDLikePost),
	})
	if err != nil {
		// Database unexpected error
		if err != sql.ErrNoRows {
			return false, err
		}
	}
	if p.ReceiverUserID == p.SenderUserID {
		// Do nothing and return if user performed action on its behalf
		return false, nil
	}
	if prevNotification != nil {
		if p.ResendCondition == nil {
			return true, nil
		}

		return p.ResendCondition(prevNotification), nil
	}

	return true, nil
}

func (m *Manager) createNotification(p SendNotificationParams) (*v1API.Notification, error) {
	params := dbNotifications.CreateNotificationParams{
		ID:                  uuid.NewString(),
		Date:                time.Now(),
		UserIDToNotify:      p.ReceiverUserID,
		UserIDWhoFiredEvent: p.SenderUserID,
		EventID:             p.EventID,
	}
	if p.Content != "" {
		params.Content = sql.NullString{String: truncateString(p.Content, 64), Valid: true}
	}
	if p.TriggerID != "" {
		params.TriggerID = sql.NullString{String: p.TriggerID, Valid: true}
	}

	return m.dir.CreateNotification(p.Ctx, params)
}

// Listen subscribes to the userID notifications channel
func (m *Manager) Listen(ctx context.Context, userID string) <-chan *v1API.Notification {
	return m.pushmanager.Subscribe(ctx, userID)
}

func truncateString(str string, num int) string {
	bnoden := str
	if len(str) > num {
		if num > 3 {
			num -= 3
		}
		bnoden = str[0:num] + "..."
	}
	return bnoden
}
