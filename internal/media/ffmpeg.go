package media

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// encodeVersion invalidates cached conversions when the encode settings change.
const encodeVersion = "h264-high-crf27-aac128-v2"

// FFmpegPath returns the ffmpeg executable, or "" when it is not installed.
func FFmpegPath() string {
	p, err := exec.LookPath("ffmpeg")
	if err != nil {
		return ""
	}
	return p
}

// InstallHint says where to get ffmpeg on the given GOOS.
func InstallHint(goos string) string {
	switch goos {
	case "darwin":
		return "ffmpeg is required to convert videos automatically. Install it with `brew install ffmpeg` or from https://ffmpeg.org/download.html#build-mac"
	case "windows":
		return "ffmpeg is required to convert videos automatically. Install it with `winget install ffmpeg` or from https://ffmpeg.org/download.html#build-windows"
	default:
		return "ffmpeg is required to convert videos automatically. Install it with your package manager (for example `sudo apt install ffmpeg` or `sudo dnf install ffmpeg`) or see https://ffmpeg.org/download.html#build-linux"
	}
}

// ConvertH264 writes an H.264/AAC MP4 with the index up front (faststart),
// which plays on every current browser including iPad Safari.
func ConvertH264(ffmpeg, src, dst string) error {
	tmp := dst + ".part.mp4"
	defer os.Remove(tmp)
	cmd := exec.Command(ffmpeg, "-y", "-nostdin", "-v", "error", "-i", src,
		"-map", "0:v:0", "-map", "0:a:0?",
		"-c:v", "libx264", "-profile:v", "high", "-preset", "medium", "-crf", "27", "-pix_fmt", "yuv420p",
		"-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "Unknown encoder") || strings.Contains(msg, "libx264") {
			msg = "this ffmpeg build has no libx264 encoder; install a full build. " + msg
		}
		return fmt.Errorf("ffmpeg failed: %s", msg)
	}
	return os.Rename(tmp, dst)
}

func fileKey(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	h.Write([]byte(encodeVersion))
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:24], nil
}

// cachedConversion converts src once per content hash; reruns reuse the result.
func cachedConversion(ffmpeg, src string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "mpress", "video")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	key, err := fileKey(src)
	if err != nil {
		return "", err
	}
	cached := filepath.Join(dir, key+".mp4")
	if _, err := os.Stat(cached); err == nil {
		return cached, nil
	}
	if err := ConvertH264(ffmpeg, src, cached); err != nil {
		return "", err
	}
	return cached, nil
}

// Remediate adds an H.264 fallback <source> for each finding using ffmpeg.
// It returns output-relative paths of files it created and the findings that
// still need attention (with the reason appended). With convert false it only
// annotates findings, which is what development builds and `check` want.
func Remediate(output string, findings []Finding, convert bool) (added []string, remaining []Finding) {
	ffmpeg := FFmpegPath()
	converted := map[string]string{} // source file -> output-relative new file
	failed := map[string]string{}
	for _, f := range findings {
		note := ""
		switch {
		case ffmpeg == "":
			note = InstallHint(runtime.GOOS)
		case !convert:
			note = "run `mpress build` to convert it automatically with ffmpeg"
		default:
			if reason, bad := failed[f.File]; bad {
				note = reason
				break
			}
			newRel, done := converted[f.File]
			if !done {
				cached, err := cachedConversion(ffmpeg, f.File)
				if err != nil {
					failed[f.File] = err.Error()
					note = err.Error()
					break
				}
				dst := strings.TrimSuffix(f.File, filepath.Ext(f.File)) + "-h264.mp4"
				if err := copyFile(cached, dst); err != nil {
					failed[f.File] = err.Error()
					note = err.Error()
					break
				}
				newRel, _ = filepath.Rel(output, dst)
				newRel = filepath.ToSlash(newRel)
				converted[f.File] = newRel
				added = append(added, newRel)
			}
			if err := insertSource(output, f, path.Base(newRel)); err != nil {
				note = err.Error()
				break
			}
			continue // fixed
		}
		f.Message += " " + note
		remaining = append(remaining, f)
	}
	return added, remaining
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// insertSource places <source src=NEW type="video/mp4"> right after the
// <source> that references f.Source. NEW sits next to the original file.
func insertSource(output string, f Finding, newName string) error {
	page := filepath.Join(output, filepath.FromSlash(f.Page))
	data, err := os.ReadFile(page)
	if err != nil {
		return err
	}
	text := string(data)
	needle := `src="` + html.EscapeString(f.Source) + `"`
	at := strings.Index(text, needle)
	if at < 0 {
		return fmt.Errorf("could not locate the video source in %s to add the fallback", f.Page)
	}
	if !strings.HasSuffix(strings.TrimSpace(text[strings.LastIndex(text[:at], "<"):at]), "<source") {
		return fmt.Errorf("video source is set on the <video> tag itself; list it with <source> elements instead")
	}
	end := strings.Index(text[at:], ">")
	if end < 0 {
		return fmt.Errorf("malformed <source> in %s", f.Page)
	}
	dir := path.Dir(f.Source)
	newURL := newName
	if dir != "." {
		newURL = strings.TrimSuffix(dir, "/") + "/" + newName
	}
	insert := `<source src="` + html.EscapeString(newURL) + `" type="video/mp4">`
	text = text[:at+end+1] + insert + text[at+end+1:]
	return os.WriteFile(page, []byte(text), 0o644)
}
