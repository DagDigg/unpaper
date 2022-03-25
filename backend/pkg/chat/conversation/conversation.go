package conversation

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"time"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/chat/message"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MessagesLimit denotes the maximum length of messages that can be returned as a response
const MessagesLimit = 10

// MaxChatMessages denotes the maximum length of messages that can be stored in the redis sorted set
const MaxChatMessages = 1000

// Conversation data structure which describe a chat room
type Conversation struct {
	ID                  string
	Participants        map[string]Participant
	CreatedAt           time.Time
	UnreadMessagesCount int64
	LastMessage         *message.Message
}

// Participant of a conversation
type Participant struct {
	UserID   string
	Username string
	JoinedAt time.Time
}

// New creates a new conversation with zero unread messages and with timestamp fields set to `now`
func New(participants ...*v1API.User) *Conversation {
	p := make(map[string]Participant, len(participants))

	for _, participant := range participants {
		p[participant.Id] = Participant{
			UserID:   participant.Id,
			Username: participant.Username,
			JoinedAt: time.Now(),
		}
	}

	return &Conversation{
		ID:                  uuid.NewString(),
		Participants:        p,
		CreatedAt:           time.Now(),
		UnreadMessagesCount: 0,
		LastMessage:         nil,
	}
}

// ToProtobuf converts a conversation to proto data structure
func (c *Conversation) ToProtobuf() *v1API.Conversation {
	conv := &v1API.Conversation{
		Id:                  c.ID,
		Participants:        participantsMapToProtobuf(c.Participants),
		CreatedAt:           timestamppb.New(c.CreatedAt),
		UnreadMessagesCount: c.UnreadMessagesCount,
	}
	if c.LastMessage != nil {
		conv.LastMessage = c.LastMessage.ToProtobuf()
	}

	return conv
}

// ToProtobuf converts a participant to proto data structure
func (p *Participant) ToProtobuf() *v1API.ConversationParticipant {
	return &v1API.ConversationParticipant{
		UserId:   p.UserID,
		Username: p.Username,
		JoinedAt: timestamppb.New(p.JoinedAt),
	}
}

// participantsMapToProtobuf converts a participant map to proto data structure
func participantsMapToProtobuf(p map[string]Participant) map[string]*v1API.ConversationParticipant {
	participants := make(map[string]*v1API.ConversationParticipant)
	for userID, participant := range p {
		participants[userID] = participant.ToProtobuf()
	}
	return participants
}

// EncodeBinary returns the base64 encoded form of the conversation
func (c *Conversation) EncodeBinary() (string, error) {
	b := &bytes.Buffer{}
	e := gob.NewEncoder(b)
	if err := e.Encode(c); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

// DecodeBinary decodes the base64 encoded string into the conversation struct
func (c *Conversation) DecodeBinary(str string) error {
	s, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return err
	}

	b := &bytes.Buffer{}
	_, err = b.Write(s)
	if err != nil {
		return err
	}

	d := gob.NewDecoder(b)
	return d.Decode(c)
}

// GetRaw returns the underlying conversation struct
func (c *Conversation) GetRaw() *Conversation {
	return c
}

// GetUserConversationsKey returns the key used for storing the conversations the user belongs to.
// An user could decide to remove a conversation, which means that he wants to `hide` it until new messages are received.
// The user will still be part of the conversation even if its removed.
// In order to permanently remove the user from the conversation, it must be deleted from the `participants` in the main conversation
func GetUserConversationsKey(userID string) string {
	return "user:" + userID + "conversations"
}

// GetConversationMessagesKey returns the key used for storing encoded messages in a specific conversation.
// e.g. `conversations:{id}:messages *Message{}` where Message is the base64 encoded message
func GetConversationMessagesKey(conversationID string) string {
	return "conversations:" + conversationID + ":messages"
}

// GetConversationPubSubKey returns the key used for storing the pubsub messages in the conversation
func GetConversationPubSubKey(conversationID string) string {
	return "conversation:pubsub:" + conversationID
}

// GetUserConversationsLastReadKey returns an rdb key used for storing last_read timestamp on specific conversation ids
func GetUserConversationsLastReadKey(userID string) string {
	return "conversations:last_read:" + userID
}

// GetActiveUserConversationIDKey returns a key used for storing the user's active conversation ID
func GetActiveUserConversationIDKey(userID string) string {
	return "conversations:active" + userID
}
