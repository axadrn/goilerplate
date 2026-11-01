package service

import "{{.ModulePath}}/internal/repository"

type UserService struct {
	userRepository      repository.UserRepository
	emailService        *EmailService
	subscriptionService *SubscriptionService
}

func NewUserService(users repository.UserRepository, email *EmailService, subscriptions *SubscriptionService) *UserService {
	return &UserService{userRepository: users, emailService: email, subscriptionService: subscriptions}
}
