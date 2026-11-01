package service

import (
	"fmt"
	"log/slog"

	"{{.ModulePath}}/internal/model"
)

func (s *UserService) finishAccountDeletion(user *model.User, name string) error {
	removeFiles := s.userFileCleanup(user.ID)
	if err := s.userRepository.Delete(user.ID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	removeFiles()
	if err := s.emailService.SendAccountDeletedEmail(user.Email, name); err != nil {
		slog.Warn("failed to send account deleted email", "user_id", user.ID, "error", err)
	}
	return nil
}
