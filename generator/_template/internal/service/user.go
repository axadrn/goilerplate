package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/validation"
)

var ErrInvalidCurrentPassword = errors.New("current password is incorrect")

func (s *UserService) ByID(id string) (*model.User, error) {
	user, err := s.userRepository.ByID(id)
	if err != nil {
		return nil, err
	}

	return s.decorateUser(user)
}

func (s *UserService) UpdatePassword(userID, currentPassword, newPassword string) error {
	user, err := s.userRepository.ByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if !user.HasPassword() {
		return fmt.Errorf("passwordless accounts cannot update password. Please set a password first")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(currentPassword))
	if err != nil {
		return ErrInvalidCurrentPassword
	}

	// Validate new password
	err = validation.ValidatePassword(newPassword)
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	hashStr := string(hashedPassword)
	user.PasswordHash = &hashStr

	err = s.userRepository.Update(user)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

func (s *UserService) UpdateName(userID, name string) (*model.User, error) {
	if err := validation.ValidateName(name); err != nil {
		return nil, err
	}
	if err := s.userRepository.UpdateName(userID, strings.TrimSpace(name), time.Now()); err != nil {
		return nil, err
	}
	return s.ByID(userID)
}

func (s *UserService) DeleteAccount(userID string) error {
	if err := s.validateAccountDeletion(userID); err != nil {
		return err
	}

	user, err := s.userRepository.ByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	name := "User"
	if user.Name != "" {
		name = user.Name
	}

	return s.finishAccountDeletion(user, name)
}
