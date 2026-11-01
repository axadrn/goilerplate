package repository

import (
	"errors"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
)

// ErrAuditCursorNotFound means a page cursor names no event of the workspace.
var ErrAuditCursorNotFound = errors.New("audit cursor not found")

type WorkspaceAuditRepository interface {
	Create(event *model.WorkspaceAuditEvent) error
	ListByWorkspace(workspaceID string, limit int) ([]model.WorkspaceAuditEvent, error)
	ListPage(workspaceID, afterID string, limit int) ([]model.WorkspaceAuditEvent, error)
}

type workspaceAuditRepository struct {
	db *sqlx.DB
}

func NewWorkspaceAuditRepository(db *sqlx.DB) WorkspaceAuditRepository {
	return &workspaceAuditRepository{db: db}
}

func (r *workspaceAuditRepository) Create(event *model.WorkspaceAuditEvent) error {
	_, err := r.db.Exec(`
		INSERT INTO workspace_audit_events (id, workspace_id, actor_user_id, action, target_type, target_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.WorkspaceID, event.ActorUserID, event.Action, event.TargetType, event.TargetID, event.Metadata, event.CreatedAt)
	return err
}

func (r *workspaceAuditRepository) ListByWorkspace(workspaceID string, limit int) ([]model.WorkspaceAuditEvent, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var events []model.WorkspaceAuditEvent
	err := r.db.Select(&events, `
		SELECT e.*, COALESCE(u.email, '') AS actor_email
		FROM workspace_audit_events e
		LEFT JOIN users u ON u.id = e.actor_user_id
		WHERE e.workspace_id = $1
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $2
	`, workspaceID, limit)
	return events, err
}

// ListPage returns up to limit events of a workspace, newest first, ordered
// by (created_at, id). With afterID it continues after that event. The
// cursor event is compared inside the database, so the page boundary uses
// exactly the stored values and the same order as ORDER BY.
func (r *workspaceAuditRepository) ListPage(workspaceID, afterID string, limit int) ([]model.WorkspaceAuditEvent, error) {
	var events []model.WorkspaceAuditEvent
	if afterID == "" {
		err := r.db.Select(&events, `
			SELECT e.*, COALESCE(u.email, '') AS actor_email
			FROM workspace_audit_events e
			LEFT JOIN users u ON u.id = e.actor_user_id
			WHERE e.workspace_id = $1
			ORDER BY e.created_at DESC, e.id DESC
			LIMIT $2
		`, workspaceID, limit)
		return events, err
	}

	var found int
	err := r.db.Get(&found, `SELECT COUNT(*) FROM workspace_audit_events WHERE id = $1 AND workspace_id = $2`, afterID, workspaceID)
	if err != nil {
		return nil, err
	}
	if found == 0 {
		return nil, ErrAuditCursorNotFound
	}
	err = r.db.Select(&events, `
		SELECT e.*, COALESCE(u.email, '') AS actor_email
		FROM workspace_audit_events e
		JOIN workspace_audit_events cursor_event ON cursor_event.id = $2
		LEFT JOIN users u ON u.id = e.actor_user_id
		WHERE e.workspace_id = $1
		  AND (e.created_at < cursor_event.created_at
		       OR (e.created_at = cursor_event.created_at AND e.id < cursor_event.id))
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $3
	`, workspaceID, afterID, limit)
	return events, err
}
