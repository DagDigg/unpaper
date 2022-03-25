package chat

import (
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/chat/conversation"
)

// Conversation interface describes the conversation behaviour
type Conversation interface {
	ToProtobuf() *v1API.Conversation
	EncodeBinary() (string, error)
	DecodeBinary(string) error
	GetRaw() *conversation.Conversation
}

// Conversation must implement chat.Conversation interface
var _ Conversation = (*conversation.Conversation)(nil)
