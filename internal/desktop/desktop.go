package desktop

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type command struct {
	name string
	args []string
}

func OpenBrowser(ctx context.Context, address string) error {
	selected, err := browserCommand(runtime.GOOS, address)
	if err != nil {
		return err
	}
	return run(ctx, selected, "")
}

func CopyToClipboard(ctx context.Context, value string) error {
	clipboardContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for _, candidate := range clipboardCommands(runtime.GOOS) {
		if _, err := exec.LookPath(candidate.name); err != nil {
			continue
		}
		if err := run(clipboardContext, candidate, value); err == nil {
			return nil
		}
	}
	return fmt.Errorf("clipboard is unavailable on %s", runtime.GOOS)
}

func browserCommand(goos, address string) (command, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return command{}, errors.New("browser address is required")
	}
	switch goos {
	case "darwin":
		return command{name: "open", args: []string{address}}, nil
	case "linux":
		return command{name: "xdg-open", args: []string{address}}, nil
	default:
		return command{}, fmt.Errorf("opening a browser is unsupported on %s", goos)
	}
}

func clipboardCommands(goos string) []command {
	switch goos {
	case "darwin":
		return []command{{name: "pbcopy"}}
	case "linux":
		return []command{
			{name: "wl-copy"},
			{name: "xclip", args: []string{"-selection", "clipboard"}},
			{name: "xsel", args: []string{"--clipboard", "--input"}},
			{name: "clip.exe"},
		}
	default:
		return nil
	}
}

func run(ctx context.Context, selected command, input string) error {
	commandContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	process := exec.CommandContext(commandContext, selected.name, selected.args...)
	if input != "" {
		process.Stdin = strings.NewReader(input)
	}
	if err := process.Run(); err != nil {
		return fmt.Errorf("run %s: %w", selected.name, err)
	}
	return nil
}
