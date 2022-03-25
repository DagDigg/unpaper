package v1

import (
	"context"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/mdutils"
	"github.com/DagDigg/unpaper/backend/users"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *unpaperServiceServer) UserInfo(ctx context.Context, req *v1API.UserInfoRequest) (*v1API.User, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "Missing userID")
	}
	usersDirectory := users.NewDirectory(s.db)

	user, err := usersDirectory.GetUser(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to get userInfo on db: %v", err)
	}

	return user, nil
}
