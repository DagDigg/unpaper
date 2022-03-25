package chat

import (
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/chat/message"
)

type Message interface {
	ToProtobuf() *v1API.ChatMessage
	EncodeBinary() (string, error)
	DecodeBinary(string) error
	GetRaw() *message.Message
}
