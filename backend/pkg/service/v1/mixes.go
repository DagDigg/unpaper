package v1

import (
	"context"
	"database/sql"
	"math/rand"
	"time"

	"github.com/DagDigg/unpaper/backend/mixes"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/dbentities"
	"github.com/DagDigg/unpaper/backend/pkg/mdutils"
	"github.com/DagDigg/unpaper/backend/posts"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FakeBackgroundImages fake bgs
var FakeBackgroundImages = []string{"linear-gradient(to top, #a18cd1 0%, #fbc2eb 100%);", "linear-gradient(45deg, #ff9a9e 0%, #fad0c4 99%, #fad0c4 100%);", "linear-gradient(to top, #ff9a9e 0%, #fecfef 99%, #fecfef 100%);", "linear-gradient(to top, #96fbc4 0%, #f9f586 100%);", "linear-gradient(to top, #cd9cf2 0%, #f6f3ff 100%);", "linear-gradient(to top, #accbee 0%, #e7f0fd 100%);", "linear-gradient(to top, #accbee 0%, #e7f0fd 100%);"}

func (s *unpaperServiceServer) GetMixes(ctx context.Context, req *empty.Empty) (*v1API.GetMixesRes, error) {
	userID, ok := mdutils.GetUserIDFromMD(ctx)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "failed to retrieve userID from metadata")
	}
	mixesDir := mixes.NewDirectory(s.db)
	currMixes, err := mixesDir.GetUserMixes(ctx, userID)
	if err != nil {
		if err != sql.ErrNoRows {
			return nil, status.Errorf(codes.Internal, "failed to retrieve user mixes: %v", err)
		}
		// Mixes do not exists for user
		return createAndReturnGetMixesRes(ctx, s.db, userID)
	}

	// Create mixes if they do not exist or they're "expired"
	if len(currMixes) == 0 {
		return createAndReturnGetMixesRes(ctx, s.db, userID)
	}

	// if !sameDay(currMixes[0].RequestedAt.AsTime().UTC(), time.Now().UTC()) {
	// 	return createAndReturnGetMixesRes(ctx, s.db, userID)
	// }
	if true { // TODO: Remove
		return createAndReturnGetMixesRes(ctx, s.db, userID)
	}

	// Mixes exists, and 24hrs has not been passed
	return &v1API.GetMixesRes{
		Mixes: currMixes,
	}, nil
}

// Returns whether two dates are on the same day
func sameDay(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func createAndReturnGetMixesRes(ctx context.Context, db *sql.DB, userID string) (*v1API.GetMixesRes, error) {
	mixesDir := mixes.NewDirectory(db)
	if err := mixesDir.DeleteUserMixes(ctx, userID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete mixes: %v", err)
	}

	mixes, err := createMixes(ctx, db, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create mixes: %v", err)
	}

	return &v1API.GetMixesRes{
		Mixes: mixes,
	}, nil
}

func createMixes(ctx context.Context, db *sql.DB, userID string) ([]*v1API.Mix, error) {
	categories := []string{"Mindfulness", "Art", "Sports"}
	postsDir := posts.NewDirectory(db)
	mixesDir := mixes.NewDirectory(db)

	res := []*v1API.Mix{}

	// Iterate for each category, then create a mix from the trending today topic for that category
	for _, c := range categories {
		// TODO: Posts should have categories (?) and query trending posts should be made with category
		postIDs, err := postsDir.GetTrendingTodayPostIDs(ctx)
		if err != nil {
			return nil, err
		}
		if len(postIDs) == 0 {
			// If no posts are found, no mixes can be created
			continue
		}
		JSONBackground, err := dbentities.NewBackgroundRawJSON(&v1API.Background{
			Fallback:        "#ffeb99",
			BackgroundImage: FakeBackgroundImages[rand.Intn(len(FakeBackgroundImages))],
		}) // TODO: Randomize
		if err != nil {
			return nil, err
		}
		mix, err := mixesDir.CreateUserMix(ctx, mixes.CreateUserMixParams{
			ID:          uuid.NewString(),
			Title:       c, // TODO: use something smarter
			UserID:      userID,
			PostIds:     postIDs,
			Background:  JSONBackground,
			RequestedAt: time.Now().UTC(),
			Category:    c,
		})
		if err != nil {
			return nil, err
		}

		res = append(res, mix)
	}

	return res, nil
}
