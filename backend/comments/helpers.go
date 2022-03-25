package comments

import (
	"github.com/DagDigg/unpaper/backend/helpers"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/dbentities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// pgCommentToPB converts a postgres comment to protobuf
type pgCommentToPBParams struct {
	c               Comment
	hasAlreadyLiked bool
}

func pgCommentToPB(params pgCommentToPBParams) (*v1API.Comment, error) {
	audio, err := dbentities.AudioRawJSONToPB(params.c.Audio)
	if err != nil {
		return nil, err
	}
	threadType, err := pgThreadTypeToPB(ThreadType(params.c.ThreadType))
	if err != nil {
		return nil, err
	}
	comment := &v1API.Comment{
		Id:       params.c.ID,
		Message:  params.c.Message.String,
		Audio:    audio,
		Author:   params.c.Author,
		ParentId: params.c.ParentID.String,
		Likes:    params.c.Likes.Int32,
		PostId:   params.c.PostID,
		Thread: &v1API.Thread{
			ThreadType: threadType,
			TargetId:   params.c.ThreadTargetID.String,
		},
		HasAlreadyLiked: params.hasAlreadyLiked,
	}

	return comment, nil
}

func hasAlreadyLiked(userIDsWhoLikes []string, userID string) bool {
	return helpers.StringSliceContains(userIDsWhoLikes, userID)
}

func pgThreadTypeToPB(t ThreadType) (v1API.ThreadType_Enum, error) {
	switch t {
	case ThreadTypeComment:
		return v1API.ThreadType_COMMENT, nil
	case ThreadTypePost:
		return v1API.ThreadType_POST, nil
	case ThreadTypeNone:
		return v1API.ThreadType_NONE, nil
	default:
		return 0, status.Errorf(codes.Internal, "failed to convert pg thread_type to pb: %v", t)
	}
}
