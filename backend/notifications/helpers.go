package notifications

import (
	"fmt"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func pgNotificationToPB(n CreateNotificationRow) (*v1API.Notification, error) {
	eventID, err := pgEventIDToPB(EventID(n.EventID))
	if err != nil {
		return nil, err
	}
	eventText, err := getEventTextByID(EventID(n.EventID))
	if err != nil {
		return nil, err
	}

	return &v1API.Notification{
		Id:        n.ID,
		Date:      timestamppb.New(n.Date.UTC()),
		TriggerId: n.TriggerID.String,
		Event: &v1API.Event{
			Id:   eventID,
			Text: string(eventText),
		},
		UserWhoFiredEvent: &v1API.UserWhoFiredEvent{
			Id:       n.UserIDWhoFiredEvent,
			Username: n.Username.String,
		},
		Content: n.Content.String,
		Read:    n.Read,
	}, nil
}

func pgUnreadNotificationsListToPB(ns []GetUnreadNotificationsRow) ([]*v1API.Notification, error) {
	var res []*v1API.Notification
	for _, n := range ns {
		pbNotification, err := pgNotificationToPB(CreateNotificationRow(n))
		if err != nil {
			return nil, err
		}
		res = append(res, pbNotification)
	}
	return res, nil
}

func pgGetAllNotificationsListToPB(ns []GetAllNotificationsRow) ([]*v1API.Notification, error) {
	var res []*v1API.Notification
	for _, n := range ns {
		pbNotification, err := pgNotificationToPB(CreateNotificationRow(n))
		if err != nil {
			return nil, err
		}
		res = append(res, pbNotification)
	}
	return res, nil
}

func pgEventIDToPB(e EventID) (v1API.EventID_Enum, error) {
	switch e {
	case EventIDLikePost:
		return v1API.EventID_LIKE_POST, nil
	case EventIDLikeComment:
		return v1API.EventID_LIKE_COMMENT, nil
	case EventIDComment:
		return v1API.EventID_COMMENT, nil
	case EventIDFollow:
		return v1API.EventID_FOLLOW, nil
	default:
		return 0, fmt.Errorf("invalid event id received: %v", e)
	}
}

func getEventTextByID(evtID EventID) (EventText, error) {
	switch evtID {
	case EventIDLikePost:
		return EventTextLikePost, nil
	case EventIDLikeComment:
		return EventTextLikeComment, nil
	case EventIDComment:
		return EventTextComment, nil
	case EventIDFollow:
		return EventTextFollow, nil
	default:
		return "", fmt.Errorf("invalid event id received: %v", evtID)
	}
}
