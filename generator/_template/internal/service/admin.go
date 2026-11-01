package service

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
)

const (
	adminPageSize = 25
)

var (
	ErrAdminPermission       = errors.New("admin permission required")
	ErrAdminSelfAction       = errors.New("admin cannot perform this action on the current account")
	ErrAdminTargetIsOperator = errors.New("operator accounts cannot be impersonated")
)

type AdminService struct {
	repository  repository.AdminRepository
	users       repository.UserRepository
	auth        *AuthService
	adminEmails map[string]struct{}
}

func NewAdminService(repo repository.AdminRepository, users repository.UserRepository, auth *AuthService, adminEmails []string) *AdminService {
	allowed := make(map[string]struct{}, len(adminEmails))
	for _, email := range adminEmails {
		allowed[strings.ToLower(strings.TrimSpace(email))] = struct{}{}
	}
	return &AdminService{repository: repo, users: users, auth: auth, adminEmails: allowed}
}

func (s *AdminService) IsAdmin(user *model.User, session *model.Session) bool {
	if user == nil || user.EmailVerifiedAt == nil || session == nil || session.IsImpersonating() {
		return false
	}
	_, ok := s.adminEmails[strings.ToLower(user.Email)]
	return ok
}

func (s *AdminService) Dashboard(user *model.User, session *model.Session, query string, page int) (*model.AdminDashboard, error) {
	if !s.IsAdmin(user, session) {
		return nil, ErrAdminPermission
	}
	if page < 1 {
		page = 1
	}
	now := time.Now()
	if err := s.repository.DeleteAuditEventsBefore(now.Add(-model.AdminAuditRetention)); err != nil {
		return nil, err
	}
	metrics, err := s.repository.Metrics(now)
	if err != nil {
		return nil, err
	}
	users, total, err := s.repository.ListUsers(query, adminPageSize, (page-1)*adminPageSize, now)
	if err != nil {
		return nil, err
	}
	for i := range users {
		_, users[i].IsAdmin = s.adminEmails[strings.ToLower(users[i].Email)]
	}
	totalPages := (total + adminPageSize - 1) / adminPageSize
	if totalPages == 0 {
		totalPages = 1
	}
	events, err := s.repository.ListAuditEvents(50)
	if err != nil {
		return nil, err
	}
	return &model.AdminDashboard{
		Metrics: metrics,
		Users: model.AdminUserPage{
			Users:      users,
			Query:      strings.TrimSpace(query),
			Page:       page,
			TotalPages: totalPages,
			TotalUsers: total,
		},
		Events: events,
	}, nil
}

func (s *AdminService) RevokeUserSessions(actor *model.User, actorSession *model.Session, targetUserID string) error {
	if !s.IsAdmin(actor, actorSession) {
		return ErrAdminPermission
	}
	if actor.ID == targetUserID {
		return ErrAdminSelfAction
	}
	if _, err := s.users.ByID(targetUserID); err != nil {
		return err
	}
	err := s.auth.RevokeAllAccess(targetUserID)
	if err != nil {
		return err
	}
	return s.recordAdminAction(actor.ID, "admin.sessions_revoked", "user", targetUserID)
}

func (s *AdminService) StartImpersonation(w http.ResponseWriter, r *http.Request, actor *model.User, actorSession *model.Session, targetUserID string) error {
	if !s.IsAdmin(actor, actorSession) {
		return ErrAdminPermission
	}
	if actor.ID == targetUserID {
		return ErrAdminSelfAction
	}
	target, err := s.users.ByID(targetUserID)
	if err != nil {
		return err
	}
	if _, isAdmin := s.adminEmails[strings.ToLower(target.Email)]; isAdmin {
		return ErrAdminTargetIsOperator
	}
	impersonationSession, err := s.auth.StartImpersonation(w, r, actor.ID, target.ID)
	if err != nil {
		return err
	}
	if err := s.recordAdminAction(actor.ID, "admin.impersonation_started", "user", target.ID); err != nil {
		if revokeErr := s.auth.RevokeSession(target.ID, impersonationSession.ID); revokeErr != nil {
			slog.Error("failed to roll back impersonation session", "error", revokeErr, "target_user_id", target.ID)
		}
		s.auth.ClearImpersonationCookie(w)
		return err
	}
	return nil
}

func (s *AdminService) StopImpersonation(w http.ResponseWriter, user *model.User, session *model.Session) error {
	if user == nil || session == nil || !session.IsImpersonating() {
		return ErrAdminPermission
	}
	actorUserID := *session.ImpersonatorUserID
	event := s.adminActionEvent(actorUserID, "admin.impersonation_stopped", "user", user.ID)
	if err := s.repository.StopImpersonationSession(session.ID, user.ID, actorUserID, event); err != nil {
		return err
	}
	s.auth.ClearImpersonationCookie(w)
	return nil
}

func (s *AdminService) recordAdminAction(actorUserID, action, targetType, targetID string) error {
	return s.repository.CreateAuditEvent(s.adminActionEvent(actorUserID, action, targetType, targetID))
}

func (s *AdminService) adminActionEvent(actorUserID, action, targetType, targetID string) *model.AdminAuditEvent {
	return &model.AdminAuditEvent{
		ID:          uuid.New().String(),
		ActorUserID: &actorUserID,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		Metadata:    "{}",
		CreatedAt:   time.Now(),
	}
}
