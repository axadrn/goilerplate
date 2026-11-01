package api

import (
	"log/slog"
	"net/http"

	"{{.ModulePath}}/internal/ctxkeys"
	"{{.ModulePath}}/internal/validation"
)

// MaxAvatarBytes caps an avatar upload request.
const MaxAvatarBytes = 10 << 20

type avatarResponse struct {
	AvatarURL *string `json:"avatar_url"`
}

// UploadAvatar replaces the avatar with the multipart field "file".
func (h *AccountHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, MaxAvatarBytes)
	err := r.ParseMultipartForm(MaxAvatarBytes)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_body", "Send the avatar as multipart/form-data with a file field, at most 10 MB.")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		ValidationError(w, map[string]string{"file": "Choose an image to upload."})
		return
	}
	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			slog.Error("failed to close file", "error", closeErr)
		}
	}()

	err = validation.ValidateFile(header, validation.ImageConstraints)
	if err != nil {
		ValidationError(w, map[string]string{"file": FieldMessage(err)})
		return
	}

	err = h.fileService.DeleteUserAvatar(user.ID)
	if err != nil {
		slog.Warn("failed to delete old avatar", "error", err, "user_id", user.ID)
	}
	_, err = h.fileService.Upload(user.ID, "user", user.ID, "avatar", file, header, true)
	if err != nil {
		slog.Error("failed to upload avatar", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}

	updatedUser, err := h.userService.ByID(user.ID)
	if err != nil {
		ServiceError(w, err)
		return
	}
	JSON(w, http.StatusOK, avatarResponse{AvatarURL: avatarURL(updatedUser)})
}

// DeleteAvatar removes the avatar.
func (h *AccountHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	user := ctxkeys.User(r.Context())
	err := h.fileService.DeleteUserAvatar(user.ID)
	if err != nil {
		slog.Error("failed to delete avatar", "error", err, "user_id", user.ID)
		ServiceError(w, err)
		return
	}
	NoContent(w)
}
