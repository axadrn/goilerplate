package service

import (
	"context"
	"log/slog"

	"{{.ModulePath}}/internal/service/newsletter"
)

type NewsletterService struct {
	subscriber newsletter.Subscriber
}

func NewNewsletterService(subscriber newsletter.Subscriber) *NewsletterService {
	return &NewsletterService{subscriber: subscriber}
}

func (s *NewsletterService) Subscribe(email string) error {
	if err := s.subscriber.Subscribe(context.Background(), email); err != nil {
		return err
	}
	slog.Info("newsletter subscription successful", "provider", s.subscriber.Name())
	return nil
}
