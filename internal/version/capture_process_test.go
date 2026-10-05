package version

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCaptureProcessHelper(t *testing.T) {
	root := os.Getenv("MPRESS_CAPTURE_PROCESS_ROOT")
	if root == "" {
		return
	}
	ready := os.Getenv("MPRESS_CAPTURE_PROCESS_READY")
	if err := os.WriteFile(ready, nil, 0600); err != nil {
		t.Fatal(err)
	}
	waitCaptureFile(t, filepath.Join(root, "start"))
	if err := Capture(root, "v1", true); err != nil {
		t.Fatal(err)
	}
}

func waitCaptureFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for capture process: %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCaptureCoordinatesSeparateProcesses(t *testing.T) {
	root := versionBoundaryFixture(t, ".mpress/versions")
	body := strings.Repeat("ordinary snapshot\n", 8192)
	writeVersionFile(t, filepath.Join(root, "site/index.html"), body)
	for i := range 32 {
		writeVersionFile(t, filepath.Join(root, "site", fmt.Sprintf("file-%d.html", i)), body)
	}
	var commands []*exec.Cmd
	outputs := map[*exec.Cmd]*bytes.Buffer{}
	for i := range 4 {
		ready := filepath.Join(root, "ready-"+string(rune('a'+i)))
		command := exec.Command(os.Args[0], "-test.run=^TestCaptureProcessHelper$", "-test.timeout=30s")
		command.Env = append(os.Environ(), "MPRESS_CAPTURE_PROCESS_ROOT="+root, "MPRESS_CAPTURE_PROCESS_READY="+ready)
		output := &bytes.Buffer{}
		command.Stdout, command.Stderr = output, output
		outputs[command] = output
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, command)
		t.Cleanup(func() { _ = command.Process.Kill() })
		waitCaptureFile(t, ready)
	}
	if err := os.WriteFile(filepath.Join(root, "start"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range commands {
		if err := command.Wait(); err != nil {
			t.Errorf("capture process failed: %v\n%s", err, outputs[command].String())
		}
	}
	if err := Verify(root, "v1"); err != nil {
		t.Fatalf("process concurrency corrupted the snapshot: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, ".mpress/versions/v1/index.html"))
	if err != nil || string(data) != body {
		t.Error("concurrent snapshot content changed")
	}
}
