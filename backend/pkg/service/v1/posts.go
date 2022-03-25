package v1

import (
	"context"
	"database/sql"
	"time"

	"github.com/DagDigg/unpaper/backend/comments"
	dbNotifications "github.com/DagDigg/unpaper/backend/notifications"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/dbentities"
	"github.com/DagDigg/unpaper/backend/pkg/logger"
	"github.com/DagDigg/unpaper/backend/pkg/mdutils"
	"github.com/DagDigg/unpaper/backend/pkg/notifications"
	"github.com/DagDigg/unpaper/backend/posts"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreatePost RPC for inserting a post in the datastore
func (s *unpaperServiceServer) CreatePost(ctx context.Context, req *v1API.CreatePostRequest) (*v1API.CreatePostResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	postsDir := posts.NewDirectory(s.db)
	rawAudio, err := dbentities.NewAudioRawJSON(&v1API.Audio{
		Id:         uuid.NewString(),
		Bytes:      req.AudioBytes,
		Format:     req.AudioFormat,
		DurationMs: req.AudioDurationMs,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not convert request audio to raw json: %v", err)
	}

	p, err := postsDir.CreatePost(ctx, posts.CreatePostParams{
		ID:        uuid.NewString(),
		Author:    userID,
		Message:   req.Message,
		Audio:     rawAudio,
		CreatedAt: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error inserting post on db: %v", err)
	}

	return &v1API.CreatePostResponse{
		Post: p,
	}, nil
}

// GetPosts RPC retrieves posts
func (s *unpaperServiceServer) GetPosts(ctx context.Context, req *v1API.GetPostsRequest) (*v1API.GetPostsResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	postsDir := posts.NewDirectory(s.db)
	commentsDir := comments.NewDirectory(s.db)

	postsList, err := postsDir.GetPosts(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve posts: %v", err)
	}

	// Add comments to posts
	for i, p := range postsList {
		comments, err := commentsDir.GetComments(ctx, userID, p.Id)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to retrieve comments for post id %q: %v", p.Id, err)
		}
		hasUserLiked, err := postsDir.HasUserLikedPost(ctx, posts.HasUserLikedPostParams{
			ID:      p.Id,
			Column2: userID,
		})
		postsList[i].HasAlreadyLiked = hasUserLiked
		postsList[i].Comments = comments
	}

	return &v1API.GetPostsResponse{
		Posts: postsList,
	}, nil
}

// GetPosts RPC retrieves a single posts
func (s *unpaperServiceServer) GetPost(ctx context.Context, req *v1API.GetPostRequest) (*v1API.GetPostResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	postsDir := posts.NewDirectory(s.db)
	commentsDir := comments.NewDirectory(s.db)

	post, err := fetchPost(fetchPostParams{
		ctx:         ctx,
		postsDir:    postsDir,
		commentsDir: commentsDir,
		postID:      req.PostId,
		userID:      userID,
	})
	if err != nil {
		return nil, err
	}

	return &v1API.GetPostResponse{
		Post: post,
	}, nil
}

// CreateComment inserts a comment for a post in the database
func (s *unpaperServiceServer) CreateComment(ctx context.Context, req *v1API.CreateCommentRequest) (*v1API.CreateCommentResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	postsDir := posts.NewDirectory(s.db)
	post, err := postsDir.GetPost(ctx, req.PostId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve post for comment: %v", err)
	}

	if req.Thread == nil && req.Message == "" { // TODO: assert minimum length
		return nil, status.Error(codes.Internal, "non threaded post must have a message")
	}

	// If it's a threaded post, check the ownership. Only post authors can thread their posts
	if req.Thread != nil {
		if req.Thread.ThreadType == v1API.ThreadType_POST && userID != post.Author {
			return nil, status.Error(codes.Internal, "only post author can comment on a thread of type 'post'")
		}
	}
	commentsDir := comments.NewDirectory(s.db)
	rawAudio, err := dbentities.NewAudioRawJSON(&v1API.Audio{
		Id:         uuid.NewString(),
		Bytes:      req.AudioBytes,
		DurationMs: req.AudioDurationMs,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not convert request audio to raw json: %v", err)
	}
	params := comments.CreateCommentParams{
		ID:         uuid.NewString(),
		Message:    sql.NullString{String: req.Message, Valid: req.Message != ""},
		Audio:      rawAudio,
		Author:     userID,
		ParentID:   sql.NullString{String: req.ParentId, Valid: req.ParentId != ""},
		PostID:     req.PostId,
		ThreadType: string(comments.ThreadTypeNone), // Default to 'none'
	}
	if req.Thread != nil {
		params.ThreadTargetID = sql.NullString{String: req.Thread.TargetId, Valid: req.Thread.TargetId != ""}
		threadType, err := pbThreadTypeToPG(req.Thread.ThreadType)
		if err != nil {
			return nil, err
		}
		params.ThreadType = string(threadType)
	}
	c, err := commentsDir.CreateComment(ctx, params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create comment: %v", err)
	}

	_, err = s.nm.Send(notifications.SendNotificationParams{
		Ctx:            ctx,
		SenderUserID:   userID,
		ReceiverUserID: post.Author,
		TriggerID:      c.Id,
		EventID:        string(dbNotifications.EventIDComment),
		Content:        c.Message,
	})
	if err != nil {
		// Do not throw error on notification send failure
		logger.Log.Error(err.Error())
	}

	return &v1API.CreateCommentResponse{
		Comment: c,
	}, nil
}

func pbThreadTypeToPG(t v1API.ThreadType_Enum) (comments.ThreadType, error) {
	switch t {
	case v1API.ThreadType_COMMENT:
		return comments.ThreadTypeComment, nil
	case v1API.ThreadType_POST:
		return comments.ThreadTypePost, nil
	case v1API.ThreadType_NONE:
		return comments.ThreadTypeNone, nil
	default:
		return "", status.Error(codes.Internal, "invalid thread_type received: %v")
	}
}

// LikeComment increments the comment's likes and returns the new updated comment
func (s *unpaperServiceServer) LikeComment(ctx context.Context, req *v1API.LikeCommentRequest) (*v1API.LikeCommentResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	commentsDir := comments.NewDirectory(s.db)

	comment, err := commentsDir.LikeComment(ctx, comments.LikeCommentParams{
		ID:     req.CommentId,
		UserID: userID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not like comment: %v", err)
	}

	_, err = s.nm.Send(notifications.SendNotificationParams{
		Ctx:             ctx,
		SenderUserID:    userID,
		ReceiverUserID:  comment.Author,
		TriggerID:       comment.Id,
		EventID:         string(dbNotifications.EventIDLikeComment),
		ResendCondition: notifications.ResendConditionAfter(5 * time.Minute),
		Content:         comment.Message,
	})
	if err != nil {
		// Do not throw error on notification send failure
		logger.Log.Error(err.Error())
	}

	return &v1API.LikeCommentResponse{
		Comment: comment,
	}, nil
}

// LikePost increments the post's likes and returns the new updated post
func (s *unpaperServiceServer) LikePost(ctx context.Context, req *v1API.LikePostRequest) (*v1API.LikePostResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	postsDir := posts.NewDirectory(s.db)
	commentsDir := comments.NewDirectory(s.db)

	hasAlreadyLiked, err := postsDir.HasUserLikedPost(ctx, posts.HasUserLikedPostParams{
		ID:      req.PostId,
		Column2: userID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve 'user has already liked post': %v", err)
	}

	_, err = likeOrDislikePost(likeDislikePostParams{
		ctx:             ctx,
		db:              s.db,
		postID:          req.PostId,
		userID:          userID,
		hasAlreadyLiked: hasAlreadyLiked,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not toggle like post: %v", err)
	}

	post, err := fetchPost(fetchPostParams{
		ctx:         ctx,
		postsDir:    postsDir,
		commentsDir: commentsDir,
		postID:      req.PostId,
		userID:      userID,
	})

	_, err = s.nm.Send(notifications.SendNotificationParams{
		Ctx:             ctx,
		SenderUserID:    userID,
		ReceiverUserID:  post.Author,
		TriggerID:       post.Id,
		EventID:         string(dbNotifications.EventIDLikePost),
		ResendCondition: notifications.ResendConditionAfter(5 * time.Minute),
		Content:         post.Message,
	})
	if err != nil {
		// Do not throw error on notification send failure
		logger.Log.Error(err.Error())
	}

	return &v1API.LikePostResponse{
		Post: post,
	}, nil
}

type likeDislikePostParams struct {
	ctx             context.Context
	db              *sql.DB
	postID          string
	userID          string
	hasAlreadyLiked bool
}

func likeOrDislikePost(params likeDislikePostParams) (*v1API.Post, error) {
	postsDir := posts.NewDirectory(params.db)
	if params.hasAlreadyLiked {
		return postsDir.RemoveLikeFromPost(params.ctx, posts.RemoveLikeFromPostParams{
			ID:      params.postID,
			Column1: params.userID,
		})
	}

	return postsDir.LikePost(params.ctx, posts.LikePostParams{
		ID:      params.postID,
		Column1: params.userID,
	})
}

type fetchPostParams struct {
	ctx         context.Context
	postsDir    *posts.Directory
	commentsDir *comments.Directory
	postID      string
	userID      string
}

func fetchPost(params fetchPostParams) (*v1API.Post, error) {
	post, err := params.postsDir.GetPost(params.ctx, params.postID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve post: %v", err)
	}

	// Add comments to post
	comments, err := params.commentsDir.GetComments(params.ctx, params.userID, post.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve comments for post id %q: %v", post.Id, err)
	}
	post.Comments = comments
	hasUserLiked, err := params.postsDir.HasUserLikedPost(params.ctx, posts.HasUserLikedPostParams{
		ID:      post.Id,
		Column2: params.userID,
	})
	post.HasAlreadyLiked = hasUserLiked

	return post, nil
}
