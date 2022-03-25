package chat

import (
	"context"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
)

// Usecase for chat package
type Usecase interface {
	GetMessages(ctx context.Context, userID, ch string, offset int64) ([]*v1API.ChatMessage, error)
	Subscribe(ctx context.Context, ch string) <-chan Message
	SendMessage(ctx context.Context, ch string, msg Message) error
	CreateConversation(ctx context.Context, conversation Conversation) error
	SetActiveConversation(ctx context.Context, userID, conversationID string) error
	DeleteActiveConversation(ctx context.Context, userID string) error
	GetConversation(ctx context.Context, userID, conversationID string) (*v1API.Conversation, error)
	GetConversations(ctx context.Context, userID string) ([]*v1API.Conversation, error)
	GetConversationsWithUser(ctx context.Context, userID, targetUserID string) ([]*v1API.Conversation, error)
	ReadConversationMessages(ctx context.Context, userID, conversationID string) (*v1API.Conversation, error)
	GetConversationInactiveUsers(ctx context.Context, senderUserID, conversationID string) ([]string, error)
	IncrementInactiveUserMsgsCount(ctx context.Context, userID, conversationID string) error
}
