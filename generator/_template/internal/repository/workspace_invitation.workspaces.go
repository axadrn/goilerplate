package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/access"
	"{{.ModulePath}}/internal/model"
)

var (
	ErrWorkspaceInvitationNotFound = errors.New("invitation not found")
	ErrWorkspaceInvitationExpired  = errors.New("invitation expired")
	ErrWorkspaceInvitationRevoked  = errors.New("invitation revoked")
	ErrWorkspaceInvitationUsed     = errors.New("invitation already accepted")
	ErrWorkspaceInvitationEmail    = errors.New("invitation belongs to another email")
	ErrAlreadyMember               = errors.New("user is already a member")
	ErrSeatLimit                   = errors.New("workspace seat limit reached")
)

type WorkspaceInvitationRepository interface {
	Create(invitation *model.WorkspaceInvitation, event *model.WorkspaceAuditEvent) error
	ByIDForWorkspace(id, workspaceID string) (*model.WorkspaceInvitation, error)
	ByTokenHash(tokenHash string) (*model.WorkspaceInvitation, error)
	ListPendingByWorkspace(workspaceID string, now time.Time) ([]model.WorkspaceInvitation, error)
	Revoke(workspaceID, invitationID string, revokedAt time.Time, event *model.WorkspaceAuditEvent) error
	Accept(tokenHash, userID, email, memberID, auditID string, now time.Time) (*model.WorkspaceMember, error)
}

type workspaceInvitationRepository struct {
	db *sqlx.DB
}

func NewWorkspaceInvitationRepository(db *sqlx.DB) WorkspaceInvitationRepository {
	return &workspaceInvitationRepository{db: db}
}

func (r *workspaceInvitationRepository) Create(invitation *model.WorkspaceInvitation, event *model.WorkspaceAuditEvent) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	cutoff := invitation.CreatedAt.Add(-model.WorkspaceInvitationRetention)
	if _, err := tx.Exec(`
		DELETE FROM workspace_invitations
		WHERE (accepted_at IS NOT NULL AND accepted_at < $1)
		   OR (revoked_at IS NOT NULL AND revoked_at < $1)
		   OR expires_at < $1
	`, cutoff); err != nil {
		return err
	}

	if _, err := tx.Exec(`UPDATE workspaces SET updated_at = updated_at WHERE id = $1`, invitation.WorkspaceID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE workspace_invitations
		SET revoked_at = $1
		WHERE workspace_id = $2 AND email = $3 AND accepted_at IS NULL AND revoked_at IS NULL
	`, invitation.CreatedAt, invitation.WorkspaceID, invitation.Email); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO workspace_invitations (id, workspace_id, email, role, token_hash, invited_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, invitation.ID, invitation.WorkspaceID, invitation.Email, invitation.Role, invitation.TokenHash, invitation.InvitedBy, invitation.ExpiresAt, invitation.CreatedAt); err != nil {
		return err
	}
	if err := insertAuditEvent(tx, event); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *workspaceInvitationRepository) ByIDForWorkspace(id, workspaceID string) (*model.WorkspaceInvitation, error) {
	invitation := &model.WorkspaceInvitation{}
	err := r.db.Get(invitation, `SELECT * FROM workspace_invitations WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err == sql.ErrNoRows {
		return nil, ErrWorkspaceInvitationNotFound
	}
	return invitation, err
}

func (r *workspaceInvitationRepository) ByTokenHash(tokenHash string) (*model.WorkspaceInvitation, error) {
	invitation := &model.WorkspaceInvitation{}
	err := r.db.Get(invitation, `
		SELECT i.*, o.name AS workspace_name, o.slug AS workspace_slug, COALESCE(u.name, '') AS inviter_name
		FROM workspace_invitations i
		JOIN workspaces o ON o.id = i.workspace_id
		LEFT JOIN users u ON u.id = i.invited_by
		WHERE i.token_hash = $1
	`, tokenHash)
	if err == sql.ErrNoRows {
		return nil, ErrWorkspaceInvitationNotFound
	}
	return invitation, err
}

func (r *workspaceInvitationRepository) ListPendingByWorkspace(workspaceID string, now time.Time) ([]model.WorkspaceInvitation, error) {
	cutoff := now.Add(-model.WorkspaceInvitationRetention)
	if _, err := r.db.Exec(`
		DELETE FROM workspace_invitations
		WHERE (accepted_at IS NOT NULL AND accepted_at < $1)
		   OR (revoked_at IS NOT NULL AND revoked_at < $1)
		   OR expires_at < $1
	`, cutoff); err != nil {
		return nil, err
	}
	var invitations []model.WorkspaceInvitation
	err := r.db.Select(&invitations, `
		SELECT * FROM workspace_invitations
		WHERE workspace_id = $1 AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > $2
		ORDER BY created_at DESC, id DESC
	`, workspaceID, now)
	return invitations, err
}

