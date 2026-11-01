package repository

import (
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
)

type AdminRepository interface {
	Metrics(now time.Time) (model.AdminMetrics, error)
	ListUsers(query string, limit, offset int, now time.Time) ([]model.AdminUser, int, error)
	CreateAuditEvent(event *model.AdminAuditEvent) error
	StopImpersonationSession(sessionID, targetUserID, operatorUserID string, event *model.AdminAuditEvent) error
	ListAuditEvents(limit int) ([]model.AdminAuditEvent, error)
	DeleteAuditEventsBefore(cutoff time.Time) error
}

type adminRepository struct {
	db *sqlx.DB
}

func NewAdminRepository(db *sqlx.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) Metrics(now time.Time) (model.AdminMetrics, error) {
	var metrics model.AdminMetrics
	queries := []struct {
		target *int
		query  string
		args   []any
	}{
		{&metrics.Users, `SELECT COUNT(*) FROM users`, nil},
		{&metrics.VerifiedUsers, `SELECT COUNT(*) FROM users WHERE email_verified_at IS NOT NULL`, nil},
		{&metrics.Workspaces, `SELECT COUNT(*) FROM users`, nil},
		{&metrics.PaidWorkspaces, `
			SELECT COUNT(*) FROM subscriptions
			WHERE plan_id <> 'free'
			  AND (status IN ('active', 'trialing', 'past_due') OR (status = 'cancelled' AND current_period_end > $1))
		`, []any{now}},
		{&metrics.ActiveSessions, `SELECT COUNT(*) FROM sessions WHERE expires_at > $1`, []any{now}},
	}
	for _, item := range queries {
		if err := r.db.Get(item.target, item.query, item.args...); err != nil {
			return model.AdminMetrics{}, err
		}
	}
	return metrics, nil
}

func (r *adminRepository) ListUsers(query string, limit, offset int, now time.Time) ([]model.AdminUser, int, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	where := `WHERE ($1 = '%%' OR LOWER(u.email) LIKE $1 OR LOWER(u.name) LIKE $1)`

	var total int
	if err := r.db.Get(&total, `
		SELECT COUNT(*)
		FROM users u
		`+where, pattern); err != nil {
		return nil, 0, err
	}

	users := []model.AdminUser{}
	err := r.db.Select(&users, `
		SELECT
			u.id,
			u.email,
			u.name,
			u.email_verified_at,
			u.created_at,
			0 AS workspace_count,
			COUNT(DISTINCT CASE WHEN s.expires_at > $2 THEN s.id END) AS active_session_count
		FROM users u
		LEFT JOIN sessions s ON s.user_id = u.id
		`+where+`
		GROUP BY u.id, u.email, u.name, u.email_verified_at, u.created_at
		ORDER BY u.created_at DESC, u.id
		LIMIT $3 OFFSET $4
	`, pattern, now, limit, offset)
	return users, total, err
}

func (r *adminRepository) CreateAuditEvent(event *model.AdminAuditEvent) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM admin_audit_events WHERE created_at < $1`, event.CreatedAt.Add(-model.AdminAuditRetention)); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO admin_audit_events (
			id, actor_user_id, workspace_id, action, target_type, target_id, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, event.ID, event.ActorUserID, event.WorkspaceID, event.Action, event.TargetType, event.TargetID, event.Metadata, event.CreatedAt); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *adminRepository) StopImpersonationSession(sessionID, targetUserID, operatorUserID string, event *model.AdminAuditEvent) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		DELETE FROM sessions
		WHERE id = $1 AND user_id = $2 AND impersonator_user_id = $3
	`, sessionID, targetUserID, operatorUserID)
	if err != nil {
		return err
	}
	if err := requireSessionRow(result); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM admin_audit_events WHERE created_at < $1`, event.CreatedAt.Add(-model.AdminAuditRetention)); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO admin_audit_events (
			id, actor_user_id, workspace_id, action, target_type, target_id, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.ActorUserID, event.WorkspaceID, event.Action, event.TargetType, event.TargetID, event.Metadata, event.CreatedAt); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *adminRepository) ListAuditEvents(limit int) ([]model.AdminAuditEvent, error) {
	events := []model.AdminAuditEvent{}
	err := r.db.Select(&events, `
		SELECT e.*, COALESCE(u.email, '') AS actor_email
		FROM admin_audit_events e
		LEFT JOIN users u ON u.id = e.actor_user_id
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $1
	`, limit)
	return events, err
}

func (r *adminRepository) DeleteAuditEventsBefore(cutoff time.Time) error {
	_, err := r.db.Exec(`DELETE FROM admin_audit_events WHERE created_at < $1`, cutoff)
	return err
}
