package service

// RevokeAllAccess ends every session of the user. Account recovery and the
// admin kill switch call it.
func (s *AuthService) RevokeAllAccess(userID string) error {
	return s.RevokeAllSessions(userID)
}

// RevokeOtherAccess ends every session of the user except currentSessionID.
// Changing, setting and removing the password call it.
func (s *AuthService) RevokeOtherAccess(userID, currentSessionID string) error {
	return s.RevokeOtherSessions(userID, currentSessionID)
}