func (r *workspaceInvitationRepository) Revoke(workspaceID, invitationID string, revokedAt time.Time, event *model.WorkspaceAuditEvent) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE workspaces SET updated_at = updated_at WHERE id = $1`, workspaceID); err != nil {
		return err
	}

	result, err := tx.Exec(`
		UPDATE workspace_invitations
		SET revoked_at = $1
		WHERE id = $2 AND workspace_id = $3 AND accepted_at IS NULL AND revoked_at IS NULL
	`, revokedAt, invitationID, workspaceID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWorkspaceInvitationNotFound
	}
	if err := insertAuditEvent(tx, event); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *workspaceInvitationRepository) Accept(
	tokenHash, userID, email, memberID, auditID string,
	now time.Time,
) (*model.WorkspaceMember, error) {
	var workspaceID string
	if err := r.db.Get(&workspaceID, `SELECT workspace_id FROM workspace_invitations WHERE token_hash = $1`, tokenHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrWorkspaceInvitationNotFound
		}
		return nil, err
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	result, err := tx.Exec(`UPDATE workspaces SET updated_at = updated_at WHERE id = $1`, workspaceID)
	if err != nil {
		return nil, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrWorkspaceNotFound
	}

	invitation := &model.WorkspaceInvitation{}
	if err := tx.Get(invitation, `SELECT * FROM workspace_invitations WHERE token_hash = $1`, tokenHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrWorkspaceInvitationNotFound
		}
		return nil, err
	}
	if invitation.AcceptedAt != nil {
		return nil, ErrWorkspaceInvitationUsed
	}
	if invitation.RevokedAt != nil {
		return nil, ErrWorkspaceInvitationRevoked
	}
	if !invitation.ExpiresAt.After(now) {
		return nil, ErrWorkspaceInvitationExpired
	}
	if !strings.EqualFold(invitation.Email, email) {
		return nil, ErrWorkspaceInvitationEmail
	}

	var existing int
	if err := tx.Get(&existing, `SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`, workspaceID, userID); err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, ErrAlreadyMember
	}

	var members int
	if err := tx.Get(&members, `SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1`, workspaceID); err != nil {
		return nil, err
	}
	subscription := &model.Subscription{}
	if err := tx.Get(subscription, `SELECT * FROM subscriptions WHERE workspace_id = $1`, workspaceID); err != nil {
		return nil, err
	}
	if err := subscription.AccessAt(now).RequireWithin(access.LimitMembers, members); err != nil {
		return nil, ErrSeatLimit
	}

	member := &model.WorkspaceMember{
		ID:          memberID,
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        invitation.Role,
		CreatedAt:   now,
	}
	if _, err := tx.Exec(`
		INSERT INTO workspace_members (id, workspace_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, member.ID, member.WorkspaceID, member.UserID, member.Role, member.CreatedAt); err != nil {
		return nil, err
	}
	accepted, err := tx.Exec(`
		UPDATE workspace_invitations
		SET accepted_at = $1
		WHERE id = $2 AND accepted_at IS NULL AND revoked_at IS NULL
	`, now, invitation.ID)
	if err != nil {
		return nil, err
	}
	acceptedRows, err := accepted.RowsAffected()
	if err != nil {
		return nil, err
	}
	if acceptedRows != 1 {
		return nil, ErrWorkspaceInvitationUsed
	}
	actorUserID := userID
	event := &model.WorkspaceAuditEvent{
		ID:          auditID,
		WorkspaceID: workspaceID,
		ActorUserID: &actorUserID,
		Action:      "workspace.invitation.accepted",
		TargetType:  "workspace_invitation",
		TargetID:    invitation.ID,
		Metadata:    "{}",
		CreatedAt:   now,
	}
	if err := insertAuditEvent(tx, event); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return member, nil
}

func insertAuditEvent(tx *sqlx.Tx, event *model.WorkspaceAuditEvent) error {
	_, err := tx.Exec(`
		INSERT INTO workspace_audit_events (id, workspace_id, actor_user_id, action, target_type, target_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.WorkspaceID, event.ActorUserID, event.Action, event.TargetType, event.TargetID, event.Metadata, event.CreatedAt)
	return err
}
