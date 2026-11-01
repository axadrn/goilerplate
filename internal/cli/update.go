package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axadrn/goilerplate/v4/generator"
	"github.com/axadrn/goilerplate/v4/internal/project"
	projectupdate "github.com/axadrn/goilerplate/v4/internal/update"
	"golang.org/x/mod/semver"
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
	current := a.templateVersion()
	if lock.TemplateVersion == current {
		a.printInfo("Already using goilerplate template " + current)
		return nil
	}
	if err := checkUpdatable(lock.TemplateVersion, current); err != nil {
		return err
	}
	if a.FetchTemplate == nil {
		return errors.New("goilerplate update cannot download templates in this build")
	}
	oldFiles, err := a.FetchTemplate(ctx, lock.TemplateVersion)
	if errors.Is(err, projectupdate.ErrVersionNotFound) {
		return fmt.Errorf("goilerplate %s was not found on the Go module proxy, so its template cannot be compared", lock.TemplateVersion)
	}
	if err != nil {
		return err
	}
	oldTemplate, err := generator.NewTemplate(lock.TemplateVersion, oldFiles)
	if err != nil {
		return err
	}
	newTemplate, err := generator.CurrentTemplate(current)
	if err != nil {
		return err
	}

	workspace, err := os.MkdirTemp("", "goilerplate-update-*")
	if err != nil {
		return fmt.Errorf("create update workspace: %w", err)
	}
	defer os.RemoveAll(workspace)
	configuration := lock.Config
	oldRoot := filepath.Join(workspace, "old")
	if err := renderTree(oldTemplate, configuration, oldRoot); err != nil {
		return fmt.Errorf("render goilerplate %s: %w", lock.TemplateVersion, err)
	}
	newRoot := filepath.Join(workspace, "new")
	if err := renderTree(newTemplate, configuration, newRoot); err != nil {
		return fmt.Errorf("render goilerplate %s: %w", current, err)
	}
	result, err := projectupdate.Merge(ctx, repository, oldRoot, newRoot, current)
	if err != nil {
		return err
	}
	a.printUpdateResult(result)
	return nil
}

// checkUpdatable says plainly when a project cannot be updated by this CLI.
func checkUpdatable(projectVersion, current string) error {
	if !semver.IsValid(projectVersion) {
		return fmt.Errorf("goilerplate.lock names an invalid template version %q", projectVersion)
	}
	if semver.Compare(semver.Major(projectVersion), projectupdate.FirstUpdatableMajor) < 0 {
		return fmt.Errorf("this project was made with goilerplate %s. goilerplate update works for projects made with %s or newer. The project keeps working as it is", projectVersion, projectupdate.FirstUpdatableMajor)
	}
	if semver.Prerelease(projectVersion) == "-dev" {
		return fmt.Errorf("this project was made with a development build of goilerplate (%s), which has no published template to update from", projectVersion)
	}
	if semver.Compare(projectVersion, current) > 0 {
		return fmt.Errorf("this project uses goilerplate %s, which is newer than this CLI (%s). Update the CLI: go install github.com/axadrn/goilerplate/v4/cmd/goilerplate@latest", projectVersion, current)
	}
	return nil
}

// renderTree renders a template the way goilerplate new does: through the
// same archive and extraction, so both trees match a generated project.
func renderTree(template generator.Template, configuration generator.Config, destination string) error {
	var archive bytes.Buffer
	if err := template.WriteArchive(&archive, configuration); err != nil {
		return err
	}
	if err := project.Extract(&archive, destination); err != nil {
		return fmt.Errorf("extract template tree: %w", err)
	}
	return nil
}

func (a *App) printUpdateResult(result projectupdate.Result) {
	if !a.StyledOutput {
		fmt.Fprintf(a.Out, "Created %s\n", result.Branch)
		fmt.Fprintf(a.Out, "%s updated cleanly\n", files(len(result.Updated)))
		for _, name := range result.Updated {
			fmt.Fprintf(a.Out, "  %s\n", name)
		}
		if len(result.Conflicts) == 0 {
			fmt.Fprintf(a.Out, "Review it with: git switch %s\n", result.Branch)
			return
		}
		fmt.Fprintf(a.Out, "%s conflict resolution\n", needFiles(len(result.Conflicts)))
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
	a.printField("Updated", files(len(result.Updated)))
	for _, name := range result.Updated {
		a.printListItem(name)
	}
	if len(result.Conflicts) == 0 {
		a.printField("Review with", "git switch "+result.Branch)
		return
	}
	a.printWarning(needFiles(len(result.Conflicts)) + " conflict resolution")
	for _, name := range result.Conflicts {
		a.printListItem(name)
	}
	a.printField("Switch with", "git switch "+result.Branch)
	a.printInfo("Resolve the normal Git conflict markers, then commit the result")
}

// files counts files in plain English: "1 file", "3 files".
func files(n int) string {
	if n == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", n)
}

// needFiles is "1 file needs" or "3 files need".
func needFiles(n int) string {
	if n == 1 {
		return "1 file needs"
	}
	return files(n) + " need"
}
