package cli

import (
	"context"
	"time"
)

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
		if status.Active {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
