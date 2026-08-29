package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
)

func (a *App) account(ctx context.Context, arguments []string) error {
	if len(arguments) == 0 || arguments[0] != "delete" {
		return errors.New("usage: goilerplate account delete --confirm <github-login>")
	}
	flags := flag.NewFlagSet("account delete", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	confirmation := flags.String("confirm", "", "confirm with your GitHub login")
	if err := flags.Parse(arguments[1:]); err != nil || flags.NArg() != 0 || *confirmation == "" {
		return errors.New("usage: goilerplate account delete --confirm <github-login>")
	}
	client, sessionToken, err := a.signedInClient()
	if err != nil {
		return err
	}
	if err := client.DeleteAccount(ctx, sessionToken, *confirmation); err != nil {
		return err
	}
	configuration, err := a.Store.Load()
	if err != nil {
		return err
	}
	configuration.SessionToken = ""
	if err := a.Store.Save(configuration); err != nil {
		return fmt.Errorf("account was deleted, but the local session could not be cleared: %w", err)
	}
	a.printSuccess("Account deleted")
	return nil
}
