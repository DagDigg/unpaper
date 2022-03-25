package v1

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/DagDigg/unpaper/backend/customers"
	"github.com/DagDigg/unpaper/backend/helpers"
	"github.com/DagDigg/unpaper/backend/lists"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/chat/conversation"
	"github.com/DagDigg/unpaper/backend/pkg/chat/message"
	"github.com/DagDigg/unpaper/backend/pkg/dbentities"
	"github.com/DagDigg/unpaper/backend/pkg/mdutils"
	"github.com/DagDigg/unpaper/backend/users"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrConversationNotFound error returned when no conversation has been found
var ErrConversationNotFound = status.Error(codes.Internal, "no conversation found")

// ListenForMessages subscribes to a redis pubsub by channel param, listening to incoming messages
func (s *unpaperServiceServer) ListenForMessages(req *v1API.ListenForMessagesRequest, stream v1API.UnpaperService_ListenForMessagesServer) error {
	ctx := stream.Context()
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.Channel == "" {
		return status.Errorf(codes.InvalidArgument, "missing channel")
	}
	var sendErr error
	done := make(chan struct{})

	// Verify room access
	// access, err := roomAccessCheck(stream.Context(), s.db, req.Channel, userID)
	// if err != nil {
	// 	return status.Error(codes.Internal, err.Error())
	// }
	// if access.Authorization != v1API.RoomAuthorization_AUTHORIZED {
	// 	return status.Error(codes.Internal, "unauthorized")
	// }

	// Mark previous messages as read
	_, err := s.chat.ReadConversationMessages(ctx, userID, req.Channel)
	if err != nil {
		sendErr = status.Errorf(codes.Internal, "failed to read previous conversation messages")
		close(done)
	}
	// Set active conversation
	s.chat.SetActiveConversation(ctx, req.Channel, req.Channel)
	// Clear active conversation on defer
	defer func() {
		s.chat.DeleteActiveConversation(ctx, userID)
	}()

	go func() {
		defer close(done)
		messages := s.chat.ListenForMessages(ctx, userID, req.Channel)
		for {
			select {
			case <-ctx.Done():
				return
			case out, ok := <-messages:
				if !ok {
					sendErr = fmt.Errorf("closed msgs channel")
					return
				}
				if err := stream.Send(out.ToProtobuf()); err != nil {
					sendErr = err
				}
			}
		}
	}()

	<-done
	return sendErr
}

// GetMessages returns the conversation messages
func (s *unpaperServiceServer) GetMessages(ctx context.Context, req *v1API.GetMessagesRequest) (*v1API.GetMessagesResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.Channel == "" {
		return nil, status.Errorf(codes.InvalidArgument, "missing channel")
	}
	messagesRes, err := s.chat.GetMessages(ctx, userID, req.Channel, req.Offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve chat messages: %v", err)
	}

	return messagesRes, nil
}

// SendMessage sends a message to the param passed channel, returning any error if occurred
func (s *unpaperServiceServer) SendMessage(ctx context.Context, req *v1API.SendMessageRequest) (*empty.Empty, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return new(empty.Empty), status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.Content == "" {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "missing content")
	}
	if req.Channel == "" {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "missing channel")
	}
	if req.Username == "" {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "missing username")
	}
	// Verify room access
	// access, err := roomAccessCheck(ctx, s.db, req.Channel, userID)
	// if err != nil {
	// 	return nil, status.Error(codes.Internal, err.Error())
	// }
	// if access.Authorization != v1API.RoomAuthorization_AUTHORIZED {
	// 	return nil, status.Error(codes.Internal, "unauthorized")
	// }

	err := s.chat.SendMessage(ctx, req.Channel, &message.Message{
		ID:             uuid.New().String(),
		UserID:         userID,
		CreatedAt:      time.Now(),
		Text:           message.Text{Content: req.Content},
		SenderUsername: req.Username,
	})
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "error sending message: %q", err)
	}
	return new(empty.Empty), nil
}

