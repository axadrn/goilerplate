package project

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func initializeGit(ctx context.Context, root string) error {
	commands := [][]string{
		{"init", "--quiet", "--initial-branch=main"},
		{"add", "--all"},
		{"commit", "--quiet", "--message", "Initial commit"},
	}
	for _, arguments := range commands {
		command := exec.CommandContext(ctx, "git", arguments...)
		command.Dir = root
		command.Env = projectGitEnvironment()
		var stderr bytes.Buffer
		command.Stderr = &stderr
		if err := command.Run(); err != nil {
			detail := strings.TrimSpace(stderr.String())
			if detail == "" {
				detail = err.Error()
			}
			return fmt.Errorf("initialize Git repository: %s", detail)
		}
	}
	return nil
}

func projectGitEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+4)
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "GIT_AUTHOR_NAME=") || strings.HasPrefix(value, "GIT_AUTHOR_EMAIL=") ||
			strings.HasPrefix(value, "GIT_COMMITTER_NAME=") || strings.HasPrefix(value, "GIT_COMMITTER_EMAIL=") {
			continue
		}
		environment = append(environment, value)
	}
	return append(environment,
		"GIT_AUTHOR_NAME=goilerplate",
		"GIT_AUTHOR_EMAIL=updates@goilerplate.com",
		"GIT_COMMITTER_NAME=goilerplate",
		"GIT_COMMITTER_EMAIL=updates@goilerplate.com",
	)
}
