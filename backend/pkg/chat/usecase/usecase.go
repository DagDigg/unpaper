package usecase

import (
	"context"
	"fmt"
	"strconv"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/chat"
	"github.com/DagDigg/unpaper/backend/pkg/chat/conversation"
	"github.com/DagDigg/unpaper/backend/pkg/chat/message"
	"github.com/DagDigg/unpaper/backend/pkg/lock"
	"github.com/go-redis/redis/v8"
)

type ucs struct {
	rdb  *redis.Client
	lock *lock.Lock
}

// New returns a new chat.Usecase
func New(rdb *redis.Client) chat.Usecase {
	l := lock.New()
	return &ucs{
		rdb:  rdb,
		lock: l,
	}
}

// Subscribe subscribes to the passed channel, listening all the messages
// and delivering them to the returned message channel
func (u *ucs) Subscribe(ctx context.Context, conversationID string) <-chan chat.Message {
	messages := make(chan chat.Message)

	go func() {
		pubsub := u.rdb.Subscribe(ctx, conversation.GetConversationPubSubKey(conversationID))
		defer close(messages)
		defer pubsub.Close()

		// Wait for confirmation that subscription is created before publishing anything.
		_, err := pubsub.Receive(ctx)
		if err != nil {
			return
		}

		go func() {
			<-ctx.Done()
			pubsub.Close()
		}()

		for c := range pubsub.Channel() {
			msg := &message.Message{}
			err = msg.DecodeBinary(c.Payload)
			if err != nil {
				return
			}
			messages <- msg
		}
	}()

	return messages
}

// SendMessage stores the passed message and publish it
// to the provided channel
func (u *ucs) SendMessage(ctx context.Context, conversationID string, msg chat.Message) error {
	msgK := conversation.GetConversationMessagesKey(conversationID)
	pubsubK := conversation.GetConversationPubSubKey(conversationID)

	unlock, err := u.lock.Lock(msgK)
	if err != nil {
		return err
	}
	defer unlock()

	val, err := msg.EncodeBinary()
	if err != nil {
		return err
	}

	zVal := &redis.Z{
		Score:  float64(msg.GetRaw().CreatedAt.UTC().Unix()),
		Member: val,
	}
	err = u.rdb.ZAdd(ctx, msgK, zVal).Err()
	if err != nil {
		return err
	}
	return u.rdb.Publish(ctx, pubsubK, val).Err()
}

// GetMessages returns a *Message channel where all the redis messages are sent
func (u *ucs) GetMessages(ctx context.Context, userID, conversationID string, offset int64) ([]*v1API.ChatMessage, error) {
	// Ignore errors ATM
	unlock, _ := u.lock.RLock(conversationID)
	defer unlock()

	// Get conversation. It is needed because it contains the Joined timestamp
	// used for retrieving messages
	conv, err := u.getConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	participant, ok := conv.Participants[userID]
	if !ok {
		return nil, fmt.Errorf("participant not found: %v", userID)
	}

	messages := []*v1API.ChatMessage{}
	// Get messages from the time user has joined
	res := u.rdb.ZRangeByScore(ctx, conversation.GetConversationMessagesKey(conversationID), &redis.ZRangeBy{
		Min:    strconv.FormatInt(participant.JoinedAt.Unix(), 10),
		Max:    "+inf",
		Offset: offset,
		// Add one to the limit, to know if there are any subsequent messages, useful for loading more
		Count: conversation.MessagesLimit + 1,
	})
	if err := res.Err(); err != nil {
		return nil, res.Err()
	}
	// TODO: slice message limit
	for _, v := range res.Val() {
		m := &message.Message{}
		_ = m.DecodeBinary(v)
		messages = append(messages, m.ToProtobuf())
	}

	return messages, nil
}

// CreateConversation stores a conversation into rdb
func (u *ucs) CreateConversation(ctx context.Context, conv chat.Conversation) error {
	c := conv.GetRaw()
	// Set conversation id for each participant
	for participantID := range c.Participants {
		// Get user conversation rdb key
		key := conversation.GetUserConversationsKey(participantID)
		// Convert raw conversation to base64
		b64Conv, err := c.EncodeBinary()
		if err != nil {
			return err
		}
		// Store conversation in a set having participant ID in the key
		if err := u.rdb.HSet(ctx, key, c.ID, b64Conv).Err(); err != nil {
			return err
		}
	}

	return nil
}

// SetActiveConversation sets the user active conversation in a redis set
func (u *ucs) SetActiveConversation(ctx context.Context, userID, conversationID string) error {
	return u.rdb.Set(ctx, conversation.GetActiveUserConversationIDKey(userID), conversationID, 0).Err()
}

// SetActiveConversation deletes the user active conversation from the redis set
func (u *ucs) DeleteActiveConversation(ctx context.Context, userID string) error {
	return u.rdb.GetDel(ctx, conversation.GetActiveUserConversationIDKey(userID)).Err()
}

// GetConversation retrieves the encoded conversation by id
func (u *ucs) GetConversation(ctx context.Context, userID, conversationID string) (*v1API.Conversation, error) {
	// Retrieve conversation
	conv, err := u.getConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}

	// Retrieve conversation last message
	messages, err := u.rdb.ZRevRange(ctx, conversation.GetConversationMessagesKey(conversationID), 0, 0).Result()
	if err != nil {
		return nil, err
	}
	// Conversation has no messages
	if len(messages) == 0 {
		return conv.ToProtobuf(), nil
	}

	// Decode last message
	m := &message.Message{}
	if err := m.DecodeBinary(messages[0]); err != nil {
		return nil, err
	}

	conv.LastMessage = m

	return conv.ToProtobuf(), nil
}

