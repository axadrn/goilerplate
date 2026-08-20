package update

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Repository struct {
	Root     string
	Head     string
	HeadTree string
}

type Result struct {
	Branch    string
	Updated   []string
	Conflicts []string
}

func Open(ctx context.Context, directory string) (Repository, error) {
	output, err := runGit(ctx, directory, nil, "rev-parse", "--show-toplevel", "HEAD", "HEAD^{tree}")
	if err != nil {
		return Repository{}, errors.New("goilerplate update requires a Git repository with at least one commit")
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		return Repository{}, errors.New("Git returned an unexpected repository state")
	}
	root := strings.TrimSpace(lines[0])
	status, err := runGit(ctx, root, nil, "status", "--porcelain=v1", "--untracked-files=normal")
	if err != nil {
		return Repository{}, fmt.Errorf("check Git worktree: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		return Repository{}, errors.New("goilerplate update requires a clean Git worktree")
	}
	return Repository{Root: root, Head: strings.TrimSpace(lines[1]), HeadTree: strings.TrimSpace(lines[2])}, nil
}

func Merge(ctx context.Context, repository Repository, oldRoot, newRoot, targetVersion string) (Result, error) {
	if !safeVersion(targetVersion) {
		return Result{}, fmt.Errorf("service returned invalid template version %q", targetVersion)
	}
	branch := "goilerplate-update-" + targetVersion
	oldCommit, newCommit, err := importTemplates(ctx, repository.Root, oldRoot, newRoot)
	if err != nil {
		return Result{}, err
	}
	bridge, err := runGit(ctx, repository.Root, strings.NewReader("goilerplate update bridge\n"),
		"commit-tree", repository.HeadTree, "-p", oldCommit)
	if err != nil {
		return Result{}, fmt.Errorf("prepare Git merge: %w", err)
	}
	mergedTree, conflicts, err := mergeTree(ctx, repository.Root, strings.TrimSpace(bridge), newCommit)
	if err != nil {
		return Result{}, err
	}
	commit, err := runGit(ctx, repository.Root, strings.NewReader("Update goilerplate template to "+targetVersion+"\n"),
		"commit-tree", mergedTree, "-p", repository.Head)
	if err != nil {
		return Result{}, fmt.Errorf("create update commit: %w", err)
	}
	zero := strings.Repeat("0", len(repository.Head))
	if _, err := runGit(ctx, repository.Root, nil, "update-ref", "refs/heads/"+branch, strings.TrimSpace(commit), zero); err != nil {
		return Result{}, fmt.Errorf("create update branch %s: it may already exist", branch)
	}
	updated, err := changedFiles(oldRoot, newRoot, conflicts)
	if err != nil {
		return Result{}, err
	}
	return Result{Branch: branch, Updated: updated, Conflicts: conflicts}, nil
}

func importTemplates(ctx context.Context, repositoryRoot, oldRoot, newRoot string) (string, string, error) {
	oldFiles, err := treeFiles(oldRoot)
	if err != nil {
		return "", "", err
	}
	newFiles, err := treeFiles(newRoot)
	if err != nil {
		return "", "", err
	}
	command := exec.CommandContext(ctx, "git", "fast-import", "--quiet", "--done")
	command.Dir = repositoryRoot
	stdin, err := command.StdinPipe()
	if err != nil {
		return "", "", err
	}
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		return "", "", fmt.Errorf("start Git template import: %w", err)
	}
	writeErr := writeTemplateCommits(stdin, oldRoot, oldFiles, newRoot, newFiles)
	closeErr := stdin.Close()
	waitErr := command.Wait()
	if writeErr != nil {
		return "", "", writeErr
	}
	if closeErr != nil {
		return "", "", closeErr
	}
	if waitErr != nil {
		return "", "", fmt.Errorf("import template trees: %s", strings.TrimSpace(stderr.String()))
	}
	marks := strings.Fields(stdout.String())
	if len(marks) != 2 {
		return "", "", errors.New("Git returned unexpected template commit IDs")
	}
	return marks[0], marks[1], nil
}

