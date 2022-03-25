package lists

import (
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/dbentities"
)

// pgListToPB converts postgres list to protobuf
func pgListToPB(pgList List) (*v1API.List, error) {
	l := dbentities.NewList()
	err := l.UnmarshalJSON(pgList.AllowedUsers)
	if err != nil {
		return nil, err
	}

	return &v1API.List{
		Id:           pgList.ID,
		Name:         pgList.Name,
		OwnerUserId:  pgList.OwnerUserID,
		AllowedUsers: l.GetMap(),
	}, nil
}
