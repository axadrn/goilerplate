package api

import "{{.ModulePath}}/internal/service"

// AccountHandler serves profile, email, password and account deletion.
type AccountHandler struct {
	authService *service.AuthService
	userService *service.UserService
}

func NewAccountHandler(authService *service.AuthService, userService *service.UserService) *AccountHandler {
	return &AccountHandler{authService: authService, userService: userService}
}
