package controller

import (
	"context"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/chat"
	"github.com/DagDigg/unpaper/backend/pkg/chat/conversation"
)

type ctrl struct {
	ucs chat.Usecase
}

// New returns a new chat.Controller
func New(ucs chat.Usecase) chat.Controller {
	return &ctrl{
		ucs: ucs,
	}
}

func (c *ctrl) SendMessage(ctx context.Context, ch string, msg chat.Message) error {
	// Send message
	if err := c.ucs.SendMessage(ctx, ch, msg); err != nil {
		return err
	}

	// Increment unread messages count to inactive users, if any
	inactiveUsers, err := c.ucs.GetConversationInactiveUsers(ctx, msg.GetRaw().UserID, ch)
	if err != nil {
		return err
	}

	if len(inactiveUsers) > 0 {
		for _, u := range inactiveUsers {
			if err := c.ucs.IncrementInactiveUserMsgsCount(ctx, u, ch); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *ctrl) GetMessages(ctx context.Context, userID, ch string, offset int64) (*v1API.GetMessagesResponse, error) {
	messages := []*v1API.ChatMessage{}

	// Get old messages
	messages, err := c.ucs.GetMessages(ctx, userID, ch, offset)
	if err != nil {
		return nil, err
	}

	hasMore := len(messages) > conversation.MessagesLimit

	res := &v1API.GetMessagesResponse{
		HasMore: hasMore,
	}

	if hasMore {
		if len(messages) != 0 {
			res.Messages = messages[0 : len(messages)-1]
			return res, nil
		}
	}

	res.Messages = messages
	return res, nil
}

func (c *ctrl) ListenForMessages(ctx context.Context, userID, ch string) <-chan chat.Message {
	messages := make(chan chat.Message)

	// Subscribe for new messages
	go func() {
		defer close(messages)

		newMsgs := c.ucs.Subscribe(ctx, ch)
		for {
			newMsg, ok := <-newMsgs
			if !ok {
				return
			}
			messages <- newMsg
		}
	}()

	return messages
}

// CreateConversation creates a new conversation on the in memory datastore
func (c *ctrl) CreateConversation(ctx context.Context, conversation chat.Conversation) error {
	return c.ucs.CreateConversation(ctx, conversation)
}

// SetActiveConversation sets the user active conversation
func (c *ctrl) SetActiveConversation(ctx context.Context, userID, conversationID string) error {
	return c.ucs.SetActiveConversation(ctx, userID, conversationID)
}

// DeleteActiveConversation deletes the user active conversation
func (c *ctrl) DeleteActiveConversation(ctx context.Context, userID string) error {
	return c.ucs.DeleteActiveConversation(ctx, userID)
}

// GetConversation retrieves the stored conversation by ID
func (c *ctrl) GetConversation(ctx context.Context, userID, conversationID string) (*v1API.Conversation, error) {
	return c.ucs.GetConversation(ctx, userID, conversationID)
}

// GetConversations retrieves the stored conversations for the userID
func (c *ctrl) GetConversations(ctx context.Context, userID string) ([]*v1API.Conversation, error) {
	return c.ucs.GetConversations(ctx, userID)
}

// GetConversationsWithUser retrieves the stored conversations between the userID and target userID
func (c *ctrl) GetConversationsWithUser(ctx context.Context, userID, targetUserID string) ([]*v1API.Conversation, error) {
	return c.ucs.GetConversationsWithUser(ctx, userID, targetUserID)
}

// ReadConversationMessages sets the `UnreadMessagesCount` to zero and updates the hashmap conversation value
func (c *ctrl) ReadConversationMessages(ctx context.Context, userID, conversationID string) (*v1API.Conversation, error) {
	return c.ucs.ReadConversationMessages(ctx, userID, conversationID)
}
