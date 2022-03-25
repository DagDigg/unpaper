package notifications

import (
	"context"
	"database/sql"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/Masterminds/squirrel"
)

// Directory is the directory which operates on db table 'notifications'
type Directory struct {
	// querier is an interface containing all of the
	// directory methods. Must be created with customers.NewDirectory(db)
	querier Querier
	db      *sql.DB
	sb      squirrel.StatementBuilderType
}

// NewDirectory creates a new users directory
func NewDirectory(db *sql.DB) *Directory {
	return &Directory{db: db, querier: New(db)}
}

// Close closes Directory database connection
func (d Directory) Close() error {
	return d.db.Close()
}

// EventText type for notification displayed text
type EventText string

// EventID unique identifier for a specific notification event
type EventID string

const (
	// EventIDLikePost 'like' post event
	EventIDLikePost EventID = "LIKE_POST"
	// EventIDLikeComment 'like' comment event
	EventIDLikeComment EventID = "LIKE_COMMENT"
	// EventIDComment 'comment' event
	EventIDComment EventID = "COMMENT"
	// EventIDFollow 'follow' event
	EventIDFollow EventID = "FOLLOW"
)

const (
	// EventTextLikePost used on a `like` post event
	EventTextLikePost EventText = "liked your post!"
	// EventTextLikeComment used on a `like` comment event
	EventTextLikeComment EventText = "liked your comment!"
	// EventTextComment used on a `comment` event
	EventTextComment EventText = "commented your post"
	// EventTextFollow used on a `follow` event
	EventTextFollow EventText = "started following you!"
)

// CreateNotification insert a new notification into db
func (d Directory) CreateNotification(ctx context.Context, params CreateNotificationParams) (*v1API.Notification, error) {
	res, err := d.querier.CreateNotification(ctx, params)
	if err != nil {
		return nil, err
	}

	return pgNotificationToPB(res)
}

// GetUnreadNotifications returns a list of notifications that haven't been read by the user
func (d Directory) GetUnreadNotifications(ctx context.Context, userID string) ([]*v1API.Notification, error) {
	res, err := d.querier.GetUnreadNotifications(ctx, userID)
	if err != nil {
		return nil, err
	}

	return pgUnreadNotificationsListToPB(res)
}

// ReadNotification sets to 'true' the 'read' column for the notificationID
func (d Directory) ReadNotification(ctx context.Context, notificationID string) (*v1API.Notification, error) {
	res, err := d.querier.ReadNotification(ctx, notificationID)
	if err != nil {
		return nil, err
	}

	return pgNotificationToPB(CreateNotificationRow(res))
}

// NotificationAlreadyExists returns if a notification has already been created.
// It does not check by ID, instead, it is calculated by the fact if there is a notification with the same sender, receiver, targetID and eventID.
// If so, it is considered that it exists
func (d Directory) NotificationAlreadyExists(ctx context.Context, params NotificationAlreadyExistsParams) (bool, error) {
	return d.querier.NotificationAlreadyExists(ctx, params)
}

// GetNotification retrieves a notification querying by the sender id, receiver id, trigger id and event id
func (d Directory) GetNotification(ctx context.Context, params GetNotificationParams) (*v1API.Notification, error) {
	res, err := d.querier.GetNotification(ctx, params)
	if err != nil {
		return nil, err
	}

	return pgNotificationToPB(CreateNotificationRow(res))
}

// GetAllNotifications returns a list of all notifications ordered by date descending. The unread ones are always returned at the top.
func (d Directory) GetAllNotifications(ctx context.Context, userID string) ([]*v1API.Notification, error) {
	res, err := d.querier.GetAllNotifications(ctx, userID)
	if err != nil {
		return nil, err
	}

	return pgGetAllNotificationsListToPB(res)
}