// SendAward sends an award to the param passed channel, returning any error if occurred
func (s *unpaperServiceServer) SendAward(ctx context.Context, req *v1API.SendAwardRequest) (*empty.Empty, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return new(empty.Empty), status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.AwardId == "" {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "missing content")
	}
	if req.Channel == "" {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "missing channel")
	}
	if req.Username == "" {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "missing username")
	}
	// Verify room access
	// access, err := roomAccessCheck(ctx, s.db, req.Channel, userID)
	// if err != nil {
	// 	return nil, status.Error(codes.Internal, err.Error())
	// }
	// if access.Authorization != v1API.RoomAuthorization_AUTHORIZED {
	// 	return nil, status.Error(codes.Internal, "unauthorized")
	// }

	err := s.chat.SendMessage(ctx, req.Channel, &message.Message{
		ID:             uuid.New().String(),
		UserID:         userID,
		CreatedAt:      time.Now(),
		Award:          message.Award{AwardID: req.AwardId},
		SenderUsername: req.Username,
	})
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "error sending message: %q", err)
	}
	return new(empty.Empty), nil
}

// SendDonation sends a donation type message
func (s *unpaperServiceServer) SendDonation(ctx context.Context, req *v1API.SendDonationRequest) (*empty.Empty, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return new(empty.Empty), status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.Channel == "" {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "missing channel")
	}
	if req.Username == "" {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "missing username")
	}
	if req.Amount == 0 {
		return new(empty.Empty), status.Error(codes.InvalidArgument, "cannot send zero amount donation")
	}
	// Verify room access
	// access, err := roomAccessCheck(ctx, s.db, req.Channel, userID)
	// if err != nil {
	// 	return nil, status.Error(codes.Internal, err.Error())
	// }
	// if access.Authorization != v1API.RoomAuthorization_AUTHORIZED {
	// 	return nil, status.Error(codes.Internal, "unauthorized")
	// }

	err := s.chat.SendMessage(ctx, req.Channel, &message.Message{
		ID:             uuid.NewString(),
		Type:           message.TypeDonation,
		UserID:         userID,
		CreatedAt:      time.Now(),
		SenderUsername: req.Username,
		Donation: message.Donation{
			Amount: req.Amount,
		},
	})
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "error sending donation")
	}

	return new(empty.Empty), nil
}

func (s *unpaperServiceServer) SendAudio(ctx context.Context, req *v1API.SendAudioRequest) (*empty.Empty, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return new(empty.Empty), status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.Channel == "" {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "missing channel")
	}
	if req.Username == "" {
		return new(empty.Empty), status.Errorf(codes.InvalidArgument, "missing username")
	}
	if req.Audio == nil {
		return new(empty.Empty), status.Error(codes.InvalidArgument, "cannot send nil audio")
	}
	// Verify room access
	// access, err := roomAccessCheck(ctx, s.db, req.Channel, userID)
	// if err != nil {
	// 	return nil, status.Error(codes.Internal, err.Error())
	// }
	// if access.Authorization != v1API.RoomAuthorization_AUTHORIZED {
	// 	return nil, status.Error(codes.Internal, "unauthorized")
	// }

	err := s.chat.SendMessage(ctx, req.Channel, &message.Message{
		ID:             uuid.NewString(),
		Type:           message.TypeDonation,
		UserID:         userID,
		CreatedAt:      time.Now(),
		SenderUsername: req.Username,
		Audio: message.Audio{
			Bytes: req.Audio,
		},
	})
	if err != nil {
		return new(empty.Empty), status.Errorf(codes.Internal, "failed to send chat audio")
	}
	return new(empty.Empty), nil
}

