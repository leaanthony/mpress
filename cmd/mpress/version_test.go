package main

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Exercise real Go build metadata: go test itself has no installed module version.
func TestCLIVersionBuildModes(t *testing.T) {
	root := t.TempDir()
	fixture, files := prepareVersionFixture(t, root)
	proxy := t.TempDir()
	server := httptest.NewServer(http.FileServer(http.Dir(proxy)))
	defer server.Close()
	for key, value := range map[string]string{
		"GOENV": "off", "GOWORK": "off", "GOTOOLCHAIN": "local", "GOFLAGS": "-modcacherw",
		"GOPROXY": server.URL, "GOPRIVATE": "", "GONOPROXY": "", "GONOSUMDB": "",
		// Only the synthetic example.com fixture is fetched from this local proxy.
		"GOSUMDB": "off", "GOMODCACHE": filepath.Join(root, "modcache"), "GOBIN": filepath.Join(root, "bin"),
	} {
		t.Setenv(key, value)
	}
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = fixture
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, output)
		}
		return strings.TrimSpace(string(output))
	}
	if got := run("go", "run", "."); got != strings.TrimSpace(sourceVersion) {
		t.Fatalf("source build version = %q, want embedded version.txt", got)
	}
	if got := run("go", "run", "-ldflags=-X main.version=9.9.9", "."); got != "9.9.9" {
		t.Fatalf("release build version = %q, want 9.9.9", got)
	}
	versions := []string{"v1.2.3", "v1.2.4-rc.1", "v1.2.4-0.20260913013901-ef3eac7f0654"}
	dir := filepath.Join(proxy, "example.com", "versioncheck", "@v")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("list", []byte("v1.2.3\n"))
	binary := filepath.Join(root, "bin", "versioncheck")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	for _, tag := range versions {
		writeVersionModule(t, files, tag, write)
		run("go", "install", "example.com/versioncheck@"+tag)
		if got := run(binary); got != strings.TrimPrefix(tag, "v") {
			t.Errorf("installed %s version = %q", tag, got)
		}
	}
	run("go", "install", "-ldflags=-X main.version=9.9.9", "example.com/versioncheck@v1.2.3")
	if got := run(binary); got != "9.9.9" {
		t.Fatalf("linker override with module metadata = %q, want 9.9.9", got)
	}
}

func writeVersionModule(t *testing.T, files map[string][]byte, tag string, write func(string, []byte)) {
	t.Helper()
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	for name, data := range files {
		entry, err := zw.Create("example.com/versioncheck@" + tag + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	write(tag+".zip", archive.Bytes())
	write(tag+".mod", files["go.mod"])
	write(tag+".info", []byte(`{"Version":"`+tag+`","Time":"2026-09-13T01:39:01Z"}`))
}

func prepareVersionFixture(t *testing.T, root string) (string, map[string][]byte) {
	t.Helper()
	fixture := filepath.Join(root, "source")
	if err := os.MkdirAll(fixture, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"go.mod":  []byte("module example.com/versioncheck\n\ngo 1.23\n"),
		"main.go": []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(cliVersion()) }\n"),
	}
	for _, name := range []string{"version.go", "version.txt"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = data
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(fixture, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return fixture, files
}
