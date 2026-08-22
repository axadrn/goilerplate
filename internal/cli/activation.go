package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/axadrn/goilerplate/v3/api"
)

var errActivationResendRequired = errors.New("Free activation needs a new email. Run goilerplate activation resend")
var errActivationUnavailable = errors.New("no active license is available for this account")

func (a *App) waitForActivation(ctx context.Context, client ServiceClient, sessionToken string) error {
	interval := a.ActivationPollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		status, err := client.ActivationStatus(ctx, sessionToken)
		if err != nil {
			return err
		}
		switch status.State {
		case api.ActivationStateActive:
			return nil
		case api.ActivationStatePending:
		case api.ActivationStateResendRequired:
			return errActivationResendRequired
		case api.ActivationStateUnavailable:
			return errActivationUnavailable
		default:
			return fmt.Errorf("unknown activation state %q", status.State)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (a *App) activation(ctx context.Context, arguments []string) error {
	if len(arguments) != 1 || arguments[0] != "resend" {
		return errors.New("usage: goilerplate activation resend")
	}
	configuration, err := a.Store.Load()
	if err != nil {
		return err
	}
	if configuration.SessionToken == "" {
		return errors.New("not signed in. Run goilerplate login")
	}
	client, err := a.NewService(a.apiURL(configuration))
	if err != nil {
		return err
	}
	status, err := client.ResendActivation(ctx, configuration.SessionToken)
	if err != nil {
		return err
	}
	switch status.State {
	case api.ActivationStateActive:
		fmt.Fprintln(a.Out, "Free access is already active")
		return nil
	case api.ActivationStatePending:
	case api.ActivationStateResendRequired:
		return errActivationResendRequired
	case api.ActivationStateUnavailable:
		return errActivationUnavailable
	default:
		return fmt.Errorf("unknown activation state %q", status.State)
	}
	fmt.Fprintln(a.Out, "Activation email sent. Waiting for confirmation...")
	if err := a.waitForActivation(ctx, client, configuration.SessionToken); err != nil {
		return err
	}
	fmt.Fprintln(a.Out, "Free access activated")
	return nil
}