// RoomAccessCheck checks whether the user is in any of the room allowed list,
// or automatically authenticates if the room is public
func (s *unpaperServiceServer) RoomAccessCheck(ctx context.Context, req *v1API.RoomAccessCheckRequest) (*v1API.RoomAccessCheckResponse, error) {
	// TODO: check

	// 	userID, ok := mdutils.GetUserIDFromMD(ctx)
	// 	if !ok {
	// 		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	// 	}
	// 	if req.RoomId == "" {
	// 		return nil, status.Error(codes.InvalidArgument, "missing room id")
	// 	}

	// 	return roomAccessCheck(ctx, s.db, req.RoomId, userID)
	// }

	// func roomAccessCheck(ctx context.Context, db *sql.DB, roomID string, userID string) (*v1API.RoomAccessCheckResponse, error) {
	// 	customersDir := customers.NewDirectory(db)
	// 	roomsDir := rooms.NewDirectory(db)

	// 	// Get room by id
	// 	room, err := roomsDir.GetRoomByID(ctx, roomID)
	// 	if err != nil {
	// 		return nil, status.Errorf(codes.Internal, "failed to retrieve room: %v", err)
	// 	}

	// 	userIDsWhoPaid, err := customersDir.GetUserIDsSubscribedToRoom(ctx, room.Id)
	// 	if err != nil {
	// 		return nil, status.Errorf(codes.Internal, "failed to get user ids who paid: %v", err)
	// 	}
	// 	hasAllowedListIDs := len(room.AllowedListIds) > 0
	// 	isRoomFreeToJoin := room.RoomType == v1API.RoomType_FREE

	// 	if room.Visibility == v1API.Visibility_PUBLIC && hasAllowedListIDs {
	// 		return nil, status.Error(codes.FailedPrecondition, "public rooms cannot have lists")
	// 	}

	// 	// Room owner has always authorization
	// 	if room.Owner == userID {
	// 		return roomAccessCheckResponse(v1API.RoomAuthorization_AUTHORIZED)
	// 	}

	// 	// Public room
	// 	if room.Visibility == v1API.Visibility_PUBLIC {
	// 		// Public free room can be joined by anyone
	// 		if isRoomFreeToJoin {
	// 			return roomAccessCheckResponse(v1API.RoomAuthorization_AUTHORIZED)
	// 		}

	// 		// Public paid room can be joined by anyone who had paid
	// 		if !isRoomFreeToJoin {
	// 			if room.RoomType == v1API.RoomType_PAID {
	// 				return handlePaidRoom(userIDsWhoPaid, userID)
	// 			}
	// 			if room.RoomType == v1API.RoomType_SUBSCRIPTION_MONTHLY {
	// 				return handleSubscriptionRoom(ctx, customersDir, room.Id, userID)
	// 			}
	// 			// Unexpected room type
	// 			return nil, status.Errorf(codes.Internal, "unexpected room type: %v", room.RoomType)
	// 		}
	// 	}

	// 	// Private room
	// 	if room.Visibility == v1.Visibility_PRIVATE {
	// 		isUserInList, err := userInList(ctx, db, room.AllowedListIds, userID)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		if !isUserInList {
	// 			// User cannot access room
	// 			return roomAccessCheckResponse(v1API.RoomAuthorization_UNJOINABLE)
	// 		}

	// 		// User is in list
	// 		// Private room joinable by paying users
	// 		if room.RoomType == v1API.RoomType_PAID {
	// 			return handlePaidRoom(userIDsWhoPaid, userID)
	// 		}

	// 		if room.RoomType == v1API.RoomType_SUBSCRIPTION_MONTHLY {
	// 			return handleSubscriptionRoom(ctx, customersDir, roomID, userID)
	// 		}

	// 		return roomAccessCheckResponse(v1API.RoomAuthorization_AUTHORIZED)
	// 	}

	// 	// Should never come at this point
	// 	return &v1API.RoomAccessCheckResponse{
	// 		Authorization: v1API.RoomAuthorization_UNJOINABLE,
	// 	}, nil
	return nil, nil
}

// handleSubscriptionRoom checks if the user have an active room subscription on db,
// returning the appropriate access check response. This function must be called after
// an eventual list check, as it only returns a subscription check.
func handleSubscriptionRoom(ctx context.Context, customersDir *customers.Directory, roomID, userID string) (*v1API.RoomAccessCheckResponse, error) {
	subsc, err := customersDir.GetRoomSubscriptionForUserID(ctx, &customers.GetRoomSubscriptionForUserIDParams{
		RoomID: roomID,
		UserID: userID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			// Customer hasn't subscribed
			return roomAccessCheckResponse(v1API.RoomAuthorization_NEED_TO_SUBSCRIBE)
		}
		return roomAccessCheckResponse(v1API.RoomAuthorization_UNJOINABLE)
	}

	// Check subscription status
	if subsc.Status == v1API.SubscriptionStatus_ACTIVE {
		// User has active subscription. Authorized
		return roomAccessCheckResponse(v1API.RoomAuthorization_AUTHORIZED)
	}

	// User subscription is in an errored state
	return roomAccessCheckResponse(v1API.RoomAuthorization_NEED_TO_SUBSCRIBE)
}

