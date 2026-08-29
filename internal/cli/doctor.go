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
	if a.StyledOutput {
		a.printHeading("doctor")
		fmt.Fprintln(a.Out)
	}
	theme := a.theme()
	for _, check := range report.Checks {
		marker := "OK"
		if check.Level == doctor.LevelWarning {
			marker = "WARN"
		}
		if check.Level == doctor.LevelError {
			marker = "FAIL"
		}
		if !a.StyledOutput {
			fmt.Fprintf(a.Out, "[%s] %s: %s\n", marker, check.Name, check.Message)
			continue
		}
		statusStyle := theme.success
		if check.Level == doctor.LevelWarning {
			statusStyle = theme.warning
		}
		if check.Level == doctor.LevelError {
			statusStyle = theme.danger
		}
		fmt.Fprintln(a.Out, statusStyle.Render("["+marker+"]")+" "+theme.value.Render(check.Name)+theme.muted.Render(": "+check.Message))
	}
	if report.Errors > 0 {
		return fmt.Errorf("doctor found %d problem(s)", report.Errors)
	}
	if a.StyledOutput {
		fmt.Fprintln(a.Out)
	}
	a.printSuccess("Ready to build")
	return nil
}
