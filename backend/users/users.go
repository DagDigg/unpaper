package users

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/Masterminds/squirrel"

	// pgx is a postgres driver
	_ "github.com/jackc/pgx/v4/stdlib"
)

// Directory is the directory which operates on db table 'users'
type Directory struct {
	// querier is an interface containing all of the
	// directory methods. Must be created with users.New(db)
	querier Querier
	db      *sql.DB
	sb      squirrel.StatementBuilderType
}

// NewDirectory creates a new users directory
func NewDirectory(db *sql.DB) *Directory {
	return &Directory{db: db, querier: New(db)}
}

// Close closes Directory database connection
func (d Directory) Close() error {
	return d.db.Close()
}

// UserType denotes the kind of registered user accessing the APIs
type UserType string

const (
	// UserTypeMember for members
	UserTypeMember UserType = "member"
	// UserTypeCreator for creators
	UserTypeCreator UserType = "creator"
)

// CreateUser creates an user on db and returns its userID. Currently used only for testing
func (d Directory) CreateUser(ctx context.Context, user CreateUserParams) (*v1API.User, error) {
	pgUser, err := d.querier.CreateUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return userPostgresToProto(pgUser)
}

// GetUser retrieves the user from database looked up by the provided userID
func (d Directory) GetUser(ctx context.Context, userID string) (*v1API.User, error) {
	pgUser, err := d.querier.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return userPostgresToProto(pgUser)
}

// GetUserByUsername retrieves the user from database looked up by the provided username
func (d Directory) GetUserByUsername(ctx context.Context, username string) (*v1API.User, error) {
	pgUser, err := d.querier.GetUserByUsername(ctx, sql.NullString{String: username, Valid: true})
	if err != nil {
		return nil, err
	}
	return userPostgresToProto(pgUser)
}

// DeleteUser deletes an user by userID
func (d Directory) DeleteUser(ctx context.Context, userID string) error {
	ok, err := d.querier.UserIDExists(ctx, userID)
	if err != nil {
		return fmt.Errorf("error selecting userID from db: '%v'", err)
	}
	if !ok {
		return fmt.Errorf("user does not exist")
	}
	return d.querier.DeleteUser(ctx, userID)
}

// VerifyEmail updates the 'email_verified' column for the argument-passed userID
func (d Directory) VerifyEmail(ctx context.Context, userID string) error {
	return d.querier.VerifyEmail(ctx, userID)
}

// GetPassword selects the user password from db by userID
func (d Directory) GetPassword(ctx context.Context, userID string) (string, error) {
	pw, err := d.querier.GetPassword(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user password: '%v'", err)
	}
	if pw.Valid == false {
		// NULL password
		return "", fmt.Errorf("NULL user password")
	}
	return pw.String, nil
}

// GetEmail selects the user email from db by userID
func (d Directory) GetEmail(ctx context.Context, userID string) (string, error) {
	email, err := d.querier.GetEmail(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user password: '%v'", err)
	}
	if email == "" {
		return "", fmt.Errorf("email is empty")
	}

	return email, nil
}

// GetUserIDFromEmail retrieves the userID from its email
func (d Directory) GetUserIDFromEmail(ctx context.Context, email string) (string, error) {
	res, err := d.querier.GetUserIDFromEmail(ctx, email)
	if err != nil {
		return "", err
	}

	return res, nil
}

// EmailExists returns whether the email exists in the db
func (d Directory) EmailExists(ctx context.Context, email string) (bool, error) {
	res, err := d.querier.EmailExists(ctx, strings.ToLower(email))
	if err != nil {
		return false, err
	}

	return res, nil
}

// UsernameExists returns whether the username exists in the db
func (d Directory) UsernameExists(ctx context.Context, username string) (bool, error) {
	res, err := d.querier.UsernameExists(ctx, sql.NullString{String: strings.ToLower(username), Valid: len(username) > 0})
	if err != nil {
		return false, err
	}

	return res, err
}

// UpdatePassword updates the 'password' column for the user
func (d Directory) UpdatePassword(ctx context.Context, userID, password string) error {
	pgPass := sql.NullString{String: password, Valid: true}
	err := d.querier.UpdatePassword(ctx, UpdatePasswordParams{ID: userID, Password: pgPass})
	if err != nil {
		return err
	}

	// Update password_changed_at with current time
	return d.querier.UpdatePasswordChangedAt(ctx, UpdatePasswordChangedAtParams{
		PasswordChangedAt: sql.NullTime{Time: time.Now(), Valid: true},
		ID:                userID,
	})
}

// GetEmailVerified returns the email_verified column value for the userID
func (d Directory) GetEmailVerified(ctx context.Context, userID string) (bool, error) {
	res, err := d.querier.GetEmailVerified(ctx, userID)
	if err != nil {
		return false, err
	}

	return res.Bool, nil
}

// UpdateUsername updates the username column
func (d Directory) UpdateUsername(ctx context.Context, params *UpdateUsernameParams) (*v1API.User, error) {
	res, err := d.querier.UpdateUsername(ctx, *params)
	if err != nil {
		return nil, err
	}

	return userPostgresToProto(res)
}

// GetUserSuggestions returns a list of username suggestions given a LIKE input
func (d Directory) GetUserSuggestions(ctx context.Context, query string) ([]*v1API.UserSuggestion, error) {
	res, err := d.querier.GetUserSuggestions(ctx, query+"%")
	if err != nil {
		return nil, err
	}

	var users []*v1API.UserSuggestion
	for _, v := range res {
		user := &v1API.UserSuggestion{
			Id:       v.ID,
			Username: v.Username.String,
		}
		users = append(users, user)
	}

	return users, nil
}

func userPostgresToProto(u User) (*v1API.User, error) {
	userType, err := pgUserTypeToProto(UserType(u.Type))
	if err != nil {
		return nil, err
	}
	return &v1API.User{
		Api:           "v1",
		Id:            u.ID,
		Email:         u.Email,
		GivenName:     u.GivenName.String,
		FamilyName:    u.FamilyName.String,
		EmailVerified: u.EmailVerified.Bool,
		Type:          userType,
		Username:      u.Username.String,
	}, nil
}

func pgUserTypeToProto(t UserType) (v1API.UserType, error) {
	switch t {
	case UserTypeCreator:
		return v1API.UserType_CREATOR, nil
	case UserTypeMember:
		return v1API.UserType_MEMBER, nil
	default:
		return 0, fmt.Errorf("unexpected userType: '%v'", t)
	}
}
