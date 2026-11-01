package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"

	"{{.ModulePath}}/internal/api"
)

// requestEmail returns the normalized email address of a form or JSON request
// so the recipient limits apply to both transports. A JSON body is read once
// and restored for the handler.
func requestEmail(r *http.Request) string {
	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType != "application/json" {
		return normalizeEmail(r.FormValue("email"))
	}
	if r.Body == nil {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, api.MaxBodyBytes+1))
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), r.Body))
	if err != nil {
		return ""
	}
	var payload struct {
		Email string `json:"email"`
	}
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return ""
	}
	return normalizeEmail(payload.Email)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
