package api

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

// MaxBodyBytes caps every JSON request body.
const MaxBodyBytes = 1 << 20

// Decode reads one JSON object from the request body into value. A request
// whose Content-Type is not application/json gets a 415
// unsupported_media_type envelope, so a cross-site page cannot send a JSON
// body without a CORS preflight. Unknown fields, trailing data, and bodies
// over MaxBodyBytes are rejected with a 400 invalid_body envelope. On
// failure Decode writes the envelope and returns false.
func Decode(w http.ResponseWriter, r *http.Request, value any) bool {
	if !isJSONRequest(r) {
		Error(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send the request body as application/json.")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(value)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_body", decodeMessage(err))
		return false
	}
	err = decoder.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		Error(w, http.StatusBadRequest, "invalid_body", "The request body must contain a single JSON object.")
		return false
	}
	return true
}

func decodeMessage(err error) string {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return "The request body is too large."
	}
	if errors.Is(err, io.EOF) {
		return "The request body is empty."
	}
	return "The request body is not valid JSON for this endpoint."
}

func isJSONRequest(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "application/json"
}
