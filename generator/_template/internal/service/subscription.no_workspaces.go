package service

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"{{.ModulePath}}/internal/access"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
)

type SubscriptionService struct {
	repo repository.SubscriptionRepository
}

func NewSubscriptionService(repo repository.SubscriptionRepository) *SubscriptionService {
	return &SubscriptionService{repo: repo}
}

func (s *SubscriptionService) CreateFreeSubscription(userID string) error {
	now := time.Now()
	subscription := &model.Subscription{
		ID:        uuid.New().String(),
		UserID:    userID,
		PlanID:    model.SubscriptionPlanFree,
		Status:    model.SubscriptionStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := s.repo.Create(subscription)
	if err != nil {
		return fmt.Errorf("failed to create free subscription: %w", err)
	}

	return nil
}

func (s *SubscriptionService) CreateInitialSubscription(userID string) error {
	now := time.Now()
	initial := access.Initial(now)
	subscription := &model.Subscription{
		ID:               uuid.New().String(),
		UserID:           userID,
		PlanID:           initial.PlanID,
		Status:           initial.Status,
		CurrentPeriodEnd: initial.CurrentPeriodEnd,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.Create(subscription); err != nil {
		return fmt.Errorf("failed to create initial subscription: %w", err)
	}
	return nil
}

func (s *SubscriptionService) Subscription(userID string) (*model.Subscription, error) {
	sub, err := s.repo.ByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	return sub, nil
}

func (s *SubscriptionService) Access(userID string) (access.Access, error) {
	subscription, err := s.Subscription(userID)
	if err != nil {
		return access.Access{}, err
	}
	return subscription.Access(), nil
}

func (s *SubscriptionService) ByProviderSubscriptionID(providerSubID string) (*model.Subscription, error) {
	sub, err := s.repo.ByProviderSubscriptionID(providerSubID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription by provider ID: %w", err)
	}

	return sub, nil
}

func (s *SubscriptionService) ByProviderCustomerID(providerCustomerID string) (*model.Subscription, error) {
	sub, err := s.repo.ByProviderCustomerID(providerCustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription by provider customer ID: %w", err)
	}

	return sub, nil
}

func (s *SubscriptionService) UpdateSubscription(sub *model.Subscription) error {
	sub.UpdatedAt = time.Now()

	err := s.repo.Update(sub)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	return nil
}

func (s *SubscriptionService) UpdateProviderState(
	sub *model.Subscription,
	providerSubscriptionID string,
	stateUpdatedAt time.Time,
	subscriptionCreatedAt *time.Time,
) (bool, error) {
	sub.UpdatedAt = time.Now()
	updated, err := s.repo.UpdateIfProviderStateNewer(sub, providerSubscriptionID, stateUpdatedAt, subscriptionCreatedAt)
	if err != nil {
		return false, fmt.Errorf("failed to update provider subscription state: %w", err)
	}
	return updated, nil
}

func (s *SubscriptionService) UpdateProviderPayment(sub *model.Subscription, providerSubscriptionID string, paymentUpdatedAt time.Time) (bool, error) {
	sub.UpdatedAt = time.Now()
	updated, err := s.repo.UpdateIfProviderPaymentNewer(sub, providerSubscriptionID, paymentUpdatedAt)
	if err != nil {
		return false, fmt.Errorf("failed to update provider payment state: %w", err)
	}
	return updated, nil
}

func (s *SubscriptionService) UpdateProviderOrder(sub *model.Subscription, orderUpdatedAt time.Time) (bool, error) {
	sub.UpdatedAt = time.Now()
	updated, err := s.repo.UpdateIfProviderOrderNewer(sub, orderUpdatedAt)
	if err != nil {
		return false, fmt.Errorf("failed to update provider order state: %w", err)
	}
	return updated, nil
}

func (s *SubscriptionService) EndPaidAccess(sub *model.Subscription) error {
	sub.Status = model.SubscriptionStatusInactive
	sub.ProviderSubscriptionID = nil
	sub.CurrentPeriodEnd = nil
	sub.Amount = nil
	sub.Currency = ""
	sub.Interval = nil

	return s.UpdateSubscription(sub)
}
