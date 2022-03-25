package notifications_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DagDigg/unpaper/backend/notifications"
	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateNotification(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	ws := v1Testing.GetWrappedServer(t)
	dir := notifications.NewDirectory(ws.Server.GetDB())
	// Add user since notifications returns the sender username
	usr, err := ws.AddUser(v1Testing.GetRandomPGUserParams())
	assert.Nil(err)

	t.Run("When successfully creating a notification", func(t *testing.T) {
		n, err := dir.CreateNotification(context.Background(), notifications.CreateNotificationParams{
			ID:                  uuid.NewString(),
			UserIDToNotify:      uuid.NewString(),
			UserIDWhoFiredEvent: usr.Id,
			TriggerID:           sql.NullString{String: uuid.NewString(), Valid: true},
			EventID:             string(notifications.EventIDComment),
			Date:                time.Now(),
		})
		assert.Nil(err)

		assert.Equal(usr.Username, n.UserWhoFiredEvent.Username)
	})
}
