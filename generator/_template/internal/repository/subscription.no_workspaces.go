package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"{{.ModulePath}}/internal/model"
)

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

type SubscriptionRepository interface {
	Create(sub *model.Subscription) error
	ByUserID(userID string) (*model.Subscription, error)
	ByProviderSubscriptionID(providerSubID string) (*model.Subscription, error)
	ByProviderCustomerID(providerCustomerID string) (*model.Subscription, error)
	Update(sub *model.Subscription) error
	UpdateIfProviderStateNewer(sub *model.Subscription, providerSubID string, stateUpdatedAt time.Time, subscriptionCreatedAt *time.Time) (bool, error)
	UpdateIfProviderPaymentNewer(sub *model.Subscription, providerSubID string, paymentUpdatedAt time.Time) (bool, error)
	UpdateIfProviderOrderNewer(sub *model.Subscription, orderUpdatedAt time.Time) (bool, error)
}

type subscriptionRepository struct {
	db *sqlx.DB
}

func NewSubscriptionRepository(db *sqlx.DB) SubscriptionRepository {
	return &subscriptionRepository{db: db}
}

func (r *subscriptionRepository) Create(sub *model.Subscription) error {
	query := `
		INSERT INTO subscriptions (
			id, user_id, plan_id, status, provider,
			provider_customer_id, provider_subscription_id,
			provider_subscription_created_at, provider_state_updated_at, provider_payment_updated_at,
			current_period_end, amount, currency, interval,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := r.db.Exec(
		query,
		sub.ID,
		sub.UserID,
		sub.PlanID,
		sub.Status,
		sub.Provider,
		sub.ProviderCustomerID,
		sub.ProviderSubscriptionID,
		sub.ProviderSubscriptionCreatedAt,
		sub.ProviderStateUpdatedAt,
		sub.ProviderPaymentUpdatedAt,
		sub.CurrentPeriodEnd,
		sub.Amount,
		sub.Currency,
		sub.Interval,
		sub.CreatedAt,
		sub.UpdatedAt,
	)

	return err
}

func (r *subscriptionRepository) ByUserID(userID string) (*model.Subscription, error) {
	sub := &model.Subscription{}
	query := `SELECT * FROM subscriptions WHERE user_id = $1`

	err := r.db.Get(sub, query, userID)
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
		return r.execConditionalUpdate(query, args...)
	}
	query += `provider_subscription_id = $12
		 AND (provider_state_updated_at IS NULL OR provider_state_updated_at < $13))`
	args = append(args, providerSubID, stateUpdatedAt)
	return r.execConditionalUpdate(query, args...)
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
		    current_period_end = $9,
		    amount = $10,
		    currency = $11,
		    interval = $12,
		    updated_at = $13
		WHERE id = $14 AND (` + condition + `)
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
		sub.CurrentPeriodEnd,
		sub.Amount,
		sub.Currency,
		sub.Interval,
		sub.UpdatedAt,
		sub.ID,
	}
	args = append(args, conditionArgs...)
	return r.execConditionalUpdate(query, args...)
}

func (r *subscriptionRepository) execConditionalUpdate(query string, args ...any) (bool, error) {
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return false, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}
