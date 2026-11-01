package newsletter

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type ResendSubscriber struct {
	client     resendContactCreator
	audienceID string
}

type resendContactCreator interface {
	CreateWithContext(ctx context.Context, params *resend.CreateContactRequest) (resend.CreateContactResponse, error)
}

func NewResendSubscriber(apiKey, audienceID string) *ResendSubscriber {
	client := resend.NewClient(apiKey)
	return &ResendSubscriber{client: client.Contacts, audienceID: audienceID}
}

func (s *ResendSubscriber) Name() string {
	return ProviderResend
}

func (s *ResendSubscriber) Subscribe(ctx context.Context, email string) error {
	_, err := s.client.CreateWithContext(ctx, &resend.CreateContactRequest{
		Email:      email,
		AudienceId: s.audienceID,
	})
	if err != nil {
		return fmt.Errorf("subscribe with Resend: %w", err)
	}
	return nil
}
