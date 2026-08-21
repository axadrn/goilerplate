package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/axadrn/goilerplate/v3/internal/doctor"
)

func (a *App) doctor(ctx context.Context, arguments []string) error {
	if len(arguments) != 0 {
		return errors.New("usage: goilerplate doctor")
	}
	directory := a.WorkingDirectory
	if directory == "" {
		var err error
		directory, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("find working directory: %w", err)
		}
	}
	if a.RunDoctor == nil {
		return errors.New("doctor is not configured")
	}
	report := a.RunDoctor(ctx, directory)
	for _, check := range report.Checks {
		marker := "OK"
		if check.Level == doctor.LevelWarning {
			marker = "WARN"
		}
		if check.Level == doctor.LevelError {
			marker = "FAIL"
		}
		fmt.Fprintf(a.Out, "[%s] %s: %s\n", marker, check.Name, check.Message)
	}
	if report.Errors > 0 {
		return fmt.Errorf("doctor found %d problem(s)", report.Errors)
	}
	fmt.Fprintln(a.Out, "Ready to build")
	return nil
}
