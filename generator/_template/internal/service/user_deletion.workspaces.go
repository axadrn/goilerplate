package service

import (
	"errors"
	"fmt"
	"log/slog"

	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
)

var ErrSoleWorkspaceOwner = errors.New("cannot delete account while sole owner of a workspace")

func (s *UserService) validateAccountDeletion(string) error {
	return nil
}

func (s *UserService) finishAccountDeletion(user *model.User, name string) error {
	removeFiles := s.userFileCleanup(user.ID)
	if err := s.workspaceService.DeleteUserIfOwnershipSafe(user.ID); err != nil {
		if errors.Is(err, repository.ErrLastOwner) {
			return ErrSoleWorkspaceOwner
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}
	removeFiles()
	if err := s.emailService.SendAccountDeletedEmail(user.Email, name); err != nil {
		slog.Warn("failed to send account deleted email", "user_id", user.ID, "error", err)
	}
	return nil
}
