package newsletter

import (
	"context"
	"errors"
	"testing"

	"github.com/resend/resend-go/v2"
)

type localRepositoryStub struct {
	email string
	err   error
}

func (s *localRepositoryStub) Subscribe(email string) error {
	s.email = email
	return s.err
}

type resendContactStub struct {
	request *resend.CreateContactRequest
	err     error
}

func (s *resendContactStub) CreateWithContext(_ context.Context, request *resend.CreateContactRequest) (resend.CreateContactResponse, error) {
	s.request = request
	return resend.CreateContactResponse{}, s.err
}

func TestLocalSubscriberStoresEmail(t *testing.T) {
	repository := &localRepositoryStub{}
	subscriber := NewLocalSubscriber(repository)

	if err := subscriber.Subscribe(context.Background(), "reader@example.com"); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if repository.email != "reader@example.com" {
		t.Fatalf("stored email = %q", repository.email)
	}
}

func TestResendSubscriberUsesAudience(t *testing.T) {
	client := &resendContactStub{}
	subscriber := &ResendSubscriber{client: client, audienceID: "audience-1"}

	if err := subscriber.Subscribe(context.Background(), "reader@example.com"); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if client.request == nil || client.request.Email != "reader@example.com" || client.request.AudienceId != "audience-1" {
		t.Fatalf("Resend request = %#v", client.request)
	}
}

func TestSubscribersWrapProviderErrors(t *testing.T) {
	providerError := errors.New("provider unavailable")
	tests := []struct {
		name       string
		subscriber Subscriber
	}{
		{name: "local", subscriber: NewLocalSubscriber(&localRepositoryStub{err: providerError})},
		{name: "Resend", subscriber: &ResendSubscriber{client: &resendContactStub{err: providerError}, audienceID: "audience-1"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.subscriber.Subscribe(context.Background(), "reader@example.com")
			if err == nil || !errors.Is(err, providerError) {
				t.Fatalf("Subscribe() error = %v, want wrapped provider error", err)
			}
		})
	}
}
