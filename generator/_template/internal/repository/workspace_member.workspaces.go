package repository

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
)

var (
	ErrWorkspaceMemberNotFound   = errors.New("member not found")
	ErrWorkspaceMemberPermission = errors.New("member permission denied")
	ErrLastOwner                 = errors.New("workspace must keep at least one owner")
)

type WorkspaceMemberRepository interface {
	ByWorkspaceAndUser(workspaceID, userID string) (*model.WorkspaceMember, error)
	ByIDForWorkspace(id, workspaceID string) (*model.WorkspaceMember, error)
	ListByWorkspace(workspaceID string) ([]model.WorkspaceMember, error)
	CountByWorkspace(workspaceID string) (int, error)
	CountOwners(workspaceID string) (int, error)
	UpdateRole(workspaceID, actorUserID, memberID, role string, event *model.WorkspaceAuditEvent) error
	Delete(workspaceID, actorUserID, memberID string, event *model.WorkspaceAuditEvent) error
	DeleteUserIfOwnershipSafe(userID string) error
}

func (r *workspaceMemberRepository) ByIDForWorkspace(id, workspaceID string) (*model.WorkspaceMember, error) {
	member := &model.WorkspaceMember{}
	err := r.db.Get(member, `SELECT * FROM workspace_members WHERE id = $1 AND workspace_id = $2`, id, workspaceID)
	if err == sql.ErrNoRows {
		return nil, ErrWorkspaceMemberNotFound
	}
	return member, err
}

type workspaceMemberRepository struct {
	db *sqlx.DB
}

func NewWorkspaceMemberRepository(db *sqlx.DB) WorkspaceMemberRepository {
	return &workspaceMemberRepository{db: db}
}

func (r *workspaceMemberRepository) ByWorkspaceAndUser(workspaceID, userID string) (*model.WorkspaceMember, error) {
	member := &model.WorkspaceMember{}
	err := r.db.Get(member, `SELECT * FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`, workspaceID, userID)
	if err == sql.ErrNoRows {
		return nil, ErrWorkspaceMemberNotFound
	}
	return member, err
}

func (r *workspaceMemberRepository) ListByWorkspace(workspaceID string) ([]model.WorkspaceMember, error) {
	var members []model.WorkspaceMember
	err := r.db.Select(&members, `
		SELECT m.*, u.email, u.name
		FROM workspace_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.workspace_id = $1
		ORDER BY CASE m.role WHEN 'owner' THEN 1 WHEN 'admin' THEN 2 ELSE 3 END, u.name, u.email
	`, workspaceID)
	return members, err
}

func (r *workspaceMemberRepository) CountByWorkspace(workspaceID string) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1`, workspaceID)
	return count, err
}

func (r *workspaceMemberRepository) CountOwners(workspaceID string) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1 AND role = 'owner'`, workspaceID)
	return count, err
}

