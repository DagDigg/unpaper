package message

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"time"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Message data structure representing
// a chat message. It can be in multiple forms: Text, Award, Donation...
type Message struct {
	// Base Message
	ID             string
	Type           MsgType
	UserID         string
	CreatedAt      time.Time
	SenderUsername string

	// Embedded message types
	Award
	Text
	Donation
	Audio
}

// MsgType denotes the type of message
type MsgType string

const (
	// TypeText is a text message type
	TypeText = MsgType("TEXT")
	// TypeAward is a award message type
	TypeAward = MsgType("AWARD")
	// TypeDonation is a donation message type
	TypeDonation = MsgType("DONATION")
	// TypeAudio is an audio message type
	TypeAudio = MsgType("AUDIO")
)

// ToProtobuf converts `Message` to its proto counterpart
func (m *Message) ToProtobuf() *v1API.ChatMessage {
	res := &v1API.ChatMessage{
		Id:        m.ID,
		UserId:    m.UserID,
		Username:  m.SenderUsername,
		CreatedAt: timestamppb.New(m.CreatedAt),
	}

	if m.Text.Content != "" {
		res.Type = v1API.MessageType_TEXT
		res.Text = &v1API.MessageText{Content: m.Content}
	}
	if m.Award.AwardID != "" {
		res.Type = v1API.MessageType_AWARD
		res.Award = &v1API.MessageAward{AwardId: m.AwardID}
	}
	if m.Donation.Amount != 0 {
		res.Type = v1API.MessageType_DONATION
		res.Donation = &v1API.MessageDonation{Amount: m.Donation.Amount}
	}
	if m.Audio.Bytes != nil {
		res.Type = v1API.MessageType_AUDIO
		res.Audio = &v1API.MessageAudio{Bytes: m.Bytes}
	}

	return res
}

// EncodeBinary encodes the *Message as base64 string
func (m *Message) EncodeBinary() (string, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	if err := e.Encode(m); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

// DecodeBinary decodes into *Message the raw base64 *Message string value
func (m *Message) DecodeBinary(str string) error {
	// Decode base64 string
	p, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return err
	}

	// Create a buffer and write the decoded base64 string bytes
	b := bytes.Buffer{}
	_, err = b.Write(p)
	if err != nil {
		return err
	}

	// Decode the buffer into *Message
	decoder := gob.NewDecoder(&b)
	return decoder.Decode(m)
}

// GetRaw returns the message as-is
func (m *Message) GetRaw() *Message {
	return m
}
