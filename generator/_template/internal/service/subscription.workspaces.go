package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"{{.ModulePath}}/internal/access"
	"{{.ModulePath}}/internal/model"
	"{{.ModulePath}}/internal/repository"
)

type SubscriptionService struct {
	repo       repository.SubscriptionRepository
	adminAudit repository.AdminRepository
}

func NewSubscriptionService(repo repository.SubscriptionRepository, adminAudit repository.AdminRepository) *SubscriptionService {
	return &SubscriptionService{repo: repo, adminAudit: adminAudit}
}

func (s *SubscriptionService) CreateFreeSubscription(workspaceID string) error {
	now := time.Now()
	subscription := &model.Subscription{
		ID:          uuid.New().String(),
		WorkspaceID: workspaceID,
		PlanID:      model.SubscriptionPlanFree,
		Status:      model.SubscriptionStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err := s.repo.Create(subscription)
	if err != nil {
		return fmt.Errorf("failed to create free subscription: %w", err)
	}

	return nil
}

func (s *SubscriptionService) Subscription(workspaceID string) (*model.Subscription, error) {
	sub, err := s.repo.ByWorkspaceID(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	return sub, nil
}

func (s *SubscriptionService) Access(workspaceID string) (access.Access, error) {
	subscription, err := s.Subscription(workspaceID)
	if err != nil {
		return access.Access{}, err
	}
	return subscription.Access(), nil
}

// EnsureSubscription restores the required row when an workspace is missing one.
func (s *SubscriptionService) EnsureSubscription(workspaceID string) (*model.Subscription, error) {
	sub, err := s.repo.ByWorkspaceID(workspaceID)
	if err == nil {
		return sub, nil
	}
	if !errors.Is(err, repository.ErrSubscriptionNotFound) {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	now := time.Now()
	initial := access.Initial(now)
	subscription := &model.Subscription{
		ID:               uuid.New().String(),
		WorkspaceID:      workspaceID,
		PlanID:           initial.PlanID,
		Status:           initial.Status,
		CurrentPeriodEnd: initial.CurrentPeriodEnd,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.repo.CreateIfMissing(subscription); err != nil {
		return nil, fmt.Errorf("failed to restore subscription: %w", err)
	}
	sub, err = s.repo.ByWorkspaceID(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get restored subscription: %w", err)
	}
	return sub, nil
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

func (s *SubscriptionService) UpdateSubscription(sub *model.Subscription, providerEventID ...string) error {
	sub.UpdatedAt = time.Now()

	err := s.repo.Update(sub)
	if err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	eventID := ""
	if len(providerEventID) > 0 {
		eventID = providerEventID[0]
	}
	if err := s.recordSubscriptionUpdate(sub, eventID); err != nil {
		return fmt.Errorf("record subscription audit event: %w", err)
	}

	return nil
}

func (s *SubscriptionService) BeginCheckout(workspaceID, provider, planID, interval string, expiresAt, pendingUntil time.Time) (string, string, bool, error) {
	attemptID := uuid.New().String()
	checkoutURL, reused, err := s.repo.BeginCheckout(workspaceID, attemptID, provider, planID, interval, expiresAt, pendingUntil, time.Now())
	if err != nil {
		return "", "", false, fmt.Errorf("failed to begin checkout: %w", err)
	}
	if reused && checkoutURL == "" {
		return "", "", false, fmt.Errorf("failed to begin checkout: reusable checkout has no URL")
	}
	return checkoutURL, attemptID, reused, nil
}

func (s *SubscriptionService) FinishCheckout(workspaceID, attemptID, checkoutURL string, expiresAt, pendingUntil time.Time) error {
	if err := s.repo.FinishCheckout(workspaceID, attemptID, checkoutURL, expiresAt, pendingUntil, time.Now()); err != nil {
		return fmt.Errorf("failed to finish checkout: %w", err)
	}
	return nil
}

func (s *SubscriptionService) RejectCheckout(workspaceID, attemptID string) {
	_ = s.repo.RejectCheckout(workspaceID, attemptID, time.Now())
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
	auditSubscription := sub
	if !updated {
		auditSubscription, err = s.repo.ByWorkspaceID(sub.WorkspaceID)
		if err != nil {
			return false, fmt.Errorf("load current subscription for audit: %w", err)
		}
	}
	if err := s.recordSubscriptionUpdate(auditSubscription, ""); err != nil {
		return updated, fmt.Errorf("record subscription audit event: %w", err)
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

func (s *SubscriptionService) EndPaidAccess(sub *model.Subscription, providerEventID ...string) error {
	sub.Status = model.SubscriptionStatusInactive
	sub.ProviderSubscriptionID = nil
	sub.CurrentPeriodEnd = nil
	sub.Amount = nil
	sub.Currency = ""
	sub.Interval = nil
	sub.ClearCheckout()

	return s.UpdateSubscription(sub, providerEventID...)
}

func (s *SubscriptionService) recordSubscriptionUpdate(sub *model.Subscription, providerEventID string) error {
	metadata, err := json.Marshal(map[string]string{
		"plan_id":  sub.PlanID,
		"provider": sub.Provider,
		"status":   sub.Status,
	})
	if err != nil {
		return err
	}
	providerSubscriptionID := ""
	if sub.ProviderSubscriptionID != nil {
		providerSubscriptionID = *sub.ProviderSubscriptionID
	}
	periodEnd := ""
	if sub.CurrentPeriodEnd != nil {
		periodEnd = sub.CurrentPeriodEnd.UTC().Format(time.RFC3339Nano)
	}
	stateUpdatedAt := ""
	if sub.ProviderStateUpdatedAt != nil {
		stateUpdatedAt = sub.ProviderStateUpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	key := ""
	if providerEventID != "" {
		key = strings.Join([]string{"provider-event", sub.Provider, providerEventID}, "|")
	} else {
		key = strings.Join([]string{
			sub.WorkspaceID,
			sub.Provider,
			providerSubscriptionID,
			sub.PlanID,
			sub.Status,
			periodEnd,
			stateUpdatedAt,
		}, "|")
	}
	workspaceID := sub.WorkspaceID
	event := &model.AdminAuditEvent{
		ID:          uuid.NewSHA1(uuid.NameSpaceOID, []byte(key)).String(),
		WorkspaceID: &workspaceID,
		Action:      "subscription.updated",
		TargetType:  "subscription",
		TargetID:    sub.ID,
		Metadata:    string(metadata),
		CreatedAt:   time.Now(),
	}
	return s.adminAudit.CreateAuditEvent(event)
}
