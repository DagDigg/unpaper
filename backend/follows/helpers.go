package follows

import (
	"database/sql"
	"time"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
)

func pgFollowUserRowToPB(f FollowUserRow) *v1API.ExtUserInfo {
	return &v1API.ExtUserInfo{
		Id:         f.ID,
		Email:      f.Email,
		GivenName:  f.GivenName.String,
		FamilyName: f.FamilyName.String,
		Username:   f.Username.String,
		IsFollowed: isFollowed(f.FollowDate, f.UnfollowDate),
	}
}

func pgUnfollowUserRowToPB(f UnfollowUserRow) *v1API.ExtUserInfo {
	return &v1API.ExtUserInfo{
		Id:         f.ID,
		Email:      f.Email,
		GivenName:  f.GivenName.String,
		FamilyName: f.FamilyName.String,
		Username:   f.Username.String,
		IsFollowed: isFollowed(f.FollowDate, f.UnfollowDate),
	}
}

func isFollowed(followDate time.Time, unfollowDate sql.NullTime) bool {
	if !unfollowDate.Valid {
		return true
	}

	return followDate.After(unfollowDate.Time)
}

func pgFollowersListToPB(f []GetFollowersRow) []*v1API.ExtUserInfo {
	res := []*v1API.ExtUserInfo{}
	for _, r := range f {
		res = append(res, pgFollowUserRowToPB(FollowUserRow(r)))
	}
	return res
}

func pgFollowingListToPB(f []GetFollowingRow) []*v1API.ExtUserInfo {
	res := []*v1API.ExtUserInfo{}
	for _, r := range f {
		res = append(res, pgFollowUserRowToPB(FollowUserRow(r)))
	}
	return res
}
