package posts

import (
	"fmt"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/dbentities"
)

func pgPostToPB(p Post) (*v1API.Post, error) {
	audio, err := dbentities.AudioRawJSONToPB(p.Audio)
	if err != nil {
		return nil, fmt.Errorf("error converting audio to PB: %v", err)
	}

	return &v1API.Post{
		Id:      p.ID,
		Message: p.Message,
		Author:  p.Author,
		Audio:   audio,
		Likes:   p.Likes.Int32,
	}, nil
}

func trendingTodayPostsListToPB(posts []GetTrendingTodayPostsRow) ([]*v1API.Post, error) {
	res := []*v1API.Post{}

	for _, p := range posts {
		pbPost, err := pgPostToPB(Post(p))
		if err != nil {
			return nil, err
		}
		res = append(res, pbPost)
	}

	return res, nil
}
