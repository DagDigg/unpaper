package v1_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestMessages(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When subscribing and sending a message", func(t *testing.T) {
		// Create public room in order to have access to it
		userParams := v1Testing.GetRandomPGUserParams()
		user, err := ws.AddUser(userParams)
		assert.Nil(err)

		ctx, cancel := context.WithCancel(metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", user.Id)))
		mockedStream := newMockStream()
		mockedStream.ContextFunc = func() context.Context { return ctx }

		createConnectedCustomer(t, ctx, ws, user.Id)
		conversationID := uuid.NewString()

		_, err = ws.Server.SendMessage(ctx, &v1API.SendMessageRequest{
			Channel:  conversationID,
			Content:  "first",
			Username: user.Username,
		})
		assert.Nil(err)

		go func() {
			getMsgsReq := &v1API.ListenForMessagesRequest{Channel: conversationID}
			err := ws.Server.ListenForMessages(getMsgsReq, mockedStream)
			if err != nil {
				t.Errorf("error calling GetMessages RPC: %q", err)
			}
		}()

		time.Sleep(time.Second)
		_, err = ws.Server.SendMessage(ctx, &v1API.SendMessageRequest{
			Channel:  conversationID,
			Content:  "second",
			Username: "Bob",
		})
		assert.Nil(err)

		time.Sleep(time.Second)
		cancel()
	})
}

func TestCreateList(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When successfully creating a list", func(t *testing.T) {
		t.Parallel()
		var (
			listName = "List name"
		)

		t.Run("When there are no allowed user ids", func(t *testing.T) {
			t.Parallel()
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "myID"))
			res := createEmptyList(ctx, listName, ws, t)
			wantAllowedUsers := make(map[string]string)
			assert.Equal(res.Name, listName)
			assert.Equal(res.AllowedUsers, wantAllowedUsers)
		})

		t.Run("When there are allowed user ids", func(t *testing.T) {
			t.Parallel()
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "myID"))
			allowedUsers := map[string]string{
				"oneID": "oneUser",
				"twoID": "twoUser",
			}
			req := &v1API.CreateListRequest{
				Name:         listName,
				AllowedUsers: allowedUsers,
			}

			res, err := ws.Server.CreateList(ctx, req)
			assert.Nil(err)

			assert.Equal(res.Name, listName)
			assert.Equal(res.AllowedUsers, allowedUsers)
		})

		t.Run("When a list is created without a name", func(t *testing.T) {
			t.Parallel()
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "myID"))
			allowedUsers := map[string]string{
				"oneID": "oneUser",
				"twoID": "twoUser",
			}
			req := &v1API.CreateListRequest{
				AllowedUsers: allowedUsers,
			}

			_, err := ws.Server.CreateList(ctx, req)
			assert.NotNil(err)
		})
	})
}

func TestUpdateList(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When succesfully creating and updating a list", func(t *testing.T) {
		t.Parallel()

		t.Run("When updating name and members", func(t *testing.T) {
			t.Parallel()
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "myID"))
			l := createEmptyList(ctx, "MyName", ws, t)

			newMembers := map[string]string{
				"oneID": "oneUser",
				"twoID": "twoUser",
			}
			newName := "Different"
			req := &v1API.UpdateListRequest{
				Id:           l.Id,
				Name:         newName,
				AllowedUsers: newMembers,
			}
			res, err := ws.Server.UpdateList(ctx, req)
			assert.Nil(err)

			assert.Equal(res.AllowedUsers, newMembers)
			assert.Equal(res.Name, newName)
		})

		t.Run("When updating name", func(t *testing.T) {
			t.Parallel()
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "myID"))
			l := createEmptyList(ctx, "MyName", ws, t)

			newMembers := make(map[string]string)
			newName := "Different"
			req := &v1API.UpdateListRequest{
				Id:   l.Id,
				Name: newName,
			}
			res, err := ws.Server.UpdateList(ctx, req)
			assert.Nil(err)

			assert.Equal(res.AllowedUsers, newMembers)
			assert.Equal(res.Name, newName)
		})

	})

	t.Run("When failing to update list", func(t *testing.T) {
		t.Parallel()
		t.Run("When updating allowed members", func(t *testing.T) {
			t.Parallel()
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "myID"))
			listName := "MyName"
			l := createEmptyList(ctx, listName, ws, t)

			req := &v1API.UpdateListRequest{
				Id: l.Id,
			}
			_, err := ws.Server.UpdateList(ctx, req)
			assert.NotNil(err)
		})
	})
}

func TestGetAllLists(t *testing.T) {
	t.Parallel()
	ws := v1Testing.GetWrappedServer(t)
	assert := assert.New(t)

	t.Run("When inserting a list and retrieving all", func(t *testing.T) {
		t.Parallel()
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "myID"))
		createReq := &v1API.CreateListRequest{
			Name: "OK",
			AllowedUsers: map[string]string{
				"1": "one",
				"2": "two",
			},
		}
		_, err := ws.Server.CreateList(ctx, createReq)
		assert.Nil(err)
		res, err := ws.Server.GetAllLists(ctx, new(empty.Empty))
		assert.Nil(err)

		assert.Equal(len(res.Lists), 1)
		assert.Equal(res.Lists[0].AllowedUsers, createReq.AllowedUsers)
	})
}

