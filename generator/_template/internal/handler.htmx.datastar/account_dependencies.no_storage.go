package handler

import "{{.ModulePath}}/internal/service"

type AccountHandler struct {
	authService *service.AuthService
	userService *service.UserService
}

func NewAccountHandler(authService *service.AuthService, userService *service.UserService) *AccountHandler {
	return &AccountHandler{authService: authService, userService: userService}
}
