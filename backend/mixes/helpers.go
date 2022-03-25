package mixes

import (
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/dbentities"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func pgMixToPB(m Mix) (*v1API.Mix, error) {
	pgBackground, err := dbentities.UnmarshalBackgroundJSON(m.Background)
	if err != nil {
		return nil, err
	}
	return &v1API.Mix{
		Id:          m.ID,
		Title:       m.Title,
		Category:    m.Category,
		Background:  pgBackground,
		PostIds:     m.PostIds,
		RequestedAt: timestamppb.New(m.RequestedAt),
	}, nil
}

func pgMixListToPB(mixes []Mix) ([]*v1API.Mix, error) {
	res := []*v1API.Mix{}
	for _, m := range mixes {
		pbMix, err := pgMixToPB(m)
		if err != nil {
			return nil, err
		}
		res = append(res, pbMix)
	}

	return res, nil
}