func TestRoomAccessCheck(t *testing.T) {
	// t.Parallel()
	// ws := v1Testing.GetWrappedServer(t)
	// assert := assert.New(t)

	t.Run("When user is authorized", func(t *testing.T) {
		// TODO: check

		// 	roomOwnerID := uuid.NewString()
		// 	userID := uuid.NewString()
		// 	allowedUsers := map[string]string{
		// 		userID:  "username1",
		// 		"user2": "username2",
		// 	}

		// 	// Use context with user id as the room owner. This because when doing the access check,
		// 	// the user requesting the access should be different from the owner, otherwise it will always return access
		// 	createRoomCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", roomOwnerID))
		// 	createConnectedCustomer(t, createRoomCtx, ws, roomOwnerID)
		// 	createRoomRes := createRoomWithList(t, createRoomCtx, ws, allowedUsers)

		// 	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", userID))
		// 	// Check if user has access to the room
		// 	accessReq := &v1API.RoomAccessCheckRequest{
		// 		RoomId: createRoomRes.Id,
		// 	}
		// 	accessRes, err := ws.Server.RoomAccessCheck(ctx, accessReq)
		// 	assert.Nil(err)

		// 	assert.Equal(accessRes.Authorization, v1API.RoomAuthorization_AUTHORIZED)
		// })

		// t.Run("When user is not authorized", func(t *testing.T) {
		// 	roomOwnerID := uuid.NewString()
		// 	userID := uuid.NewString()
		// 	allowedUsers := map[string]string{
		// 		"user1": "username1",
		// 		"user2": "username2",
		// 	}

		// 	// Use context with user id as the room owner. This because when doing the access check,
		// 	// the user requesting the access should be different from the owner, otherwise it will always return access
		// 	createRoomCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", roomOwnerID))
		// 	createConnectedCustomer(t, createRoomCtx, ws, roomOwnerID)
		// 	createRoomRes := createRoomWithList(t, createRoomCtx, ws, allowedUsers)

		// 	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", userID))
		// 	// Check if user has access to the room
		// 	accessReq := &v1API.RoomAccessCheckRequest{
		// 		RoomId: createRoomRes.Id,
		// 	}
		// 	accessRes, err := ws.Server.RoomAccessCheck(ctx, accessReq)
		// 	assert.Nil(err)

		// 	assert.Equal(accessRes.Authorization, v1API.RoomAuthorization_UNJOINABLE)
		// })

		// t.Run("When room is public, free and have list", func(t *testing.T) {
		// 	userID := uuid.NewString()
		// 	allowedUsers := map[string]string{
		// 		"user1": "username1",
		// 		"user2": "username2",
		// 	}
		// 	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", userID))
		// 	createConnectedCustomer(t, ctx, ws, userID)
		// 	// Create list
		// 	createListReq := &v1API.CreateListRequest{
		// 		Name:         "OK",
		// 		AllowedUsers: allowedUsers,
		// 	}
		// 	_, err := ws.Server.CreateList(ctx, createListReq)
		// 	assert.Nil(err)

		// 	// TODO: check
		// 	// Create room
		// 	createRoomReq := &v1API.CreateRoomRequest{
		// 		Name:           "MyCoolRoom",
		// 		Description:    "MyCoolDesc",
		// 		Visibility:     v1API.Visibility_PUBLIC,
		// 		AllowedListIds: []string{createdList.Id},
		// 		RoomType:       v1API.RoomType_FREE,
		// 	}

		// 	_, err = ws.Server.CreateRoom(ctx, createRoomReq)
		// 	assert.NotNil(err)
	})

	t.Run("When room is one time pay to join and public", func(t *testing.T) {
		// TODO: Check

		// roomOwnerID := uuid.NewString()
		// userID := uuid.NewString()
		// ownerCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", roomOwnerID))
		// createConnectedCustomer(t, ownerCtx, ws, roomOwnerID)

		// // Create room
		// createRoomReq := &v1API.CreateRoomRequest{
		// 	Name:        "MyCoolRoom",
		// 	Description: "MyCoolDesc",
		// 	Visibility:  v1API.Visibility_PUBLIC,
		// 	Price:       10, // Paid
		// 	RoomType:    v1API.RoomType_PAID,
		// }
		// createRoomRes, err := ws.Server.CreateRoom(ownerCtx, createRoomReq)
		// assert.Nil(err)

		// // Fake user paying
		// customersDir := customers.NewDirectory(ws.Server.GetDB())

		// _, err = customersDir.StoreRoomSubscription(ownerCtx, &customers.StoreRoomSubscriptionParams{
		// 	ID:                   uuid.NewString(),
		// 	UserID:               userID,
		// 	CustomerID:           uuid.NewString(),
		// 	CurrentPeriodEnd:     sql.NullTime{Time: time.Now().Add(1 * time.Hour), Valid: true},
		// 	Status:               string(customers.SubscriptionStatusActive),
		// 	RoomID:               createRoomRes.Id,
		// 	RoomSubscriptionType: string(customers.RoomSubscriptionTypeSubscriptionMonthly),
		// })
		// assert.Nil(err)

		// ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", userID))
		// accessReq := &v1API.RoomAccessCheckRequest{
		// 	RoomId: createRoomRes.Id,
		// }
		// accessRes, err := ws.Server.RoomAccessCheck(ctx, accessReq)
		// assert.Nil(err)

		// assert.Equal(accessRes.Authorization, v1API.RoomAuthorization_AUTHORIZED)
	})

	t.Run("When room requires subscription", func(t *testing.T) {
		// TODO: check

		// roomOwnerID := uuid.NewString()
		// userID := uuid.NewString()
		// ownerCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", roomOwnerID))
		// createConnectedCustomer(t, ownerCtx, ws, roomOwnerID)

		// // Create room
		// createRoomReq := &v1API.CreateRoomRequest{
		// 	Name:        "MyCoolRoom",
		// 	Description: "MyCoolDesc",
		// 	Visibility:  v1API.Visibility_PUBLIC,
		// 	Price:       10, // Paid with subscription
		// 	RoomType:    v1API.RoomType_SUBSCRIPTION_MONTHLY,
		// }
		// createRoomRes, err := ws.Server.CreateRoom(ownerCtx, createRoomReq)
		// assert.Nil(err)

		// // Check if user has access to the room
		// ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", userID))

		// // Bypass e2e payment and store room subscription
		// customersDir := customers.NewDirectory(ws.Server.GetDB())

		// _, err = customersDir.StoreRoomSubscription(ctx, &customers.StoreRoomSubscriptionParams{
		// 	ID:                   uuid.NewString(),
		// 	UserID:               userID,
		// 	CustomerID:           uuid.NewString(),
		// 	CurrentPeriodEnd:     sql.NullTime{Time: time.Now().Add(1 * time.Hour), Valid: true},
		// 	Status:               string(customers.SubscriptionStatusActive),
		// 	RoomID:               createRoomRes.Id,
		// 	RoomSubscriptionType: string(customers.RoomSubscriptionTypeSubscriptionMonthly),
		// })
		// assert.Nil(err)

		// accessReq := &v1API.RoomAccessCheckRequest{
		// 	RoomId: createRoomRes.Id,
		// }
		// accessRes, err := ws.Server.RoomAccessCheck(ctx, accessReq)
		// assert.Nil(err)

		// assert.Equal(accessRes.Authorization, v1API.RoomAuthorization_AUTHORIZED)
	})
}

