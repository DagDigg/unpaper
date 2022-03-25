package dbentities

import (
	"encoding/json"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
)

// List contains a map for storing data into db
type List struct {
	m map[string]string
}

// NewList creates a new list with a default empty map
func NewList() *List {
	m := make(map[string]string)
	return &List{m}
}

// NewListFromMap creates a *List from a given map
func NewListFromMap(m map[string]string) *List {
	if m == nil {
		return NewList()
	}
	return &List{m}
}

// ToRawJSON converts a list map to json raw data
func (l *List) ToRawJSON() ([]byte, error) {
	return json.Marshal(l.m)
}

// UnmarshalJSON unmarshals raw byte data into the list map
func (l *List) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.m)
}

// GetMap returns the underlying list map
func (l *List) GetMap() map[string]string {
	return l.m
}

// NewAudioRawJSON convers an API Audio to json.RawMessage
func NewAudioRawJSON(a *v1API.Audio) (json.RawMessage, error) {
	bytes, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(bytes), nil
}

// AudioRawJSONToPB unmarshal an audio json.RawMessage into a protobuf *Audio
func AudioRawJSONToPB(audio json.RawMessage) (*v1API.Audio, error) {
	a := &v1API.Audio{}
	if err := json.Unmarshal(audio, a); err != nil {
		return nil, err
	}
	return a, nil
}

func NewMixRawJSON(m *v1API.Mix) ([]byte, error) {
	return json.Marshal(m)
}

func UnmarshalMixesJSON(data []byte) ([]*v1API.Mix, error) {
	res := []*v1API.Mix{}
	if err := json.Unmarshal(data, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func NewBackgroundRawJSON(b *v1API.Background) ([]byte, error) {
	return json.Marshal(b)
}

func UnmarshalBackgroundJSON(data []byte) (*v1API.Background, error) {
	res := &v1API.Background{}
	if err := json.Unmarshal(data, res); err != nil {
		return nil, err
	}

	return res, nil
}
