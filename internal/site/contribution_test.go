package site

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestShellInstallerDownloadsVerifiesAndRunsLatestRelease(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell test")
	}
	for _, scenario := range []struct {
		name        string
		exit        int
		badChecksum bool
	}{
		{name: "curl"},
		{name: "wget fallback"},
		{name: "contribution fails", exit: 23},
		{name: "checksum mismatch", exit: 1, badChecksum: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root := t.TempDir()
			bin := filepath.Join(root, "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			archiveContents := "M-Press test archive"
			checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(archiveContents)))
			if scenario.badChecksum {
				checksum = strings.Repeat("0", 64)
			}
			writeExecutable(t, filepath.Join(bin, "uname"), `#!/bin/sh
if [ "${1:-}" = "-s" ]; then printf 'Linux\n'; else printf 'x86_64\n'; fi
`)
			downloaderBody := fmt.Sprintf(`#!/bin/sh
destination=
source_url=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output) destination=$2; shift 2 ;;
    --output-document=*) destination=${1#*=}; shift ;;
    *) source_url=$1; shift ;;
  esac
done
printf '%%s\n' "$source_url" >> "$MPRESS_TEST_LOG"
case "$source_url" in
  */checksums.txt) printf '%s  mpress-linux-amd64.tar.gz\n' > "$destination" ;;
  *) printf '%s' > "$destination" ;;
esac
`, checksum, archiveContents)
			if scenario.name == "wget fallback" {
				writeExecutable(t, filepath.Join(bin, "curl"), "#!/bin/sh\nexit 1\n")
				writeExecutable(t, filepath.Join(bin, "wget"), downloaderBody)
			} else {
				writeExecutable(t, filepath.Join(bin, "curl"), downloaderBody)
			}
			writeExecutable(t, filepath.Join(bin, "tar"), `#!/bin/sh
destination=
while [ "$#" -gt 0 ]; do
  if [ "$1" = "-C" ]; then destination=$2; break; fi
  shift
done
printf '#!/bin/sh\nprintf "%%s\\n" "$@"\nexit "${MPRESS_TEST_EXIT:-0}"\n' > "$destination/mpress"
`)

			installer := filepath.Join(root, "mpress-contribute.sh")
			if err := os.WriteFile(installer, []byte(renderContributionInstallShell("https://github.com/example/docs.git", "next")), 0o755); err != nil {
				t.Fatal(err)
			}
			logPath := filepath.Join(root, "downloads.log")
			temporary := filepath.Join(root, "temporary")
			if err := os.Mkdir(temporary, 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("sh", installer)
			command.Env = append(os.Environ(), "PATH="+bin+":/usr/bin:/bin", "MPRESS_RELEASE_BASE_URL=https://releases.example.test", "MPRESS_TEST_LOG="+logPath, "TMPDIR="+temporary, fmt.Sprintf("MPRESS_TEST_EXIT=%d", scenario.exit))
			output, err := command.CombinedOutput()
			if command.ProcessState == nil || command.ProcessState.ExitCode() != scenario.exit {
				t.Fatalf("installer failed: %v\n%s", err, output)
			}
			want := "contribute\n--branch\nnext\nhttps://github.com/example/docs.git"
			if scenario.badChecksum {
				want = "Checksum verification failed for mpress-linux-amd64.tar.gz."
			}
			if got := strings.TrimSpace(string(output)); got != want {
				t.Fatalf("installer arguments = %q", got)
			}
			entries, err := os.ReadDir(temporary)
			if err != nil || len(entries) != 0 {
				t.Fatalf("temporary downloads remain: %v, %v", entries, err)
			}
			downloads, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"https://releases.example.test/mpress-linux-amd64.tar.gz", "https://releases.example.test/checksums.txt"} {
				if !strings.Contains(string(downloads), want) {
					t.Errorf("download log missing %q: %s", want, downloads)
				}
			}
		})
	}
}

func TestContributionInstallersRequireSHA256Verification(t *testing.T) {
	installers := map[string]string{
		"shell":      renderContributionInstallShell("https://github.com/example/docs.git", "main"),
		"PowerShell": renderContributionInstallPowerShell("https://github.com/example/docs.git", "main"),
	}
	for name, script := range installers {
		for _, want := range []string{"checksums.txt", "SHA256", "Checksum verification failed", "releases/latest/download", "contribute"} {
			if !strings.Contains(strings.ToLower(script), strings.ToLower(want)) {
				t.Errorf("%s installer missing %q", name, want)
			}
		}
	}
	if !strings.Contains(installers["shell"], "command -v mpress") {
		t.Fatal("shell installer does not check for an installed M-Press executable")
	}
	if !strings.Contains(installers["PowerShell"], "Get-Command mpress") {
		t.Fatal("PowerShell installer does not check for an installed M-Press executable")
	}
}