func (u *ucs) ReadConversationMessages(ctx context.Context, userID, conversationID string) (*v1API.Conversation, error) {
	// Retrieve conversation
	conv, err := u.getConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}

	// Set `unread messages count` to zero
	conv.UnreadMessagesCount = 0

	// Convert raw conversation to base64
	b64Conv, err := conv.EncodeBinary()
	if err != nil {
		return nil, err
	}

	// Update conversation
	key := conversation.GetUserConversationsKey(userID)
	if err := u.rdb.HSet(ctx, key, conv.ID, b64Conv).Err(); err != nil {
		return nil, err
	}

	return conv.ToProtobuf(), nil
}

func (u *ucs) getConversation(ctx context.Context, userID, conversationID string) (*conversation.Conversation, error) {
	c, err := u.rdb.HGet(ctx, conversation.GetUserConversationsKey(userID), conversationID).Result()
	if err != nil {
		return nil, err
	}

	conv := &conversation.Conversation{}
	if err := conv.DecodeBinary(c); err != nil {
		return nil, err
	}

	return conv, nil
}

// GetConversations retrieves the encoded conversation for the userID
func (u *ucs) GetConversations(ctx context.Context, userID string) ([]*v1API.Conversation, error) {
	c, err := u.rdb.HGetAll(ctx, conversation.GetUserConversationsKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	conversationsList := []*v1API.Conversation{}

	for _, conversationStr := range c {
		conv, err := u.decodeConversation(ctx, &decodeConversationParams{
			conversationStr: conversationStr,
		})
		if err != nil {
			return nil, err
		}

		if conv != nil {
			conversationsList = append(conversationsList, conv.ToProtobuf())
		}
	}

	return conversationsList, nil
}

type decodeConversationParams struct {
	conversationStr              string
	mustContainParticipantUserID string
}

func (u *ucs) decodeConversation(ctx context.Context, params *decodeConversationParams) (*conversation.Conversation, error) {
	if params == nil {
		return nil, fmt.Errorf("invalid nil parameters received")
	}

	conv := &conversation.Conversation{}
	if err := conv.DecodeBinary(params.conversationStr); err != nil {
		return nil, err
	}

	if params.mustContainParticipantUserID != "" {
		// Check if conversation has participant
		if _, ok := conv.Participants[params.mustContainParticipantUserID]; !ok {
			return nil, nil
		}
	}

	// Retrieve conversation last message
	messages, err := u.rdb.ZRevRange(ctx, conversation.GetConversationMessagesKey(conv.ID), 0, 0).Result()
	if err != nil {
		return nil, err
	}
	// Conversation has no messages
	if len(messages) == 0 {
		return conv, nil
	}

	// Decode last message
	m := &message.Message{}
	if err := m.DecodeBinary(messages[0]); err != nil {
		return nil, err
	}

	conv.LastMessage = m
	return conv, nil
}

// GetConversationsWithUser returns a list of conversations between the 2 users
func (u *ucs) GetConversationsWithUser(ctx context.Context, userID, targetUserID string) ([]*v1API.Conversation, error) {
	c, err := u.rdb.HGetAll(ctx, conversation.GetUserConversationsKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	conversationsList := []*v1API.Conversation{}

	for _, conversationStr := range c {
		conv, err := u.decodeConversation(ctx, &decodeConversationParams{
			conversationStr:              conversationStr,
			mustContainParticipantUserID: targetUserID,
		})
		if err != nil {
			return nil, err
		}

		if conv != nil {
			conversationsList = append(conversationsList, conv.ToProtobuf())
		}
	}

	return conversationsList, nil
}

func (u *ucs) GetConversationInactiveUsers(ctx context.Context, senderUserID, conversationID string) ([]string, error) {
	inactiveUsers := []string{}
	conv, err := u.getConversation(ctx, senderUserID, conversationID)
	if err != nil {
		fmt.Println("err getting  conversation")
		return nil, err
	}

	for _, p := range conv.Participants {
		activeConvKey := conversation.GetActiveUserConversationIDKey(p.UserID)
		activeConvID, err := u.rdb.Get(ctx, activeConvKey).Result()
		switch {
		case err == redis.Nil, activeConvID == "":
			// Key does not exist, or it's empty. User has no active conversations
			inactiveUsers = append(inactiveUsers, p.UserID)
			continue
		case err != nil:
			return nil, err
		}

		if activeConvID != conversationID {
			// User is inactive
			inactiveUsers = append(inactiveUsers, p.UserID)
		}

	}

	return inactiveUsers, nil
}

func (u *ucs) IncrementInactiveUserMsgsCount(ctx context.Context, userID, conversationID string) error {
	conv, err := u.getConversation(ctx, userID, conversationID)
	if err != nil {
		fmt.Println("err get conv")
		return err
	}

	// Increment unread messages count
	conv.UnreadMessagesCount++

	// TODO: repeated like createconversation
	// Get user conversation rdb key
	key := conversation.GetUserConversationsKey(userID)
	// Convert raw conversation to base64
	b64Conv, err := conv.EncodeBinary()
	if err != nil {
		fmt.Println("err encoding conv")
		return err
	}
	// Store conversation in a set having participant ID in the key
	if err := u.rdb.HSet(ctx, key, conv.ID, b64Conv).Err(); err != nil {
		fmt.Println("err setting conv")
		return err
	}

	return nil
}

// TODO: conversations stored in one place
// Users have a list of joined conversations which is map[conversationID]{JoinedAt: time.Time, unreadMessagesCount: number}
// TODO: reduce number of redis operations (sendMessage get conversation)
// TODO: exclude own user from user suggestions
