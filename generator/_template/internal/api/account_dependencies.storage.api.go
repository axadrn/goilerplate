package api

import "{{.ModulePath}}/internal/service"

// AccountHandler serves profile, email, password, avatar and account deletion.
type AccountHandler struct {
	authService *service.AuthService
	userService *service.UserService
	fileService *service.FileService
}

func NewAccountHandler(authService *service.AuthService, userService *service.UserService, fileService *service.FileService) *AccountHandler {
	return &AccountHandler{authService: authService, userService: userService, fileService: fileService}
}
