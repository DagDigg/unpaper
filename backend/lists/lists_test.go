package lists_test

import (
	"context"
	"testing"

	"github.com/DagDigg/unpaper/backend/lists"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/dbentities"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
)

func TestCreateList(t *testing.T) {
	t.Parallel()
	dir := getListsDir(t)
	assert := assert.New(t)

	t.Run("When creating a list", func(t *testing.T) {
		t.Parallel()
		l := dbentities.NewListFromMap(map[string]string{"idOne": "usernameOne", "idTwo": "usernameTwo"})

		byteData, err := l.ToRawJSON()
		assert.Nil(err)
		params := &lists.CreateListParams{
			ID:           uuid.NewString(),
			Name:         "MyName",
			AllowedUsers: byteData,
		}

		res, err := dir.CreateList(context.Background(), params)
		assert.Nil(err)

		assert.Equal(res.Id, params.ID)
		assert.Equal(res.Name, params.Name)
		assert.Equal(res.OwnerUserId, params.OwnerUserID)
		assert.Equal(res.AllowedUsers, l.GetMap())
	})

	t.Run("When creating a list with no users", func(t *testing.T) {
		t.Parallel()
		l := dbentities.NewList()
		byteData, err := l.ToRawJSON()
		assert.Nil(err)
		params := &lists.CreateListParams{
			ID:           uuid.NewString(),
			Name:         "MyName",
			AllowedUsers: byteData,
		}

		res, err := dir.CreateList(context.Background(), params)
		assert.Nil(err)

		assert.Equal(res.Id, params.ID)
		assert.Equal(res.Name, params.Name)
		assert.Equal(res.OwnerUserId, params.OwnerUserID)
		assert.Equal(res.AllowedUsers, l.GetMap())
	})
}

func TestUpdateList(t *testing.T) {
	t.Parallel()
	dir := getListsDir(t)
	assert := assert.New(t)

	t.Run("When updating allowed user ids", func(t *testing.T) {
		// Insert list
		l := dbentities.NewList()
		by, err := l.ToRawJSON()
		assert.Nil(err)
		roomID := uuid.NewString()
		p := &lists.CreateListParams{
			ID:           roomID,
			Name:         "MyName",
			AllowedUsers: by,
		}

		_, err = dir.CreateList(context.Background(), p)
		assert.Nil(err)

		l = dbentities.NewListFromMap(map[string]string{"idOne": "usernameOne", "idTwo": "usernameTwo"})
		newByteData, err := l.ToRawJSON()
		assert.Nil(err)
		updateParams := &lists.UpdateAllowedUsersParams{
			ID:           roomID,
			AllowedUsers: newByteData,
		}
		res, err := dir.UpdateAllowedUsers(context.Background(), updateParams)
		assert.Nil(err)

		assert.Equal(res.AllowedUsers, l.GetMap())
	})
}

func TestGetAllLists(t *testing.T) {
	t.Parallel()
	dir := getListsDir(t)
	assert := assert.New(t)

	t.Run("When getting multiple lists", func(t *testing.T) {
		listsByID := make(map[string]struct{})
		listsToAdd := 10
		ownerUserID := uuid.NewString()

		// Insert n lists
		for i := 0; i < listsToAdd; i++ {
			l := insertList(dir, ownerUserID, t)
			listsByID[l.Id] = struct{}{}
		}

		// Get All lists
		res, err := dir.GetListsByOwnerID(context.Background(), ownerUserID)
		assert.Nil(err)

		// Assert lists
		for _, l := range res {
			_, ok := listsByID[l.Id]
			assert.True(ok)
			delete(listsByID, l.Id)
		}
		assert.Equal(len(listsByID), 0)
	})
}

// insertList creates a list with empty allowed_users and returns
// the protobuf list
func insertList(dir *lists.Directory, ownerUserID string, t *testing.T) *v1API.List {
	l := dbentities.NewList()
	by, err := l.ToRawJSON()
	if err != nil {
		t.Errorf("error marshaling list: %q", err)
	}
	roomID := uuid.NewString()
	p := &lists.CreateListParams{
		ID:           roomID,
		Name:         "MyName",
		AllowedUsers: by,
		OwnerUserID:  ownerUserID,
	}

	res, err := dir.CreateList(context.Background(), p)
	if err != nil {
		t.Errorf("error creating list: %v", err)
	}

	return res
}

func getListsDir(t *testing.T) *lists.Directory {
	ws := v1Testing.GetWrappedServer(t)
	dir := lists.NewDirectory(ws.Server.GetDB())

	return dir
}
