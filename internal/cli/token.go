package cli

import (
	"context"
	"errors"
	"fmt"
)

func (a *App) token(ctx context.Context, arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("usage: goilerplate token <create|list|revoke>")
	}
	client, sessionToken, err := a.signedInClient()
	if err != nil {
		return err
	}
	switch arguments[0] {
	case "create":
		if len(arguments) != 3 {
			return errors.New("usage: goilerplate token create <license-id> <name>")
		}
		response, err := client.CreateLicenseToken(ctx, sessionToken, arguments[1], arguments[2])
		if err != nil {
			return err
		}
		fmt.Fprintln(a.Out, response.Value)
		fmt.Fprintln(a.Out, "Save this key now. It is shown only once.")
		return nil
	case "list":
		if len(arguments) != 2 {
			return errors.New("usage: goilerplate token list <license-id>")
		}
		response, err := client.LicenseTokens(ctx, sessionToken, arguments[1])
		if err != nil {
			return err
		}
		for _, token := range response.Tokens {
			status := "active"
			if token.RevokedAt != nil {
				status = "revoked"
			}
			fmt.Fprintf(a.Out, "%s  %s  %s  created %s\n", token.ID, token.Name, status, token.CreatedAt.Format("2006-01-02"))
		}
		return nil
	case "revoke":
		if len(arguments) != 3 {
			return errors.New("usage: goilerplate token revoke <license-id> <token-id>")
		}
		if err := client.RevokeLicenseToken(ctx, sessionToken, arguments[1], arguments[2]); err != nil {
			return err
		}
		fmt.Fprintln(a.Out, "CI key revoked")
		return nil
	default:
		return fmt.Errorf("unknown token command %q", arguments[0])
	}
}