func writeTemplateCommits(destination io.Writer, oldRoot string, oldFiles []string, newRoot string, newFiles []string) error {
	writer := bufio.NewWriter(destination)
	_, _ = io.WriteString(writer, "feature done\nreset refs/goilerplate/update-import\n\n")
	if err := writeTemplateCommit(writer, oldRoot, oldFiles, ":1", "", "old goilerplate template"); err != nil {
		return err
	}
	if err := writeTemplateCommit(writer, newRoot, newFiles, ":2", ":1", "new goilerplate template"); err != nil {
		return err
	}
	_, _ = io.WriteString(writer, "get-mark :1\nget-mark :2\nreset refs/goilerplate/update-import\n\ndone\n")
	return writer.Flush()
}

func writeTemplateCommit(writer *bufio.Writer, root string, files []string, mark, parent, message string) error {
	fmt.Fprintln(writer, "commit refs/goilerplate/update-import")
	fmt.Fprintln(writer, "mark", mark)
	fmt.Fprintln(writer, "author goilerplate <noreply@goilerplate.com> 0 +0000")
	fmt.Fprintln(writer, "committer goilerplate <noreply@goilerplate.com> 0 +0000")
	fmt.Fprintf(writer, "data %d\n%s\n", len(message), message)
	if parent != "" {
		fmt.Fprintln(writer, "from", parent)
	}
	fmt.Fprintln(writer, "deleteall")
	for _, name := range files {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return fmt.Errorf("read template file %s: %w", name, err)
		}
		fmt.Fprintln(writer, "M 100644 inline", quotePath(name))
		fmt.Fprintln(writer, "data", len(content))
		if _, err := writer.Write(content); err != nil {
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			return err
		}
	}
	fmt.Fprintln(writer)
	return nil
}

func mergeTree(ctx context.Context, root, customerCommit, newCommit string) (string, []string, error) {
	command := exec.CommandContext(ctx, "git", "merge-tree", "--write-tree", "--name-only", "--messages", customerCommit, newCommit)
	command.Dir = root
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) || exitError.ExitCode() != 1 {
			return "", nil, fmt.Errorf("merge template update: %s", strings.TrimSpace(stderr.String()))
		}
	}
	lines := strings.Split(strings.ReplaceAll(stdout.String(), "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return "", nil, errors.New("Git returned no merged tree")
	}
	var conflicts []string
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		conflicts = append(conflicts, line)
	}
	return strings.TrimSpace(lines[0]), uniqueSorted(conflicts), nil
}

func changedFiles(oldRoot, newRoot string, conflicts []string) ([]string, error) {
	oldFiles, err := fileDigests(oldRoot)
	if err != nil {
		return nil, err
	}
	newFiles, err := fileDigests(newRoot)
	if err != nil {
		return nil, err
	}
	conflictSet := make(map[string]bool, len(conflicts))
	for _, name := range conflicts {
		conflictSet[name] = true
	}
	all := make(map[string]bool, len(oldFiles)+len(newFiles))
	for name := range oldFiles {
		all[name] = true
	}
	for name := range newFiles {
		all[name] = true
	}
	var changed []string
	for name := range all {
		if oldFiles[name] != newFiles[name] && !conflictSet[name] {
			changed = append(changed, name)
		}
	}
	sort.Strings(changed)
	return changed, nil
}

func fileDigests(root string) (map[string][sha256.Size]byte, error) {
	files, err := treeFiles(root)
	if err != nil {
		return nil, err
	}
	digests := make(map[string][sha256.Size]byte, len(files))
	for _, name := range files {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return nil, err
		}
		digests[name] = sha256.Sum256(content)
	}
	return digests, nil
}

func treeFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("template contains unsupported file %s", name)
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read template tree: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func quotePath(value string) string {
	var quoted strings.Builder
	quoted.WriteByte('"')
	for _, value := range []byte(value) {
		switch {
		case value == '\\' || value == '"':
			quoted.WriteByte('\\')
			quoted.WriteByte(value)
		case value < 32 || value >= 127:
			fmt.Fprintf(&quoted, "\\%03o", value)
		default:
			quoted.WriteByte(value)
		}
	}
	quoted.WriteByte('"')
	return quoted.String()
}

func safeVersion(value string) bool {
	if len(value) < 2 || value[0] != 'v' {
		return false
	}
	for _, character := range value[1:] {
		if character >= '0' && character <= '9' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character == '.' || character == '-' || character == '+' {
			continue
		}
		return false
	}
	return true
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func runGit(ctx context.Context, directory string, input io.Reader, arguments ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", arguments...)
	command.Dir = directory
	command.Stdin = input
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", errors.New(message)
	}
	return stdout.String(), nil
}
