package model

import "testing"

func TestUserKeepsPasswordStateAfterHashIsRemoved(t *testing.T) {
	hash := "password-hash"
	user := &User{PasswordHash: &hash}
	user.PasswordSet = user.HasPassword()
	user.PasswordHash = nil

	if !user.HasPassword() {
		t.Fatal("HasPassword() = false after password hash was removed from request context")
	}
}