func createEmptyList(ctx context.Context, name string, ws *v1Testing.WrappedServer, t *testing.T) *v1API.List {
	req := &v1API.CreateListRequest{
		Name: name,
	}

	res, err := ws.Server.CreateList(ctx, req)
	if err != nil {
		t.Errorf("error creating list: %v", err)
	}

	return res
}

func createConnectedCustomer(t *testing.T, ctx context.Context, ws *v1Testing.WrappedServer, userID string) {
	q := ws.Server.GetSB().Insert("customers").
		Columns("id", "customer_id", "first_name", "last_name", "has_connected_account").
		Values(userID, uuid.NewString(), "Bob", "Kelso", true)

	_, err := q.Exec()
	if err != nil {
		t.Error(err)
	}
}

// make and configure a mocked v1.UnpaperService_GetMessagesServer
func newMockStream() *v1Testing.UnpaperService_ListenForMessagesServerMock {
	return &v1Testing.UnpaperService_ListenForMessagesServerMock{
		ContextFunc: func() context.Context {
			return metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-user-id", "myID"))
		},
		RecvMsgFunc: func(m interface{}) error {
			fmt.Printf("\nRECEIVING:\n %+v\n", m)
			return nil
		},
		SendFunc: func(chatMessage *v1API.ChatMessage) error {
			fmt.Printf("%v\n", chatMessage.Text)
			return nil
		},
		SendHeaderFunc: func(mD metadata.MD) error {
			fmt.Printf("\n\n\n CALL SEND HEADER")
			return nil
		},
		SendMsgFunc: func(m interface{}) error {
			fmt.Printf("\n\n\n CALL SEND MSG")
			return nil
		},
		SetHeaderFunc: func(mD metadata.MD) error {
			fmt.Printf("\n\n\n CALL SET HEADER")
			return nil
		},
		SetTrailerFunc: func(mD metadata.MD) {
			fmt.Printf("\n\n\n CALL SET TRAILER")
		},
	}
}
