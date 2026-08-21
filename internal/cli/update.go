package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/axadrn/goilerplate/v3/api"
	"github.com/axadrn/goilerplate/v3/internal/project"
	projectupdate "github.com/axadrn/goilerplate/v3/internal/update"
)

func (a *App) updateProject(ctx context.Context, arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: goilerplate update")
	}
	directory := a.WorkingDirectory
	if directory == "" {
		var err error
		directory, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("find working directory: %w", err)
		}
	}
	repository, err := projectupdate.Open(ctx, directory)
	if err != nil {
		return err
	}
	lock, err := projectupdate.ReadLock(repository.Root)
	if err != nil {
		return err
	}
	client, token, err := a.updateClient(ctx)
	if err != nil {
		return err
	}

	workspace, err := os.MkdirTemp("", "goilerplate-update-*")
	if err != nil {
		return fmt.Errorf("create update workspace: %w", err)
	}
	defer os.RemoveAll(workspace)
	oldArchive, err := os.Create(filepath.Join(workspace, "old.tar.gz"))
	if err != nil {
		return err
	}
	defer oldArchive.Close()
	oldVersion, err := client.UpdateTree(ctx, token, api.GenerateRequest{
		TemplateVersion: lock.TemplateVersion,
		Answers:         lock.Answers,
	}, oldArchive)
	if err != nil {
		return err
	}
	if oldVersion != lock.TemplateVersion {
		return fmt.Errorf("service returned old template %s, want %s", oldVersion, lock.TemplateVersion)
	}

	newArchive, err := os.Create(filepath.Join(workspace, "new.tar.gz"))
	if err != nil {
		return err
	}
	defer newArchive.Close()
	targetVersion, err := client.UpdateTree(ctx, token, api.GenerateRequest{Answers: lock.Answers}, newArchive)
	if err != nil {
		return err
	}
	if targetVersion == lock.TemplateVersion {
		fmt.Fprintf(a.Out, "Already using goilerplate template %s\n", targetVersion)
		return nil
	}
	oldRoot := filepath.Join(workspace, "old")
	newRoot := filepath.Join(workspace, "new")
	if err := extractArchive(oldArchive, oldRoot); err != nil {
		return err
	}
	if err := extractArchive(newArchive, newRoot); err != nil {
		return err
	}
	result, err := projectupdate.Merge(ctx, repository, oldRoot, newRoot, targetVersion)
	if err != nil {
		return err
	}
	printUpdateResult(a.Out, result)
	return nil
}

func (a *App) updateClient(ctx context.Context) (ServiceClient, string, error) {
	configuration, err := a.Store.Load()
	if err != nil {
		return nil, "", err
	}
	token := strings.TrimSpace(a.MachineToken)
	if token == "" && configuration.SessionToken == "" {
		if err := a.login(ctx, nil); err != nil {
			return nil, "", err
		}
		configuration, err = a.Store.Load()
		if err != nil {
			return nil, "", err
		}
	}
	if token == "" {
		token = configuration.SessionToken
	}
	apiURL := configuration.APIURL
	if apiURL == "" {
		apiURL = a.DefaultAPIURL
	}
	client, err := a.NewService(apiURL)
	if err != nil {
		return nil, "", err
	}
	return client, token, nil
}

func extractArchive(file *os.File, destination string) error {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("read update download: %w", err)
	}
	if err := project.Extract(file, destination); err != nil {
		return fmt.Errorf("extract update tree: %w", err)
	}
	return nil
}

func printUpdateResult(output io.Writer, result projectupdate.Result) {
	fmt.Fprintf(output, "Created %s\n", result.Branch)
	fmt.Fprintf(output, "%d files updated cleanly\n", len(result.Updated))
	for _, name := range result.Updated {
		fmt.Fprintf(output, "  %s\n", name)
	}
	if len(result.Conflicts) == 0 {
		fmt.Fprintf(output, "Review it with: git switch %s\n", result.Branch)
		return
	}
	fmt.Fprintf(output, "%d files need conflict resolution\n", len(result.Conflicts))
	for _, name := range result.Conflicts {
		fmt.Fprintf(output, "  %s\n", name)
	}
	fmt.Fprintf(output, "Switch with: git switch %s\n", result.Branch)
	fmt.Fprintln(output, "Resolve the normal Git conflict markers, then commit the result")
}
