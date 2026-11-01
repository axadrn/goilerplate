package service

import "fmt"

func forgotPasswordEmailTemplate(signInURL, appName string) emailContent {
	return emailContent{
		Subject:     fmt.Sprintf("Reset your password for %s", appName),
		Preheader:   "Reset your password securely.",
		Heading:     "Reset your password",
		Intro:       []string{"You requested to reset your password. For security, we'll remove your password and sign you in with this link."},
		ActionLabel: "Reset password",
		ActionURL:   signInURL,
		Outro: []string{
			"After signing in, you can set a new password in Settings.",
			"This link expires in 10 minutes and can only be used once.",
			"If you didn't request this, you can safely ignore this email. Your password won't be changed.",
		},
		AppName: appName,
	}
}

func magicLinkEmailTemplate(magicURL, appName string) emailContent {
	return emailContent{
		Subject:     fmt.Sprintf("Sign in to %s", appName),
		Preheader:   fmt.Sprintf("Use this secure link to sign in to %s.", appName),
		Heading:     fmt.Sprintf("Sign in to %s", appName),
		Intro:       []string{"Use the button below to sign in to your account."},
		ActionLabel: "Sign in",
		ActionURL:   magicURL,
		Outro: []string{
			"This link expires in 10 minutes and can only be used once.",
			"If you didn't request this, you can safely ignore this email.",
		},
		AppName: appName,
	}
}

func welcomeEmailTemplate(name, dashboardURL, appName string) emailContent {
	return emailContent{
		Subject:     fmt.Sprintf("Welcome to %s!", appName),
		Preheader:   "Your account is ready.",
		Heading:     fmt.Sprintf("Welcome to %s!", appName),
		Greeting:    fmt.Sprintf("Hi %s,", name),
		Intro:       []string{"Your email is verified and your account is active."},
		ActionLabel: "Get started",
		ActionURL:   dashboardURL,
		Outro:       []string{"If you have questions, reach out to our support team."},
		AppName:     appName,
	}
}

func workspaceInvitationEmailTemplate(invitationURL, workspaceName, inviterName, appName string) emailContent {
	return emailContent{
		Subject:     fmt.Sprintf("Join %s on %s", workspaceName, appName),
		Preheader:   fmt.Sprintf("%s invited you to join %s.", inviterName, workspaceName),
		Heading:     fmt.Sprintf("Join %s", workspaceName),
		Intro:       []string{fmt.Sprintf("%s invited you to join %s on %s.", inviterName, workspaceName, appName)},
		ActionLabel: "Accept the invitation",
		ActionURL:   invitationURL,
		Outro: []string{
			"This invitation expires in 7 days and can only be accepted by this email address.",
			"If you were not expecting this invitation, you can safely ignore this email.",
		},
		AppName: appName,
	}
}

func emailChangeVerificationTemplate(name, verifyURL, appName string) emailContent {
	return emailContent{
		Subject:     fmt.Sprintf("Verify your new email for %s", appName),
		Preheader:   "Verify your new email address.",
		Heading:     "Verify your new email",
		Greeting:    fmt.Sprintf("Hi %s,", name),
		Intro:       []string{"You requested to change your email address. Use the button below to verify the new address."},
		ActionLabel: "Verify email",
		ActionURL:   verifyURL,
		Outro: []string{
			"This link expires in 24 hours.",
			"If you didn't request this change, you can safely ignore this email.",
		},
		AppName: appName,
	}
}

func emailChangeNotificationTemplate(name, newEmail, appName string) emailContent {
	return emailContent{
		Subject:   fmt.Sprintf("Email change requested for %s", appName),
		Preheader: "A change to your email address was requested.",
		Heading:   "Email change requested",
		Greeting:  fmt.Sprintf("Hi %s,", name),
		Intro: []string{
			fmt.Sprintf("A request was made to change your email address to %s.", newEmail),
			"If this was you, verify the new email address using the link we sent to it.",
		},
		Outro:   []string{"If you didn't request this change, your account may be compromised. Please secure your account immediately by changing your password."},
		AppName: appName,
	}
}

func accountDeletedEmailTemplate(name, appName string) emailContent {
	return emailContent{
		Subject:   fmt.Sprintf("Your %s account has been deleted", appName),
		Preheader: "Your account has been deleted.",
		Heading:   "Your account has been deleted",
		Greeting:  fmt.Sprintf("Hi %s,", name),
		Intro: []string{
			fmt.Sprintf("Your account has been permanently deleted from %s.", appName),
			"All your data, including your profile, files, and settings, has been removed from our systems.",
		},
		Outro: []string{
			"If you didn't request this deletion, please contact our support team immediately, though we won't be able to recover your account.",
			"We're sorry to see you go. If you change your mind, you're welcome to create a new account anytime.",
		},
		AppName: appName,
	}
}
