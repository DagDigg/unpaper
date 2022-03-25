package v1

import (
	"context"

	"github.com/DagDigg/unpaper/backend/notifications"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/mdutils"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListenForNotifications starts a GRPC stream which sends notification as they are read from the rdb
func (s *unpaperServiceServer) ListenForNotifications(req *empty.Empty, stream v1API.UnpaperService_ListenForNotificationsServer) error {
	ctx := stream.Context()
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	var sendErr error

	done := make(chan struct{})
	go func() {
		defer close(done)

		notifications := s.nm.Listen(ctx, userID)
		for {
			select {
			case <-ctx.Done():
				return
			case out, ok := <-notifications:
				if !ok {
					return
				}
				if err := stream.Send(out); err != nil {
					sendErr = err
					return
				}
			}
		}
	}()

	<-done
	return sendErr
}

// GetAllNotifications returns all notifications for the user
func (s *unpaperServiceServer) GetAllNotifications(ctx context.Context, req *empty.Empty) (*v1API.GetAllNotificationsRes, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	notificationsDir := notifications.NewDirectory(s.db)

	res, err := notificationsDir.GetAllNotifications(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve notifications: %v", err)
	}

	return &v1API.GetAllNotificationsRes{
		Notifications: res,
	}, nil
}

// ReadNotification sets to true the `read` column on db and returns the updated notification
func (s *unpaperServiceServer) ReadNotification(ctx context.Context, req *v1API.ReadNotificationRequest) (*v1API.ReadNotificationResponse, error) {
	notificationsDir := notifications.NewDirectory(s.db)
	n, err := notificationsDir.ReadNotification(ctx, req.NotificationId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read notification: %v", err)
	}

	return &v1API.ReadNotificationResponse{
		Notification: n,
	}, nil
}
