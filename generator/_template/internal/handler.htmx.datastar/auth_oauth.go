package handler

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
)

var (
	errOAuthEmailMissing    = errors.New("oauth provider returned no email")
	errOAuthEmailUnverified = errors.New("oauth provider has not verified the email")
)

func generateOAuthState() string {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		panic("failed to generate oauth state: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(value)
}
