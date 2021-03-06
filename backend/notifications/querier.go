// Code generated by sqlc. DO NOT EDIT.

package notifications

import (
	"context"
)

type Querier interface {
	CreateNotification(ctx context.Context, arg CreateNotificationParams) (CreateNotificationRow, error)
	GetAllNotifications(ctx context.Context, userIDToNotify string) ([]GetAllNotificationsRow, error)
	GetNotification(ctx context.Context, arg GetNotificationParams) (GetNotificationRow, error)
	GetNotificationByID(ctx context.Context, id string) (GetNotificationByIDRow, error)
	GetUnreadNotifications(ctx context.Context, userIDToNotify string) ([]GetUnreadNotificationsRow, error)
	NotificationAlreadyExists(ctx context.Context, arg NotificationAlreadyExistsParams) (bool, error)
	ReadNotification(ctx context.Context, id string) (ReadNotificationRow, error)
}

var _ Querier = (*Queries)(nil)
