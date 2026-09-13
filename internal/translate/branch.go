package translate

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// PrepareWorkBranch moves a checkout onto a new, non-default branch before
// M-Press enables authoring tools. It is shared by the CLI and development UI
// so both entry points apply the same safety rule.
func PrepareWorkBranch(ctx context.Context, project, branch string) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return errors.New("a work branch name is required")
	}
	switch strings.ToLower(branch) {
	case "main", "master", "trunk":
		return errors.New("choose a work branch instead of the default branch")
	}
	if err := exec.CommandContext(ctx, "git", "check-ref-format", "--branch", branch).Run(); err != nil {
		return errors.New("the work branch name is not valid")
	}
	currentOutput, _ := exec.CommandContext(ctx, "git", "-C", project, "branch", "--show-current").Output()
	if strings.TrimSpace(string(currentOutput)) == branch {
		return nil
	}
	command := exec.CommandContext(ctx, "git", "-C", project, "switch", "-c", branch)
	if output, err := command.CombinedOutput(); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("could not create branch %q: %s", branch, message)
	}
	return nil
}
