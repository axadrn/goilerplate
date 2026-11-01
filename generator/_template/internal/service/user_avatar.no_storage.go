package service

import "{{.ModulePath}}/internal/model"

func (s *UserService) decorateUser(user *model.User) (*model.User, error) {
	return user, nil
}

func (s *UserService) userFileCleanup(string) func() {
	return func() {}
}
