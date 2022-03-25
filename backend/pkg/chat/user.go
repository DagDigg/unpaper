package chat

import (
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/chat/user"
)

type User interface {
	ToProtobuf() *v1API.ChatUser
	EncodeBinary() (string, error)
	DecodeBinary(string) error
	GetRaw() *user.User
}
