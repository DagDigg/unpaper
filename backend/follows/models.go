// Code generated by sqlc. DO NOT EDIT.

package follows

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Comment struct {
	Likes           sql.NullInt32
	Audio           json.RawMessage
	Author          string
	ParentID        sql.NullString
	PostID          string
	ThreadType      string
	ID              string
	ThreadTargetID  sql.NullString
	Message         sql.NullString
	UserIdsWhoLikes []string
}

type ConnectedAccount struct {
	CanReceivePayments bool
	UserID             string
	CustomerID         string
	AccountID          string
}

type ConnectedCustomer struct {
	UserID              string
	CustomerID          string
	ConnectedCustomerID string
	AccountID           string
}

type Customer struct {
	TrialUsed  sql.NullBool
	ID         string
	CustomerID string
	FirstName  string
	LastName   string
	AccountID  sql.NullString
}

type Follow struct {
	FollowerUserID  string
	FollowingUserID string
	FollowDate      time.Time
	UnfollowDate    sql.NullTime
}

type List struct {
	AllowedUsers json.RawMessage
	ID           string
	Name         string
	OwnerUserID  string
}

type Mix struct {
	ID          string
	UserID      string
	Category    string
	PostIds     []string
	Background  json.RawMessage
	RequestedAt time.Time
	Title       string
}

type Notification struct {
	ID                  string
	UserIDToNotify      string
	UserIDWhoFiredEvent string
	Date                time.Time
	Read                bool
	TriggerID           sql.NullString
	EventID             string
	Content             sql.NullString
}

type Post struct {
	Likes           sql.NullInt32
	Audio           json.RawMessage
	ID              string
	Author          string
	Message         string
	UserIdsWhoLikes []string
	CreatedAt       sql.NullTime
}

type RoomSubscription struct {
	LatestInvoice        json.RawMessage
	CurrentPeriodEnd     sql.NullTime
	CustomerID           string
	ConnectedCustomerID  string
	AccountID            string
	ID                   string
	Status               string
	RoomID               string
	RoomSubscriptionType string
	UserID               string
}

type StripeDefaultPaymentMethod struct {
	ExpMonth   int32
	ExpYear    int32
	IsDefault  sql.NullBool
	ID         string
	LastFour   string
	UserID     string
	CustomerID string
}

type StripePrice struct {
	CustomerID string
	ID         string
	UserID     string
	Plan       string
	Active     bool
}

type StripeSubscription struct {
	CurrentPeriodEnd time.Time
	LatestInvoice    json.RawMessage
	ID               string
	UserID           string
	CustomerID       string
	Status           string
}

type User struct {
	EmailVerified     sql.NullBool
	PasswordChangedAt sql.NullTime
	Email             string
	Password          sql.NullString
	ID                string
	FamilyName        sql.NullString
	Type              string
	GivenName         sql.NullString
	Username          sql.NullString
}
