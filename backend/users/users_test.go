package users_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	v1Testing "github.com/DagDigg/unpaper/backend/pkg/service/v1/testing"
	"github.com/DagDigg/unpaper/backend/users"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v4/stdlib"
)

func TestGetUser(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dir := getUsersDirectory(t)

	t.Cleanup(func() {
		err := dir.Close()
		if err != nil {
			t.Fatalf("error closing directory: %v", err)
		}
	})

	t.Run("When getting an user", func(t *testing.T) {
		t.Parallel()
		// Insert mock user into db
		userParams := v1Testing.GetRandomPGUserParams()
		pgUser, err := dir.CreateUser(ctx, userParams)
		if err != nil {
			t.Fatal(err)
		}

		// Get inserted user from db
		user, err := dir.GetUser(ctx, pgUser.Id)
		if err != nil {
			t.Fatal(err)
		}
		if user.Id != pgUser.Id {
			t.Errorf("userID mismatch: inserted: %q, selected: %q", userParams.ID, user.Id)
		}
		if user.Email != pgUser.Email {
			t.Errorf("email mismatch: inserted: %q, selected: %q", userParams.Email, user.Email)
		}
		if user.EmailVerified != pgUser.EmailVerified {
			t.Errorf("emailVerified mismatch: inserted: %q, selected: '%v'", "false", user.EmailVerified)
		}
		if user.Type != pgUser.Type {
			t.Errorf("userType mismatch: inserted: %q, selected: '%v'", "MEMBER", user.Type)
		}
	})
}

func TestVerifyEmail(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	dir := getUsersDirectory(t)

	t.Cleanup(func() {
		err := dir.Close()
		if err != nil {
			t.Fatalf("error closing directory: %v", err)
		}
	})

	t.Run("When verifying email", func(t *testing.T) {
		t.Parallel()
		// Insert mock user into db
		userParams := v1Testing.GetRandomPGUserParams()
		user, err := dir.CreateUser(ctx, userParams)
		if err != nil {
			t.Fatal(err)
		}
		if user.EmailVerified == true {
			t.Error("user created with email already verified")
		}

		err = dir.VerifyEmail(ctx, user.Id)
		if err != nil {
			t.Errorf("error verifying email: '%v'", err)
		}

		// Get inserted user from db
		dbUser, err := dir.GetUser(ctx, user.Id)
		if err != nil {
			t.Fatal(err)
		}
		if dbUser.EmailVerified == false {
			t.Error("email_verified has not been succesfully set to true")
		}
	})
}

func TestDeleteUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := getUsersDirectory(t)

	userParams := v1Testing.GetRandomPGUserParams()
	user, err := dir.CreateUser(ctx, userParams)
	if err != nil {
		t.Error(err)
	}

	t.Run("When user exists", func(t *testing.T) {
		t.Parallel()
		err = dir.DeleteUser(ctx, user.Id)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("When user does not exist", func(t *testing.T) {
		t.Parallel()
		err = dir.DeleteUser(ctx, "I do not exist")
		if err == nil {
			t.Error("expected error, received none")
		}
	})
}

func TestGetPassword(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := getUsersDirectory(t)

	t.Run("When user has password set", func(t *testing.T) {
		t.Parallel()
		// Insert mock user into db
		userParams := v1Testing.GetRandomPGUserParams()
		user, err := dir.CreateUser(ctx, userParams)
		if err != nil {
			t.Fatal(err)
		}

		psw, err := dir.GetPassword(ctx, user.Id)
		if err != nil {
			t.Errorf("error during getPassword: '%v'", err)
		}
		if psw == "" {
			t.Error("selected empty password")
		}
	})

	t.Run("When user has no password set", func(t *testing.T) {
		t.Parallel()
		// Insert mock user into db without password
		userParams := v1Testing.GetRandomPGUserParams()
		userParams.Password = sql.NullString{String: "", Valid: false}
		user, err := dir.CreateUser(ctx, userParams)
		if err != nil {
			t.Errorf("error storing user into db: %v", err)
		}

		// Retrieve password
		_, err = dir.GetPassword(ctx, user.Id)
		if err == nil {
			t.Fatal("expected error, got none")
		}
		wantErr := fmt.Errorf("NULL user password")
		if err.Error() != wantErr.Error() {
			t.Errorf("errors mismatch. got: '%v', want: '%v'", err, wantErr)
		}
	})
}

func TestUpdatePassword(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dir := getUsersDirectory(t)

	t.Run("When user has password", func(t *testing.T) {
		// Insert mock user into db
		// Insert mock user into db
		userParams := v1Testing.GetRandomPGUserParams()
		user, err := dir.CreateUser(ctx, userParams)
		if err != nil {
			t.Fatal(err)
		}

		newPass := "myCoolNewPass"
		err = dir.UpdatePassword(ctx, user.Id, newPass)
		if err != nil {
			t.Errorf("error occurred during password update: '%v'", err)
		}

		dbPass, err := dir.GetPassword(ctx, user.Id)
		if err != nil {
			t.Errorf("error occurred during getPassword: '%v'", err)
		}

		if dbPass != newPass {
			t.Errorf("password badly updated. got: '%v', want: '%v'", dbPass, newPass)
		}
	})
}

func TestGetUserSuggestions(t *testing.T) {
	ctx := context.Background()
	dir := getUsersDirectory(t)

	// Insert users
	usernames := []string{"Abc", "ABcD", "abD", "XXx", "az"}
	for _, u := range usernames {
		_, err := dir.CreateUser(ctx, users.CreateUserParams{
			ID:       uuid.NewString(),
			Email:    uuid.NewString(),
			Username: sql.NullString{String: u, Valid: true},
		})
		if err != nil {
			t.Errorf("error creating user: %v", err)
		}
	}

	t.Run("When getting suggestions", func(t *testing.T) {
		tests := []struct {
			query         string
			wantResLength int
		}{
			{
				query:         "Ab",
				wantResLength: 3,
			},
			{
				query:         "A",
				wantResLength: 4,
			},
			{
				query:         "x",
				wantResLength: 1,
			},
			{
				query:         "z",
				wantResLength: 0,
			},
		}

		for _, tt := range tests {
			res, err := dir.GetUserSuggestions(ctx, tt.query)
			if err != nil {
				t.Errorf("error calling GetUserSuggestions: %v", err)
			}

			if len(res) != tt.wantResLength {
				t.Errorf("unexpected users length with query: %v. got: %d, want: %d", tt.query, len(res), tt.wantResLength)
			}

		}
	})

}
func getUsersDirectory(t *testing.T) *users.Directory {
	ws := v1Testing.GetWrappedServer(t)
	return users.NewDirectory(ws.Server.GetDB())
}
