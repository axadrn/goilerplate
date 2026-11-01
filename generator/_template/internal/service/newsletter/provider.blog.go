package newsletter

import "context"

const (
	ProviderLocal  = "local"
	ProviderResend = "resend"
)

type Subscriber interface {
	Subscribe(ctx context.Context, email string) error
	Name() string
}
