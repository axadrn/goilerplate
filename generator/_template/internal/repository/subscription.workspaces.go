package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
)

var (
	ErrSubscriptionNotFound    = errors.New("subscription not found")
	ErrCheckoutPending         = errors.New("checkout is still being created")
	ErrCheckoutSelection       = errors.New("a different checkout is already open")
	ErrCheckoutURLMissing      = errors.New("checkout URL is missing")
	ErrSubscriptionAlreadyPaid = errors.New("subscription is already paid")
)

type SubscriptionRepository interface {
	Create(sub *model.Subscription) error
	CreateIfMissing(sub *model.Subscription) error
	ByWorkspaceID(workspaceID string) (*model.Subscription, error)
	ByProviderSubscriptionID(providerSubID string) (*model.Subscription, error)
	ByProviderCustomerID(providerCustomerID string) (*model.Subscription, error)
	Update(sub *model.Subscription) error
	UpdateIfProviderStateNewer(sub *model.Subscription, providerSubID string, stateUpdatedAt time.Time, subscriptionCreatedAt *time.Time) (bool, error)
	UpdateIfProviderPaymentNewer(sub *model.Subscription, providerSubID string, paymentUpdatedAt time.Time) (bool, error)
	UpdateIfProviderOrderNewer(sub *model.Subscription, orderUpdatedAt time.Time) (bool, error)
	BeginCheckout(workspaceID, attemptID, provider, planID, interval string, expiresAt, pendingUntil, now time.Time) (string, bool, error)
	FinishCheckout(workspaceID, attemptID, checkoutURL string, expiresAt, pendingUntil, now time.Time) error
	RejectCheckout(workspaceID, attemptID string, now time.Time) error
}

type subscriptionRepository struct {
	db *sqlx.DB
}

func NewSubscriptionRepository(db *sqlx.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) Create(sub *model.Subscription) error {
	return r.create(sub, "")
}

func (r *subscriptionRepository) CreateIfMissing(sub *model.Subscription) error {
	return r.create(sub, " ON CONFLICT (workspace_id) DO NOTHING")
}

func (r *subscriptionRepository) create(sub *model.Subscription, conflictClause string) error {
	query := `
		INSERT INTO subscriptions (
			id, workspace_id, plan_id, status, provider,
			provider_customer_id, provider_subscription_id,
			provider_subscription_created_at, provider_state_updated_at, provider_payment_updated_at,
			checkout_attempt_id, checkout_provider, checkout_plan_id, checkout_interval,
			checkout_url, checkout_expires_at, checkout_pending_until,
			current_period_end, amount, currency, interval,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)` + conflictClause + `
	`

	_, err := r.db.Exec(
		query,
		sub.ID,
		sub.WorkspaceID,
		sub.PlanID,
		sub.Status,
		sub.Provider,
		sub.ProviderCustomerID,
		sub.ProviderSubscriptionID,
		sub.ProviderSubscriptionCreatedAt,
		sub.ProviderStateUpdatedAt,
		sub.ProviderPaymentUpdatedAt,
		sub.CheckoutAttemptID,
		sub.CheckoutProvider,
		sub.CheckoutPlanID,
		sub.CheckoutInterval,
		sub.CheckoutURL,
		sub.CheckoutExpiresAt,
		sub.CheckoutPendingUntil,
		sub.CurrentPeriodEnd,
		sub.Amount,
		sub.Currency,
		sub.Interval,
		sub.CreatedAt,
		sub.UpdatedAt,
	)

	return err
}

