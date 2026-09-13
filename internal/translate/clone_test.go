package translate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCloneProjectUsesLocalGitCheckout(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, configFilename), []byte("site:\n  title: Clone test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, source, "init")
	runGit(t, source, "add", configFilename)
	runGit(t, source, "-c", "user.name=M-Press Test", "-c", "user.email=test@example.invalid", "commit", "-m", "initial")
	destination := filepath.Join(t.TempDir(), "checkout")
	checkout, err := CloneProject(context.Background(), source, "", destination)
	if err != nil {
		t.Fatal(err)
	}
	if checkout != destination {
		t.Fatalf("checkout = %q", checkout)
	}
	if _, err := os.Stat(filepath.Join(checkout, configFilename)); err != nil {
		t.Fatal(err)
	}
}

func TestCloneProjectRefusesNonEmptyDestination(t *testing.T) {
	destination := t.TempDir()
	if err := os.WriteFile(filepath.Join(destination, "keep.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := CloneProject(context.Background(), "example.invalid/repo", "", destination); err == nil {
		t.Fatal("expected non-empty destination error")
	}
}

func TestCloneProjectRejectsCredentialsInRepositoryURL(t *testing.T) {
	if _, err := CloneProject(context.Background(), "https://token@example.com/private.git", "", filepath.Join(t.TempDir(), "checkout")); err == nil {
		t.Fatal("repository URL credentials were accepted")
	}
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
