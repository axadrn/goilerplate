package service

import (
	"errors"
	"fmt"
	"time"
)

var ErrActiveSubscription = errors.New("cannot delete account with active subscription")

func (s *UserService) validateAccountDeletion(userID string) error {
	subscription, err := s.subscriptionService.Subscription(userID)
	if err != nil {
		return fmt.Errorf("failed to check subscription: %w", err)
	}
	if subscription.BlocksAccountDeletion(time.Now()) {
		return ErrActiveSubscription
	}
	return nil
}
