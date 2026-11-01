package service

import "{{.ModulePath}}/internal/repository"

type UserService struct {
	userRepository      repository.UserRepository
	fileService         *FileService
	emailService        *EmailService
	subscriptionService *SubscriptionService
}

func NewUserService(users repository.UserRepository, files *FileService, email *EmailService, subscriptions *SubscriptionService) *UserService {
	return &UserService{userRepository: users, fileService: files, emailService: email, subscriptionService: subscriptions}
}