// handlePaidRoom is a handler for one-time paid rooms
func handlePaidRoom(userIDsWhoPaid []string, userID string) (*v1API.RoomAccessCheckResponse, error) {
	if helpers.StringSliceContains(userIDsWhoPaid, userID) {
		// User have paid
		return roomAccessCheckResponse(v1API.RoomAuthorization_AUTHORIZED)
	}

	// User have NOT paid
	return roomAccessCheckResponse(v1API.RoomAuthorization_NEED_TO_PAY)
}

func roomAccessCheckResponse(authorization v1API.RoomAuthorization_Enum) (*v1API.RoomAccessCheckResponse, error) {
	return &v1API.RoomAccessCheckResponse{
		Authorization: authorization,
	}, nil
}

// userInList returns whether a user is present in any of the list IDs.
// Every listID is used to fetch list members, which are compared angainst the userID
func userInList(ctx context.Context, db *sql.DB, listIDs []string, userID string) (bool, error) {
	listsDir := lists.NewDirectory(db)

	for _, l := range listIDs {
		listFound, _ := listsDir.GetListByID(ctx, l) // Ignore error
		_, ok := listFound.AllowedUsers[userID]
		if ok {
			return true, nil
		}
	}

	return false, nil
}

// CreateList creates a new list
func (s *unpaperServiceServer) CreateList(ctx context.Context, req *v1API.CreateListRequest) (*v1API.List, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "missing room name")
	}

	listsDir := lists.NewDirectory(s.db)

	rawList, err := dbentities.NewListFromMap(req.AllowedUsers).ToRawJSON()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal listMap: %v", err)
	}

	l, err := listsDir.CreateList(ctx, &lists.CreateListParams{
		ID:           uuid.NewString(),
		Name:         req.Name,
		AllowedUsers: rawList,
		OwnerUserID:  userID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create list: %v", err)
	}

	return l, nil
}

// UpdateList updates the list by ID, setting new members and/or new name
func (s *unpaperServiceServer) UpdateList(ctx context.Context, req *v1API.UpdateListRequest) (*v1API.List, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "list name cannot be empty")
	}
	listsDir := lists.NewDirectory(s.db)

	// Retrieve list
	listToEdit, err := listsDir.GetListByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve list: %v", err)
	}

	// Current map of allowedUserIDs
	allowedUserIDsMap := dbentities.NewListFromMap(listToEdit.AllowedUsers)
	// Map of request allowedUserIDs
	newAllowedUserIDsMap := dbentities.NewListFromMap(req.AllowedUsers)

	var l *v1API.List
	if req.Name != listToEdit.Name {
		// Should update name
		l, err = listsDir.UpdateName(ctx, &lists.UpdateNameParams{
			Name: req.Name,
			ID:   req.Id,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update list name: %v", err)
		}
	}

	rawList, err := newAllowedUserIDsMap.ToRawJSON()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal list: %v", err)
	}

	if !reflect.DeepEqual(newAllowedUserIDsMap, allowedUserIDsMap) {
		l, err = listsDir.UpdateAllowedUsers(ctx, &lists.UpdateAllowedUsersParams{
			ID:           req.Id,
			AllowedUsers: rawList,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update allowed_users: %v", err)
		}
	}

	if req.Name == listToEdit.Name && reflect.DeepEqual(newAllowedUserIDsMap, allowedUserIDsMap) {
		// No changes were made
		return listToEdit, nil
	}

	return l, nil
}

// GetUserSuggestions returns a fixed length lists of users that matches the passed query
func (s *unpaperServiceServer) GetUserSuggestions(ctx context.Context, req *v1API.GetUserSuggestionsRequest) (*v1API.GetUserSuggestionsResponse, error) {
	if req.Query == "" {
		// Return nothing on empty query
		return &v1API.GetUserSuggestionsResponse{Users: []*v1API.UserSuggestion{}}, nil
	}

	usersDir := users.NewDirectory(s.db)

	res, err := usersDir.GetUserSuggestions(ctx, req.Query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get users suggestions: %v", err)
	}

	return &v1API.GetUserSuggestionsResponse{
		Users: res,
	}, nil
}

