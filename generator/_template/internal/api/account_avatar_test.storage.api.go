package api_test

import (
	"bytes"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"{{.ModulePath}}/internal/api"
)

func multipartBody(t *testing.T, field, filename string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	_, err = part.Write(content)
	if err != nil {
		t.Fatalf("write form file: %v", err)
	}
	err = writer.Close()
	if err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	buffer := &bytes.Buffer{}
	err := png.Encode(buffer, image.NewRGBA(image.Rect(0, 0, 2, 2)))
	if err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buffer.Bytes()
}

func (e *apiEnvironment) upload(t *testing.T, token, field, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := multipartBody(t, field, filename, content)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/account/avatar", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	e.handler.ServeHTTP(recorder, request)
	return recorder
}

func TestUploadAndDeleteAvatar(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "avatar", "Avatar")
	token := environment.signIn(t, user.ID)

	recorder := environment.upload(t, token, "file", "me.png", pngBytes(t))
	expectStatus(t, recorder, http.StatusOK)
	uploaded := decodeBody[struct {
		AvatarURL *string `json:"avatar_url"`
	}](t, recorder)
	if uploaded.AvatarURL == nil || *uploaded.AvatarURL == "" {
		t.Fatalf("avatar_url = %v", uploaded.AvatarURL)
	}
	account := decodeBody[map[string]any](t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: token}))
	if account["avatar_url"] != *uploaded.AvatarURL {
		t.Fatalf("account avatar_url = %#v, want %s", account["avatar_url"], *uploaded.AvatarURL)
	}

	replaced := environment.upload(t, token, "file", "again.png", pngBytes(t))
	expectStatus(t, replaced, http.StatusOK)
	files, err := environment.files.AllUserFiles(user.ID)
	if err != nil || len(files) != 1 {
		t.Fatalf("files = %d, err = %v, want the replaced avatar only", len(files), err)
	}

	deleted := environment.send(t, apiRequest{method: http.MethodDelete, path: "/api/v1/account/avatar", bearer: token})
	expectStatus(t, deleted, http.StatusNoContent)
	account = decodeBody[map[string]any](t, environment.send(t, apiRequest{method: http.MethodGet, path: "/api/v1/account", bearer: token}))
	if account["avatar_url"] != nil {
		t.Fatalf("avatar_url = %#v after delete", account["avatar_url"])
	}
}

func TestUploadAvatarValidation(t *testing.T) {
	environment := newAPIEnvironment(t)
	user := seedVerifiedUser(t, environment, "invalid", "Invalid")
	token := environment.signIn(t, user.ID)

	notImage := environment.upload(t, token, "file", "notes.png", []byte("plain text, not an image"))
	detail := expectError(t, notImage, http.StatusUnprocessableEntity, "validation_failed")
	if detail.Fields["file"] == "" {
		t.Fatalf("fields = %#v", detail.Fields)
	}
	wrongField := environment.upload(t, token, "avatar", "me.png", pngBytes(t))
	detail = expectError(t, wrongField, http.StatusUnprocessableEntity, "validation_failed")
	if detail.Fields["file"] != "Choose an image to upload." {
		t.Fatalf("fields = %#v", detail.Fields)
	}
	jsonBody := environment.send(t, apiRequest{method: http.MethodPost, path: "/api/v1/account/avatar", body: `{}`, bearer: token})
	expectError(t, jsonBody, http.StatusBadRequest, "invalid_body")
	tooLarge := environment.upload(t, token, "file", "huge.png", make([]byte, api.MaxAvatarBytes+1))
	expectError(t, tooLarge, http.StatusBadRequest, "invalid_body")
}
