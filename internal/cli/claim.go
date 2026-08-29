package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

func (a *App) claim(ctx context.Context, arguments []string) error {
	flags := flag.NewFlagSet("claim", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	code := flags.String("code", "", "confirm with the emailed code")
	if err := flags.Parse(arguments); err != nil {
		return claimUsage()
	}
	normalizedCode := strings.ToUpper(strings.TrimSpace(*code))
	if normalizedCode != "" {
		if flags.NArg() != 0 {
			return claimUsage()
		}
	} else if flags.NArg() != 1 {
		return claimUsage()
	}
	client, sessionToken, err := a.signedInClient()
	if err != nil {
		return err
	}
	if normalizedCode != "" {
		if err := client.ConfirmLicenseClaim(ctx, sessionToken, normalizedCode); err != nil {
			return err
		}
		if a.StyledOutput {
			a.printSuccess("License claimed")
			a.printInfo("Run goilerplate whoami to see it")
		} else {
			fmt.Fprintln(a.Out, "License claimed. Run goilerplate whoami to see it")
		}
		return nil
	}
	if err := client.BeginLicenseClaim(ctx, sessionToken, flags.Arg(0)); err != nil {
		return err
	}
	if a.StyledOutput {
		a.printInfo("If that email belongs to an unclaimed purchase, a claim code is on its way")
		a.printField("Finish with", "goilerplate claim --code <code>")
	} else {
		fmt.Fprintln(a.Out, "If that email belongs to an unclaimed purchase, a claim code is on its way")
		fmt.Fprintln(a.Out, "Finish with: goilerplate claim --code <code>")
	}
	return nil
}

func claimUsage() error {
	return errors.New("usage: goilerplate claim <purchase-email> or goilerplate claim --code <code>")
}
