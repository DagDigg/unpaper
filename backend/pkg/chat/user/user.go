package user

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
)

type User struct {
	ID       string
	Username string
	ImageURL string
}

func (u *User) ToProtobuf() *v1API.ChatUser {
	return &v1API.ChatUser{
		Id:       u.ID,
		Username: u.Username,
		ImageUrl: u.ImageURL,
	}
}

func (u *User) EncodeBinary() (string, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	if err := e.Encode(u); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

func (u *User) DecodeBinary(str string) error {
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

	// Decode into &User{}
	d := gob.NewDecoder(&b)
	return d.Decode(u)
}

func (u *User) GetRaw() *User {
	return u
}