func (r *subscriptionRepository) ByWorkspaceID(workspaceID string) (*model.Subscription, error) {
	sub := &model.Subscription{}
	query := `SELECT * FROM subscriptions WHERE workspace_id = $1`

	err := r.db.Get(sub, query, workspaceID)
	if err == sql.ErrNoRows {
		return nil, ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (r *subscriptionRepository) ByProviderSubscriptionID(providerSubID string) (*model.Subscription, error) {
	sub := &model.Subscription{}
	query := `SELECT * FROM subscriptions WHERE provider_subscription_id = $1`

	err := r.db.Get(sub, query, providerSubID)
	if err == sql.ErrNoRows {
		return nil, ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (r *subscriptionRepository) ByProviderCustomerID(providerCustomerID string) (*model.Subscription, error) {
	sub := &model.Subscription{}
	query := `SELECT * FROM subscriptions WHERE provider_customer_id = $1`

	err := r.db.Get(sub, query, providerCustomerID)
	if err == sql.ErrNoRows {
		return nil, ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func (r *subscriptionRepository) Update(sub *model.Subscription) error {
	updated, err := r.updateWhere(sub, "TRUE")
	if err != nil {
		return err
	}
	if !updated {
		return ErrSubscriptionNotFound
	}
	return nil
}

func (r *subscriptionRepository) UpdateIfProviderStateNewer(
	sub *model.Subscription,
	providerSubID string,
	stateUpdatedAt time.Time,
	subscriptionCreatedAt *time.Time,
) (bool, error) {
	paymentReset := ""
	if subscriptionCreatedAt != nil {
		paymentReset = "provider_payment_updated_at = NULL,"
	}
	query := `
		UPDATE subscriptions
		SET plan_id = $1,
		    status = $2,
		    provider = $3,
		    provider_customer_id = $4,
		    provider_subscription_id = $5,
		    provider_subscription_created_at = $6,
		    provider_state_updated_at = $7,
		    checkout_attempt_id = NULL,
		    checkout_provider = '',
		    checkout_plan_id = '',
		    checkout_interval = '',
		    checkout_url = NULL,
		    checkout_expires_at = NULL,
		    checkout_pending_until = NULL,
		    ` + paymentReset + `
		    current_period_end = $8,
		    amount = CASE WHEN $1 = 'free' THEN NULL ELSE amount END,
		    currency = CASE WHEN $1 = 'free' THEN '' ELSE currency END,
		    interval = $9,
		    updated_at = $10
		WHERE id = $11 AND (`
	args := []any{
		sub.PlanID,
		sub.Status,
		sub.Provider,
		sub.ProviderCustomerID,
		sub.ProviderSubscriptionID,
		sub.ProviderSubscriptionCreatedAt,
		sub.ProviderStateUpdatedAt,
		sub.CurrentPeriodEnd,
		sub.Interval,
		sub.UpdatedAt,
		sub.ID,
	}
	if subscriptionCreatedAt != nil {
		query += `provider_subscription_created_at IS NULL
			 OR provider_subscription_created_at < $12
			 OR (provider_subscription_created_at = $12
			     AND (provider_state_updated_at IS NULL OR provider_state_updated_at < $13)))`
		args = append(args, *subscriptionCreatedAt, stateUpdatedAt)
		return r.execConditionalUpdate(sub.WorkspaceID, query, args...)
	}
	query += `provider_subscription_id = $12
		 AND (provider_state_updated_at IS NULL OR provider_state_updated_at < $13))`
	args = append(args, providerSubID, stateUpdatedAt)
	return r.execConditionalUpdate(sub.WorkspaceID, query, args...)
}

func (r *subscriptionRepository) UpdateIfProviderPaymentNewer(
	sub *model.Subscription,
	providerSubID string,
	paymentUpdatedAt time.Time,
) (bool, error) {
	query := `
		UPDATE subscriptions
		SET amount = $1,
		    currency = $2,
		    provider_payment_updated_at = $3,
		    updated_at = $4
		WHERE id = $5
		  AND provider_subscription_id = $6
		  AND (provider_payment_updated_at IS NULL OR provider_payment_updated_at < $7)
	`
	return r.execConditionalUpdate(
		sub.WorkspaceID,
		query,
		sub.Amount,
		sub.Currency,
		sub.ProviderPaymentUpdatedAt,
		sub.UpdatedAt,
		sub.ID,
		providerSubID,
		paymentUpdatedAt,
	)
}

func (r *subscriptionRepository) UpdateIfProviderOrderNewer(sub *model.Subscription, orderUpdatedAt time.Time) (bool, error) {
	query := `
		UPDATE subscriptions
		SET provider = $1,
		    provider_customer_id = $2,
		    amount = $3,
		    currency = $4,
		    provider_payment_updated_at = $5,
		    updated_at = $6
		WHERE id = $7
		  AND (provider_state_updated_at IS NULL OR provider_state_updated_at < $8)
		  AND (provider_payment_updated_at IS NULL OR provider_payment_updated_at < $8)
	`
	return r.execConditionalUpdate(
		sub.WorkspaceID,
		query,
		sub.Provider,
		sub.ProviderCustomerID,
		sub.Amount,
		sub.Currency,
		sub.ProviderPaymentUpdatedAt,
		sub.UpdatedAt,
		sub.ID,
		orderUpdatedAt,
	)
}

func (r *subscriptionRepository) updateWhere(sub *model.Subscription, condition string, conditionArgs ...any) (bool, error) {
	query := `
		UPDATE subscriptions
		SET plan_id = $1,
		    status = $2,
		    provider = $3,
		    provider_customer_id = $4,
		    provider_subscription_id = $5,
		    provider_subscription_created_at = $6,
		    provider_state_updated_at = $7,
		    provider_payment_updated_at = $8,
		    checkout_attempt_id = $9,
		    checkout_provider = $10,
		    checkout_plan_id = $11,
		    checkout_interval = $12,
		    checkout_url = $13,
		    checkout_expires_at = $14,
		    checkout_pending_until = $15,
		    current_period_end = $16,
		    amount = $17,
		    currency = $18,
		    interval = $19,
		    updated_at = $20
		WHERE id = $21 AND (` + condition + `)
	`

	args := []any{
		sub.PlanID,
		sub.Status,
		sub.Provider,
		sub.ProviderCustomerID,
		sub.ProviderSubscriptionID,
		sub.ProviderSubscriptionCreatedAt,
		sub.ProviderStateUpdatedAt,
		sub.ProviderPaymentUpdatedAt,
		sub.CheckoutAttemptID,
		sub.CheckoutProvider,
		sub.CheckoutPlanID,
		sub.CheckoutInterval,
		sub.CheckoutURL,
		sub.CheckoutExpiresAt,
		sub.CheckoutPendingUntil,
		sub.CurrentPeriodEnd,
		sub.Amount,
		sub.Currency,
		sub.Interval,
		sub.UpdatedAt,
		sub.ID,
	}
	args = append(args, conditionArgs...)
	return r.execConditionalUpdate(sub.WorkspaceID, query, args...)
}

func (r *subscriptionRepository) BeginCheckout(workspaceID, attemptID, provider, planID, interval string, expiresAt, pendingUntil, now time.Time) (string, bool, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE workspaces SET updated_at = updated_at WHERE id = $1`, workspaceID); err != nil {
		return "", false, err
	}
	sub := &model.Subscription{}
	if err := tx.Get(sub, `SELECT * FROM subscriptions WHERE workspace_id = $1`, workspaceID); err != nil {
		if err == sql.ErrNoRows {
			return "", false, ErrSubscriptionNotFound
		}
		return "", false, err
	}
	if sub.HasProviderSubscription() {
		return "", false, ErrSubscriptionAlreadyPaid
	}
	if sub.CheckoutPendingUntil != nil && now.Before(*sub.CheckoutPendingUntil) {
		if sub.CheckoutURL != nil && *sub.CheckoutURL != "" && sub.CheckoutExpiresAt != nil && now.Before(*sub.CheckoutExpiresAt) {
			if sub.CheckoutProvider != provider || sub.CheckoutPlanID != planID || sub.CheckoutInterval != interval {
				return "", false, ErrCheckoutSelection
			}
			return *sub.CheckoutURL, true, nil
		}
		return "", false, ErrCheckoutPending
	}
	if _, err := tx.Exec(`
		UPDATE subscriptions
		SET checkout_attempt_id = $1, checkout_provider = $2, checkout_plan_id = $3,
		    checkout_interval = $4, checkout_url = NULL, checkout_expires_at = $5,
		    checkout_pending_until = $6, updated_at = $7
		WHERE workspace_id = $8
	`, attemptID, provider, planID, interval, expiresAt, pendingUntil, now, workspaceID); err != nil {
		return "", false, err
	}
	return "", false, tx.Commit()
}

func (r *subscriptionRepository) FinishCheckout(workspaceID, attemptID, checkoutURL string, expiresAt, pendingUntil, now time.Time) error {
	if checkoutURL == "" {
		return ErrCheckoutURLMissing
	}
	updated, err := r.execConditionalUpdate(workspaceID, `
		UPDATE subscriptions
		SET checkout_url = $1, checkout_expires_at = $2, checkout_pending_until = $3, updated_at = $4
		WHERE workspace_id = $5 AND checkout_attempt_id = $6
	`, checkoutURL, expiresAt, pendingUntil, now, workspaceID, attemptID)
	if err != nil {
		return err
	}
	if !updated {
		return ErrCheckoutPending
	}
	return nil
}

func (r *subscriptionRepository) RejectCheckout(workspaceID, attemptID string, now time.Time) error {
	_, err := r.execConditionalUpdate(workspaceID, `
		UPDATE subscriptions
		SET checkout_attempt_id = NULL, checkout_provider = '', checkout_plan_id = '',
		    checkout_interval = '', checkout_url = NULL, checkout_expires_at = NULL,
		    checkout_pending_until = NULL, updated_at = $1
		WHERE workspace_id = $2 AND checkout_attempt_id = $3
	`, now, workspaceID, attemptID)
	return err
}

func (r *subscriptionRepository) execConditionalUpdate(workspaceID, query string, args ...any) (bool, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.Exec(`UPDATE workspaces SET updated_at = updated_at WHERE id = $1`, workspaceID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rows == 0 {
		return false, ErrWorkspaceNotFound
	}
	result, err = tx.Exec(query, args...)
	if err != nil {
		return false, err
	}

	rows, err = result.RowsAffected()
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return rows > 0, nil
}