func TestShellInstallerUsesInstalledMPressWithoutDownloading(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell test")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(bin, "mpress"), `#!/bin/sh
printf '%s\n' "$@"
`)
	writeExecutable(t, filepath.Join(bin, "curl"), `#!/bin/sh
echo "curl must not run" >&2
exit 97
`)
	installer := filepath.Join(root, "mpress-contribute.sh")
	if err := os.WriteFile(installer, []byte(renderContributionInstallShell("https://github.com/example/docs.git", "next")), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("sh", installer)
	command.Env = append(os.Environ(), "PATH="+bin+":/usr/bin:/bin")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("installer failed: %v\n%s", err, output)
	}
	if got := strings.TrimSpace(string(output)); got != "contribute\n--branch\nnext\nhttps://github.com/example/docs.git" {
		t.Fatalf("installed M-Press arguments = %q", got)
	}
}

func TestContributionInstallersQuoteGeneratedConfiguration(t *testing.T) {
	shell := renderContributionInstallShell("https://example.test/lea's-docs.git", "release candidate")
	if !strings.Contains(shell, `target='https://example.test/lea'"'"'s-docs.git'`) || !strings.Contains(shell, `--branch 'release candidate' "$target"`) {
		t.Fatalf("shell installer does not safely quote configuration:\n%s", shell)
	}
	powerShell := renderContributionInstallPowerShell("https://example.test/lea's-docs.git", "release candidate")
	if !strings.Contains(powerShell, `else { 'https://example.test/lea''s-docs.git' }`) || !strings.Contains(powerShell, `@("contribute", "--branch", 'release candidate', $target)`) {
		t.Fatalf("PowerShell installer does not safely quote configuration:\n%s", powerShell)
	}
}

func TestShellInstallerForwardsPageAndBrowserDraftFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell test")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(bin, "mpress"), "#!/bin/sh\nprintf '%s\\n' \"$@\"\n")
	installer := filepath.Join(root, "mpress-contribute.sh")
	if err := os.WriteFile(installer, []byte(renderContributionInstallShell("https://github.com/example/docs.git", "next")), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("sh", installer, "https://docs.example.test/guide/", "change.mpress-draft", "translate")
	command.Env = append(os.Environ(), "PATH="+bin+":/usr/bin:/bin")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("installer failed: %v\n%s", err, output)
	}
	want := "contribute\n--branch\nnext\nhttps://docs.example.test/guide/\n--goal\ntranslate\n--draft-file\nchange.mpress-draft"
	if got := strings.TrimSpace(string(output)); got != want {
		t.Fatalf("installed M-Press arguments = %q, want %q", got, want)
	}
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestShellInstallerForwardsContributorOptions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell test")
	}
	root := t.TempDir()
	writeExecutable(t, filepath.Join(root, "mpress"), "#!/bin/sh\nprintf '%s\\n' \"$@\"\n")
	installer := filepath.Join(root, "install.sh")
	writeExecutable(t, installer, renderContributionInstallShell("https://example.test/docs.git", "main"))
	for _, args := range [][]string{
		{"--goal", "translate", "--checkout", "my docs; $(not-a-command)", "--no-open", "--port=4567"},
		{"https://example.test/page/", "--goal", "translate", "--checkout", "my docs; $(not-a-command)"},
		{"https://example.test/page/", "", "translate", "--no-open"},
	} {
		command := exec.Command("sh", append([]string{installer}, args...)...)
		command.Env = append(os.Environ(), "PATH="+root+":/usr/bin:/bin")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("installer failed: %v\n%s", err, output)
		}
		for _, arg := range args {
			if arg != "" && !strings.Contains(string(output), arg+"\n") {
				t.Errorf("argument %q missing from %q", arg, output)
			}
		}
		if strings.Contains(string(output), "--draft-file\n--goal") {
			t.Fatalf("option interpreted as draft: %s", output)
		}
	}
}

func TestShellInstallerHelpAndMissingGit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell test")
	}
	root := t.TempDir()
	installer := filepath.Join(root, "install.sh")
	writeExecutable(t, installer, renderContributionInstallShell("https://example.test/docs.git", "main"))
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Fatal(err)
	}
	for _, help := range []bool{true, false} {
		args := []string{installer}
		if help {
			args = append(args, "--help")
		}
		command := exec.Command(shell, args...)
		command.Env = append(os.Environ(), "PATH="+root)
		output, err := command.CombinedOutput()
		if help {
			if err != nil || !strings.Contains(string(output), "--goal translate") {
				t.Fatalf("help: %v\n%s", err, output)
			}
		} else if err == nil || !strings.Contains(string(output), "Git is required") {
			t.Fatalf("missing Git: %v\n%s", err, output)
		}
	}
}

func TestPowerShellInstallerForwardsContributorOptions(t *testing.T) {
	powershell, err := exec.LookPath("pwsh")
	if err != nil {
		t.Skip("PowerShell is not installed")
	}
	root := t.TempDir()
	binary := filepath.Join(root, "mpress")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	source := filepath.Join(root, "main.go")
	writeExecutable(t, source, `package main
import ("fmt"; "os")
func main() { for _, arg := range os.Args[1:] { fmt.Println(arg) }; os.Exit(23) }
`)
	build := exec.Command("go", "build", "-o", binary, source)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fixture: %v\n%s", err, output)
	}
	installer := filepath.Join(root, "install.ps1")
	writeExecutable(t, installer, renderContributionInstallPowerShell("https://example.test/docs.git", "main"))
	command := exec.Command(powershell, "-NoProfile", "-File", installer, "--goal", "translate", "--checkout", "my docs; $(not-a-command)", "--no-open")
	command.Env = append(os.Environ(), "PATH="+root+string(os.PathListSeparator)+os.Getenv("PATH"))
	output, err := command.CombinedOutput()
	if command.ProcessState == nil || command.ProcessState.ExitCode() != 23 {
		t.Fatalf("exit status: %v\n%s", err, output)
	}
	want := "contribute\n--branch\nmain\nhttps://example.test/docs.git\n--goal\ntranslate\n--checkout\nmy docs; $(not-a-command)\n--no-open"
	if got := strings.TrimSpace(strings.ReplaceAll(string(output), "\r\n", "\n")); got != want {
		t.Fatalf("arguments = %q, want %q", got, want)
	}
	command = exec.Command(powershell, "-NoProfile", "-File", installer, "--help")
	output, err = command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "--goal translate") {
		t.Fatalf("help: %v\n%s", err, output)
	}
}
