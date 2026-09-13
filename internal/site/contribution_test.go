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
	for _, downloader := range []string{"curl", "wget fallback"} {
		t.Run(downloader, func(t *testing.T) {
			root := t.TempDir()
			bin := filepath.Join(root, "bin")
			if err := os.MkdirAll(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			archiveContents := "M-Press test archive"
			checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(archiveContents)))
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
			if downloader == "wget fallback" {
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
printf '#!/bin/sh\nprintf "%%s\\n" "$@"\n' > "$destination/mpress"
`)

			installer := filepath.Join(root, "mpress-contribute.sh")
			if err := os.WriteFile(installer, []byte(renderContributionInstallShell("https://github.com/example/docs.git", "next")), 0o755); err != nil {
				t.Fatal(err)
			}
			logPath := filepath.Join(root, "downloads.log")
			command := exec.Command("sh", installer)
			command.Env = append(os.Environ(), "PATH="+bin+":/usr/bin:/bin", "MPRESS_RELEASE_BASE_URL=https://releases.example.test", "MPRESS_TEST_LOG="+logPath)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("installer failed: %v\n%s", err, output)
			}
			if got := strings.TrimSpace(string(output)); got != "contribute\n--branch\nnext\nhttps://github.com/example/docs.git" {
				t.Fatalf("installer arguments = %q", got)
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
	want := "contribute\n--branch\nnext\n--goal\ntranslate\nhttps://docs.example.test/guide/\n--draft-file\nchange.mpress-draft"
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
