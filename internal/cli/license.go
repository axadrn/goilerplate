package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/axadrn/goilerplate/v3/api"
)

func (a *App) license(ctx context.Context, arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("usage: goilerplate license <invite|members|remove>")
	}
	switch arguments[0] {
	case "members":
		if len(arguments) != 2 {
			return errors.New("usage: goilerplate license members <license-id>")
		}
		client, token, err := a.signedInClient()
		if err != nil {
			return err
		}
		response, err := client.LicenseMembers(ctx, token, arguments[1])
		if err != nil {
			return err
		}
		for _, member := range response.Members {
			fmt.Fprintf(a.Out, "%s  @%s  %s  %s\n", member.UserID, member.GitHubLogin, member.Role, member.Email)
		}
		for _, invitation := range response.Invitations {
			fmt.Fprintf(a.Out, "%s  pending  %s  %s  expires %s\n", invitation.ID, invitation.Email, invitation.Role, invitation.ExpiresAt.Format("2006-01-02"))
		}
		return nil
	case "invite":
		flags := flag.NewFlagSet("license invite", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		owner := flags.Bool("owner", false, "invite as an Owner")
		if err := flags.Parse(arguments[1:]); err != nil || flags.NArg() != 2 {
			return errors.New("usage: goilerplate license invite [--owner] <license-id> <email>")
		}
		role := api.LicenseRoleMember
		if *owner {
			role = api.LicenseRoleOwner
		}
		client, token, err := a.signedInClient()
		if err != nil {
			return err
		}
		response, err := client.InviteLicenseMember(ctx, token, flags.Arg(0), api.InviteLicenseMemberRequest{Email: flags.Arg(1), Role: role})
		if err != nil {
			return err
		}
		if response.Joined {
			fmt.Fprintln(a.Out, "Access added. The developer already has a goilerplate account.")
		} else {
			fmt.Fprintln(a.Out, "Invitation sent. Access connects when the developer runs goilerplate login.")
		}
		return nil
	case "remove":
		if len(arguments) != 3 {
			return errors.New("usage: goilerplate license remove <license-id> <user-or-invitation-id>")
		}
		client, token, err := a.signedInClient()
		if err != nil {
			return err
		}
		if err := client.RemoveLicenseMember(ctx, token, arguments[1], arguments[2]); err != nil {
			return err
		}
		fmt.Fprintln(a.Out, "Access removed. Company CI keys were revoked when an active member was removed.")
		return nil
	default:
		return fmt.Errorf("unknown license command %q", arguments[0])
	}
}
