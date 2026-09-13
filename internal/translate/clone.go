package translate

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneProject creates a local checkout for translation. Git authentication is
// delegated to the user's existing credential helper or gh configuration.
func CloneProject(ctx context.Context, repository, branch, destination string) (string, error) {
	repository = strings.TrimSpace(repository)
	destination = strings.TrimSpace(destination)
	if repository == "" || destination == "" {
		return "", errors.New("repository and checkout directory are required")
	}
	if parsed, parseErr := url.Parse(repository); parseErr == nil && parsed.User != nil {
		return "", errors.New("repository URLs must not contain credentials; use the Git credential helper or gh")
	}
	abs, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	if info, statErr := os.Stat(abs); statErr == nil {
		if !info.IsDir() {
			return "", fmt.Errorf("checkout destination %s already exists", abs)
		}
		entries, readErr := os.ReadDir(abs)
		if readErr != nil {
			return "", readErr
		}
		if len(entries) != 0 {
			return "", fmt.Errorf("checkout destination %s is not empty", abs)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", statErr
	}
	args := []string{"clone", "--filter=blob:none"}
	if strings.TrimSpace(branch) != "" {
		args = append(args, "--branch", strings.TrimSpace(branch), "--single-branch")
	}
	args = append(args, "--", repository, abs)
	command := exec.CommandContext(ctx, "git", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		// Git output can contain a credential-bearing repository URL. Keep it out
		// of the returned error and let Git's credential helper report details in
		// the invoking terminal when needed.
		return "", fmt.Errorf("git clone failed: %w", err)
	}
	if _, err := os.Stat(filepath.Join(abs, configFilename)); err != nil {
		return "", fmt.Errorf("cloned repository does not contain %s", configFilename)
	}
	_ = output
	return abs, nil
}

const configFilename = "mpress.yaml"