func (r *workspaceMemberRepository) UpdateRole(workspaceID, actorUserID, memberID, role string, event *model.WorkspaceAuditEvent) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE workspaces SET updated_at = updated_at WHERE id = $1`, workspaceID); err != nil {
		return err
	}
	actor := &model.WorkspaceMember{}
	if err := tx.Get(actor, `SELECT * FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`, workspaceID, actorUserID); err != nil {
		if err == sql.ErrNoRows {
			return ErrWorkspaceMemberPermission
		}
		return err
	}
	current := &model.WorkspaceMember{}
	if err := tx.Get(current, `SELECT * FROM workspace_members WHERE id = $1 AND workspace_id = $2`, memberID, workspaceID); err != nil {
		if err == sql.ErrNoRows {
			return ErrWorkspaceMemberNotFound
		}
		return err
	}
	if !actor.Can(model.WorkspacePermissionManageMembers) ||
		((current.Role == model.WorkspaceRoleOwner || role == model.WorkspaceRoleOwner) && !actor.Can(model.WorkspacePermissionManageOwners)) {
		return ErrWorkspaceMemberPermission
	}
	if current.Role == model.WorkspaceRoleOwner && role != model.WorkspaceRoleOwner {
		var owners int
		if err := tx.Get(&owners, `SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1 AND role = 'owner'`, workspaceID); err != nil {
			return err
		}
		if owners <= 1 {
			return ErrLastOwner
		}
	}
	result, err := tx.Exec(`
		UPDATE workspace_members
		SET role = $1
		WHERE id = $2 AND workspace_id = $3
	`, role, memberID, workspaceID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWorkspaceMemberNotFound
	}
	event.Metadata = auditMetadata(map[string]string{"from": current.Role, "to": role})
	if err := insertAuditEvent(tx, event); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *workspaceMemberRepository) Delete(workspaceID, actorUserID, memberID string, event *model.WorkspaceAuditEvent) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE workspaces SET updated_at = updated_at WHERE id = $1`, workspaceID); err != nil {
		return err
	}
	actor := &model.WorkspaceMember{}
	if err := tx.Get(actor, `SELECT * FROM workspace_members WHERE workspace_id = $1 AND user_id = $2`, workspaceID, actorUserID); err != nil {
		if err == sql.ErrNoRows {
			return ErrWorkspaceMemberPermission
		}
		return err
	}
	current := &model.WorkspaceMember{}
	if err := tx.Get(current, `SELECT * FROM workspace_members WHERE id = $1 AND workspace_id = $2`, memberID, workspaceID); err != nil {
		if err == sql.ErrNoRows {
			return ErrWorkspaceMemberNotFound
		}
		return err
	}
	isLeaving := actor.ID == current.ID
	if !isLeaving && (!actor.Can(model.WorkspacePermissionManageMembers) ||
		(current.Role == model.WorkspaceRoleOwner && !actor.Can(model.WorkspacePermissionManageOwners))) {
		return ErrWorkspaceMemberPermission
	}
	if current.Role == model.WorkspaceRoleOwner {
		var owners int
		if err := tx.Get(&owners, `SELECT COUNT(*) FROM workspace_members WHERE workspace_id = $1 AND role = 'owner'`, workspaceID); err != nil {
			return err
		}
		if owners <= 1 {
			return ErrLastOwner
		}
	}
	result, err := tx.Exec(`DELETE FROM workspace_members WHERE id = $1 AND workspace_id = $2`, memberID, workspaceID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWorkspaceMemberNotFound
	}
	if !isLeaving {
		event.Metadata = auditMetadata(map[string]string{"user_id": current.UserID})
	}
	if err := insertAuditEvent(tx, event); err != nil {
		return err
	}
	return tx.Commit()
}

func auditMetadata(values map[string]string) string {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}

func (r *workspaceMemberRepository) DeleteUserIfOwnershipSafe(userID string) error {
	var workspaceIDs []string
	if err := r.db.Select(&workspaceIDs, `
		SELECT workspace_id
		FROM workspace_members
		WHERE user_id = $1
		ORDER BY workspace_id
	`, userID); err != nil {
		return err
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, workspaceID := range workspaceIDs {
		if _, err := tx.Exec(`UPDATE workspaces SET updated_at = updated_at WHERE id = $1`, workspaceID); err != nil {
			return err
		}
	}

	var soleOwnerCount int
	if err := tx.Get(&soleOwnerCount, `
		SELECT COUNT(*)
		FROM workspace_members mine
		WHERE mine.user_id = $1
		  AND mine.role = 'owner'
		  AND NOT EXISTS (
		      SELECT 1 FROM workspace_members other
		      WHERE other.workspace_id = mine.workspace_id
		        AND other.role = 'owner'
		        AND other.user_id != mine.user_id
		  )
	`, userID); err != nil {
		return err
	}
	if soleOwnerCount > 0 {
		return ErrLastOwner
	}

	result, err := tx.Exec(`DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}
	return tx.Commit()
}
