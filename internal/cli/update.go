package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
		Answers:         lock.Config,
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
	targetVersion, err := client.UpdateTree(ctx, token, api.GenerateRequest{Answers: lock.Config}, newArchive)
	if err != nil {
		return err
	}
	if targetVersion == lock.TemplateVersion {
		a.printInfo("Already using goilerplate template " + targetVersion)
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
	a.printUpdateResult(result)
	return nil
}

func (a *App) updateClient(ctx context.Context) (ServiceClient, string, error) {
	configuration, err := a.Store.Load()
	if err != nil {
		return nil, "", err
	}
	if configuration.SessionToken == "" {
		if err := a.login(ctx, nil); err != nil {
			return nil, "", err
		}
		configuration, err = a.Store.Load()
		if err != nil {
			return nil, "", err
		}
	}
	client, err := a.NewService(a.apiURL(configuration))
	if err != nil {
		return nil, "", err
	}
	return client, configuration.SessionToken, nil
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

func (a *App) printUpdateResult(result projectupdate.Result) {
	if !a.StyledOutput {
		fmt.Fprintf(a.Out, "Created %s\n", result.Branch)
		fmt.Fprintf(a.Out, "%d files updated cleanly\n", len(result.Updated))
		for _, name := range result.Updated {
			fmt.Fprintf(a.Out, "  %s\n", name)
		}
		if len(result.Conflicts) == 0 {
			fmt.Fprintf(a.Out, "Review it with: git switch %s\n", result.Branch)
			return
		}
		fmt.Fprintf(a.Out, "%d files need conflict resolution\n", len(result.Conflicts))
		for _, name := range result.Conflicts {
			fmt.Fprintf(a.Out, "  %s\n", name)
		}
		fmt.Fprintf(a.Out, "Switch with: git switch %s\n", result.Branch)
		fmt.Fprintln(a.Out, "Resolve the normal Git conflict markers, then commit the result")
		return
	}
	a.printHeading("update")
	fmt.Fprintln(a.Out)
	a.printSuccess("Created " + result.Branch)
	a.printField("Updated", fmt.Sprintf("%d files", len(result.Updated)))
	for _, name := range result.Updated {
		a.printListItem(name)
	}
	if len(result.Conflicts) == 0 {
		a.printField("Review with", "git switch "+result.Branch)
		return
	}
	a.printWarning(fmt.Sprintf("%d files need conflict resolution", len(result.Conflicts)))
	for _, name := range result.Conflicts {
		a.printListItem(name)
	}
	a.printField("Switch with", "git switch "+result.Branch)
	a.printInfo("Resolve the normal Git conflict markers, then commit the result")
}
