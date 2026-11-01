package handler

import "{{.ModulePath}}/internal/service"

type AccountHandler struct {
	authService *service.AuthService
	userService *service.UserService
	fileService *service.FileService
}

func NewAccountHandler(authService *service.AuthService, userService *service.UserService, fileService *service.FileService) *AccountHandler {
	return &AccountHandler{authService: authService, userService: userService, fileService: fileService}
}
