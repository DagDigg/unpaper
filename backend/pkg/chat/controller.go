package chat

import (
	"context"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
)

// Controller for the chat package
type Controller interface {
	SendMessage(ctx context.Context, ch string, msg Message) error
	GetMessages(ctx context.Context, userID, ch string, offset int64) (*v1API.GetMessagesResponse, error)
	ListenForMessages(ctx context.Context, userID, ch string) <-chan Message
	SetActiveConversation(ctx context.Context, userID, conversationID string) error
	DeleteActiveConversation(ctx context.Context, userID string) error
	GetConversation(ctx context.Context, userID, conversationID string) (*v1API.Conversation, error)
	GetConversations(ctx context.Context, userID string) ([]*v1API.Conversation, error)
	CreateConversation(ctx context.Context, conversation Conversation) error
	GetConversationsWithUser(ctx context.Context, userID, targetUserID string) ([]*v1API.Conversation, error)
	ReadConversationMessages(ctx context.Context, userID, conversationID string) (*v1API.Conversation, error)
}
