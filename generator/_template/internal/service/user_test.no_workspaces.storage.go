package service

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/testutil"
)

func TestUpdatePasswordRequiresCurrentPasswordAndPersistsNewHash(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{
		ID:       "password-change-user",
		Password: validTestPassword,
		Verified: true,
	})
	userRepository := repository.NewUserRepository(database)
	userService := NewUserService(userRepository, nil, nil, nil)
	newPassword := "Z7!mP4#qR9@vL2$x"

	if err := userService.UpdatePassword(fixture.User.ID, "wrong password", newPassword); !errors.Is(err, ErrInvalidCurrentPassword) {
		t.Fatalf("wrong current password error = %v, want ErrInvalidCurrentPassword", err)
	}
	if err := userService.UpdatePassword(fixture.User.ID, validTestPassword, "short"); err == nil {
		t.Fatal("UpdatePassword() accepted a weak password")
	}
	if err := userService.UpdatePassword(fixture.User.ID, validTestPassword, newPassword); err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}

	updated, err := userRepository.ByID(fixture.User.ID)
	if err != nil {
		t.Fatalf("updated user lookup: %v", err)
	}
	if updated.PasswordHash == nil {
		t.Fatal("updated password hash is nil")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*updated.PasswordHash), []byte(newPassword)); err != nil {
		t.Fatalf("new password does not match stored hash: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*updated.PasswordHash), []byte(validTestPassword)); err == nil {
		t.Fatal("old password still matches stored hash")
	}
}

func TestUpdatePasswordRejectsPasswordlessAccount(t *testing.T) {
	database := testutil.NewDatabase(t)
	fixture := testutil.SeedUser(t, database, testutil.UserOptions{ID: "passwordless-user", Verified: true})
	userService := NewUserService(repository.NewUserRepository(database), nil, nil, nil)

	if err := userService.UpdatePassword(fixture.User.ID, "anything", validTestPassword); err == nil {
		t.Fatal("UpdatePassword() accepted a passwordless account")
	}
}
