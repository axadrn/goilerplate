package service

import "{{.ModulePath}}/internal/repository"

type UserService struct {
	userRepository   repository.UserRepository
	emailService     *EmailService
	workspaceService *WorkspaceService
}

func NewUserService(users repository.UserRepository, email *EmailService, workspaces *WorkspaceService) *UserService {
	return &UserService{userRepository: users, emailService: email, workspaceService: workspaces}
}
