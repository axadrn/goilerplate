package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"{{.ModulePath}}/internal/repository"
	"{{.ModulePath}}/internal/service"
)

func TestLogoutPreservesCookieWhenRevocationFails(t *testing.T) {
	sessionRepository := &failingLogoutSessionRepository{}
	authService := service.NewAuthService(
		nil, nil, sessionRepository, nil, nil,
		false, time.Hour, time.Hour, time.Hour, time.Hour, time.Hour,
	)
	handler := &authHandler{authService: authService}
	request := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	request.AddCookie(&http.Cookie{Name: "session_token", Value: "valid-token"})
	recorder := httptest.NewRecorder()

	handler.Logout(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatal("logout cleared the cookie after revocation failed")
	}
}

type failingLogoutSessionRepository struct {
	repository.SessionRepository
}

func (r *failingLogoutSessionRepository) DeleteByTokenHash(string) error {
	return errors.New("temporary database failure")
}