func (s *unpaperServiceServer) GetAllLists(ctx context.Context, req *empty.Empty) (*v1API.GetAllListsResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	listsDir := lists.NewDirectory(s.db)

	allLists, err := listsDir.GetListsByOwnerID(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get all lists: %v", err)
	}

	return &v1API.GetAllListsResponse{
		Lists: allLists,
	}, nil
}

func (s *unpaperServiceServer) GetListByID(ctx context.Context, req *v1API.GetListByIDRequest) (*v1API.List, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "missing list id")
	}

	listsDir := lists.NewDirectory(s.db)

	res, err := listsDir.GetListByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get all lists: %v", err)
	}

	return res, nil
}

func (s *unpaperServiceServer) CreateConversation(ctx context.Context, req *v1API.CreateConversationRequest) (*v1API.CreateConversationResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	usersDir := users.NewDirectory(s.db)
	senderUser, err := usersDir.GetUser(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve participant user: %v", err)
	}
	receiverUser, err := usersDir.GetUserByUsername(ctx, req.ParticipantUsername)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve participant user: %v", err)
	}

	_, err = s.getConversationWithUser(ctx, userID, receiverUser.Id)
	if err != nil {
		// A conversation already exists, or another error occurred
		// return nil, err TODO. check
	}

	conv := conversation.New(senderUser, receiverUser)

	err = s.chat.CreateConversation(ctx, conv)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save conversation: %v", err)
	}

	return &v1API.CreateConversationResponse{
		Conversation: conv.ToProtobuf(),
	}, nil
}

// TODO: Support multiple users
func (s *unpaperServiceServer) getConversationWithUser(ctx context.Context, userID, receiverUserID string) (*v1API.Conversation, error) {
	prevConv, err := s.chat.GetConversationsWithUser(ctx, userID, receiverUserID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve previous conversations with user")
	}

	// Check if a previous conversation with the same participants already exists
	if len(prevConv) > 0 {
		for _, c := range prevConv {
			if len(c.Participants) != 2 { // TODO: A conversation can be made with more participants
				continue
			}

			prevConvParticipantsList := make([]string, len(c.Participants))

			i := 0
			for k := range c.Participants {
				prevConvParticipantsList[i] = k
				i++
			}

			// Check if slice of prev participants has same elements as a slice composed by the userID
			// and the requestedParticipants
			if helpers.SameStringSlice(prevConvParticipantsList, []string{userID, receiverUserID}) {
				// User already has a conversation with target participants
				return c, nil
			}
		}
	}

	return nil, ErrConversationNotFound
}

func (s *unpaperServiceServer) GetConversation(ctx context.Context, req *v1API.GetConversationRequest) (*v1API.GetConversationResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	conv, err := s.chat.GetConversation(ctx, userID, req.ConversationId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve conversation: %v", err)
	}

	return &v1API.GetConversationResponse{
		Conversation: conv,
	}, nil
}

func (s *unpaperServiceServer) GetConversations(ctx context.Context, req *v1API.GetConversationsRequest) (*v1API.GetConversationsResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	conv, err := s.chat.GetConversations(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve conversation: %v", err)
	}

	return &v1API.GetConversationsResponse{
		Conversations: conv,
	}, nil
}

func (s *unpaperServiceServer) GetConversationWithParticipants(ctx context.Context, req *v1API.GetConversationWithParticipantsRequest) (*v1API.GetConversationWithParticipantsResponse, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}

	if req.UserIds == nil || len(req.UserIds) != 1 { // TODO: support multiple users
		return nil, status.Error(codes.InvalidArgument, "missing userids or wrong length provided")
	}

	prevConv, err := s.getConversationWithUser(ctx, userID, req.UserIds[0])
	if err != nil {
		if err == ErrConversationNotFound {
			// Conversation not found. Do not return error
			return &v1API.GetConversationWithParticipantsResponse{
				Conversation: nil,
				Found:        false,
			}, nil
		}
		// Other error occurred
		return nil, err
	}

	return &v1API.GetConversationWithParticipantsResponse{
		Conversation: prevConv,
		Found:        true,
	}, nil
}
