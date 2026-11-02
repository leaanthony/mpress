package media

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mk(kind string, payload ...[]byte) []byte {
	var body []byte
	for _, p := range payload {
		body = append(body, p...)
	}
	out := make([]byte, 8, 8+len(body))
	binary.BigEndian.PutUint32(out, uint32(8+len(body)))
	copy(out[4:], kind)
	return append(out, body...)
}

// fakeMP4 builds the minimal box tree the probe reads.
func fakeMP4(fourcc string) []byte {
	entry := make([]byte, 78)
	binary.BigEndian.PutUint16(entry[24:], 1920)
	binary.BigEndian.PutUint16(entry[26:], 1080)
	stsd := mk("stsd", make([]byte, 8), mk(fourcc, entry))
	hdlr := mk("hdlr", make([]byte, 8), []byte("vide"), make([]byte, 12))
	trak := mk("trak", mk("mdia", hdlr, mk("minf", mk("stbl", stsd))))
	return append(mk("ftyp", []byte("isom\x00\x00\x00\x00")), mk("moov", trak)...)
}

func TestProbeCodecs(t *testing.T) {
	for fourcc, want := range map[string]string{"avc1": "h264", "av01": "av1", "hvc1": "hevc", "vp09": "vp9"} {
		path := filepath.Join(t.TempDir(), "v.mp4")
		os.WriteFile(path, fakeMP4(fourcc), 0o644)
		info, err := ProbeFile(path)
		if err != nil || info.Codec != want || info.Width != 1920 || info.Height != 1080 {
			t.Errorf("%s: %+v %v", fourcc, info, err)
		}
	}
}

func TestAuditRequiresUniversalSource(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "av1.mp4"), fakeMP4("av01"), 0o644)
	os.WriteFile(filepath.Join(dir, "h264.mp4"), fakeMP4("avc1"), 0o644)
	only := `<video><source src="/av1.mp4" type="video/mp4"></video>`
	both := `<video><source src="/av1.mp4"><source src="h264.mp4"></video>`
	remote := `<video><source src="https://example.com/x.mp4"></video>`
	if got := AuditHTML(dir, "blog/p/index.html", []byte(only)); len(got) != 1 {
		t.Errorf("AV1-only should warn: %v", got)
	}
	if got := AuditHTML(dir, "index.html", []byte(both)); len(got) != 0 {
		t.Errorf("H.264 fallback should pass: %v", got)
	}
	if got := AuditHTML(dir, "index.html", []byte(remote)); len(got) != 0 {
		t.Errorf("remote source cannot be judged: %v", got)
	}
}

func TestRemediateWithFakeFFmpeg(t *testing.T) {
	bin := t.TempDir()
	// A stand-in ffmpeg that writes an H.264 file to its last argument.
	script := "#!/bin/sh\nfor a; do last=$a; done\ncat \"$H264\" > \"$last\"\n"
	os.WriteFile(filepath.Join(bin, "ffmpeg"), []byte(script), 0o755)
	out := t.TempDir()
	h264 := filepath.Join(t.TempDir(), "h.mp4")
	os.WriteFile(h264, fakeMP4("avc1"), 0o644)
	t.Setenv("H264", h264)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	os.WriteFile(filepath.Join(out, "a.mp4"), fakeMP4("av01"), 0o644)
	page := `<video><source src="/a.mp4" type="video/mp4"></video>`
	os.WriteFile(filepath.Join(out, "index.html"), []byte(page), 0o644)

	findings, _ := AuditTree(out)
	added, remaining := Remediate(out, findings, true)
	if len(added) != 1 || len(remaining) != 0 {
		t.Fatalf("added=%v remaining=%v", added, remaining)
	}
	if again, _ := AuditTree(out); len(again) != 0 {
		t.Errorf("page still flagged: %v", again)
	}
	got, _ := os.ReadFile(filepath.Join(out, "index.html"))
	if want := `<source src="/a.mp4" type="video/mp4"><source src="/a-h264.mp4" type="video/mp4">`; !strings.Contains(string(got), want) {
		t.Errorf("html = %s", got)
	}
}

func TestRemediateWithoutFFmpegGivesHint(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	out := t.TempDir()
	os.WriteFile(filepath.Join(out, "a.mp4"), fakeMP4("av01"), 0o644)
	os.WriteFile(filepath.Join(out, "index.html"), []byte(`<video><source src="/a.mp4"></video>`), 0o644)
	findings, _ := AuditTree(out)
	_, remaining := Remediate(out, findings, true)
	if len(remaining) != 1 || !strings.Contains(remaining[0].Message, "https://ffmpeg.org/download.html") {
		t.Errorf("remaining=%v", remaining)
	}
}
