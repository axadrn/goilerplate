package service

import (
	"log/slog"
	"{{.ModulePath}}/internal/model"
)

func (s *UserService) decorateUser(user *model.User) (*model.User, error) {
	avatar, err := s.fileService.Avatar("user", user.ID)
	if err == nil {
		user.AvatarURL = s.fileService.URL(avatar)
	}
	return user, nil
}

// userFileCleanup lists the files of a user before the account is deleted
// and returns a function that removes them from storage. Call it only after
// the deletion succeeded, so a refused deletion keeps every file.
func (s *UserService) userFileCleanup(userID string) func() {
	files, err := s.fileService.AllUserFiles(userID)
	if err != nil {
		slog.Warn("failed to list user files for deletion", "user_id", userID, "error", err)
		return func() {}
	}
	return func() {
		err := s.fileService.DeleteFilesFromStorage(files)
		if err != nil {
			slog.Warn("failed to delete user files from storage", "user_id", userID, "error", err)
		}
	}
}
