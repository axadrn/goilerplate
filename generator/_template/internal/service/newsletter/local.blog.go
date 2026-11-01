package newsletter

import (
	"context"
	"fmt"
)

type LocalRepository interface {
	Subscribe(email string) error
}

type LocalSubscriber struct {
	repository LocalRepository
}

func NewLocalSubscriber(repository LocalRepository) *LocalSubscriber {
	return &LocalSubscriber{repository: repository}
}

func (s *LocalSubscriber) Name() string {
	return ProviderLocal
}

func (s *LocalSubscriber) Subscribe(_ context.Context, email string) error {
	if err := s.repository.Subscribe(email); err != nil {
		return fmt.Errorf("store newsletter subscriber: %w", err)
	}
	return nil
}
