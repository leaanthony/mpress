package dev

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func newWatchFixture(t *testing.T) (*Server, string) {
	t.Helper()
	root := authoringFixture(t)
	server, err := NewServer(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return server, root
}

func watchFingerprint(t *testing.T, server *Server) string {
	t.Helper()
	fingerprint, err := server.scanWatchState()
	if err != nil {
		t.Fatalf("scan watch state: %v", err)
	}
	return fingerprint
}

func TestWatchFingerprintDetectsPreservedTimestampContentChange(t *testing.T) {
	server, root := newWatchFixture(t)
	before := watchFingerprint(t, server)

	path := filepath.Join(root, "content", "index.md")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Keep the byte length identical and restore the original timestamp. A
	// max-modification-time watcher therefore sees no change at all.
	changed := []byte(string(original))
	copy(changed, []byte("Modified"))
	if err := os.WriteFile(path, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	restored, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !restored.ModTime().Equal(info.ModTime()) || restored.Size() != info.Size() {
		t.Fatalf("fixture did not preserve metadata: size %d->%d, mtime %s->%s", info.Size(), restored.Size(), info.ModTime(), restored.ModTime())
	}

	if after := watchFingerprint(t, server); after == before {
		t.Fatal("content change with preserved mtime and size was not detected")
	}
}

func TestWatchFingerprintTracksDeletionRenameAndRestore(t *testing.T) {
	server, root := newWatchFixture(t)
	original := filepath.Join(root, "content", "index.md")
	preserved := filepath.Join(root, "content", "preserved.md")
	if err := os.WriteFile(preserved, []byte("---\ntitle: Preserved\n---\n\n# Old page\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	baseline := watchFingerprint(t, server)

	if err := os.Remove(original); err != nil {
		t.Fatal(err)
	}
	deleted := watchFingerprint(t, server)
	if deleted == baseline {
		t.Fatal("deleted source was not detected")
	}

	if err := os.Rename(preserved, filepath.Join(root, "content", "renamed.md")); err != nil {
		t.Fatal(err)
	}
	renamed := watchFingerprint(t, server)
	if renamed == deleted {
		t.Fatal("renamed source was not detected")
	}

	if err := os.WriteFile(preserved, []byte("---\ntitle: Preserved\n---\n\n# Old page\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	restored := watchFingerprint(t, server)
	if restored == renamed {
		t.Fatal("restored source was not detected")
	}
}

func TestWatchFingerprintIncludesStaticConfigAndCustomCSS(t *testing.T) {
	server, root := newWatchFixture(t)
	baseline := watchFingerprint(t, server)

	writeDevFixture(t, root, "static/app.css", "body { color: black; }\n")
	withStatic := watchFingerprint(t, server)
	if withStatic == baseline {
		t.Fatal("new static asset was not detected")
	}

	writeDevFixture(t, root, "mpress.yaml", "site:\n  title: Renamed project\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n  navFile: _nav.yaml\n  customCSS: theme.css\n")
	writeDevFixture(t, root, "theme.css", ":root { --accent: red; }\n")
	withConfig := watchFingerprint(t, server)
	if withConfig == withStatic {
		t.Fatal("configuration change was not detected")
	}

	writeDevFixture(t, root, "theme.css", ":root { --accent: blue; }\n")
	withCSS := watchFingerprint(t, server)
	if withCSS == withConfig {
		t.Fatal("custom stylesheet change was not detected")
	}
}

func TestWatchFingerprintIgnoresGeneratedAndUnrelatedFiles(t *testing.T) {
	server, root := newWatchFixture(t)
	baseline := watchFingerprint(t, server)

	for _, name := range []string{
		"site/generated.html",
		".mpress/cache/state.json",
		".git/HEAD",
		"node_modules/dependency/index.js",
		"notes.txt",
	} {
		writeDevFixture(t, root, name, "ignored\n")
	}
	if after := watchFingerprint(t, server); after != baseline {
		t.Fatal("generated, cached or unrelated files changed the watch fingerprint")
	}
}

func TestWatchFingerprintFollowsSafeInternalAliases(t *testing.T) {
	server, root := newWatchFixture(t)
	target := filepath.Join(root, "content", "real.md")
	if err := os.WriteFile(target, []byte("---\ntitle: Real\n---\n\n# Alias target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	beforeAlias := watchFingerprint(t, server)
	devSymlink(t, "real.md", filepath.Join(root, "content", "alias.md"))
	withAlias := watchFingerprint(t, server)
	if withAlias == beforeAlias {
		t.Fatal("new internal alias was not detected")
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("---\ntitle: Real\n---\n\n# New target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(target, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if after := watchFingerprint(t, server); after == withAlias {
		t.Fatal("alias target content change was not detected")
	}
}

func TestWatchFingerprintReportsUnsafeAliasAndRecovers(t *testing.T) {
	server, root := newWatchFixture(t)
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.md")
	if err := os.WriteFile(secret, []byte("# Outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "content", "escape.md")
	devSymlink(t, secret, link)
	if _, err := server.scanWatchState(); err == nil {
		t.Fatal("unsafe external alias did not surface an explicit scan error")
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if _, err := server.scanWatchState(); err != nil {
		t.Fatalf("watch did not recover after removing unsafe alias: %v", err)
	}
}

func TestWatchRebuildsOnPreservedTimestampChange(t *testing.T) {
	server, root := newWatchFixture(t)
	server.rebuild(false)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.watch(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	// Let the watcher establish its initial baseline before mutating inputs.
	time.Sleep(200 * time.Millisecond)

	path := filepath.Join(root, "content", "index.md")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := []byte(string(original))
	copy(changed, []byte("Modified"))
	if err := os.WriteFile(path, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if server.currentState().Revision > 1 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("preview was not rebuilt after a preserved-mtime content change: %+v", server.currentState())
}

func TestWatchDoesNotRebuildForGeneratedOutputChurn(t *testing.T) {
	server, root := newWatchFixture(t)
	server.rebuild(false)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.watch(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	time.Sleep(200 * time.Millisecond)
	revision := server.currentState().Revision
	writeDevFixture(t, root, "site/extra-generated.html", "<html></html>\n")
	writeDevFixture(t, root, "notes.txt", "unrelated\n")
	time.Sleep(900 * time.Millisecond)
	if got := server.currentState().Revision; got != revision {
		t.Fatalf("generated or unrelated file triggered rebuild: revision %d -> %d", revision, got)
	}
}

func BenchmarkWatchScan1000Docs(b *testing.B) {
	root := b.TempDir()
	writeDevFixture(b, root, "mpress.yaml", "build:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n  navFile: _nav.yaml\n")
	for i := 0; i < 1000; i++ {
		path := filepath.Join("content", fmt.Sprintf("section-%02d", i%20), fmt.Sprintf("page-%04d.md", i))
		writeDevFixture(b, root, path, "---\ntitle: Page\n---\n\n# Heading\n\nBody text for the page.\n")
	}
	writeDevFixture(b, root, "static/.keep", "")
	server, err := NewServer(root, Options{})
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = server.Close() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := server.scanWatchState(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestWatchFingerprintHashesLargeFileMiddleWithoutTimestampChange(t *testing.T) {
	server, root := newWatchFixture(t)
	path := filepath.Join(root, "static", "large.bin")
	content := make([]byte, 5<<20)
	for index := range content {
		content[index] = byte(index % 251)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	baseline := watchFingerprint(t, server)

	edited := append([]byte(nil), content...)
	middle := len(edited) / 2
	edited[middle] ^= 0xff
	if err := os.WriteFile(path, edited, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	if after := watchFingerprint(t, server); after == baseline {
		t.Fatal("same-size middle edit of a large asset with preserved mtime was not detected")
	}
}

func TestWatchFingerprintTracksConfiguredNavigationOutsideContent(t *testing.T) {
	server, root := newWatchFixture(t)
	writeDevFixture(t, root, "project-nav.yaml", "- label: Home\n  link: /\n")
	writeDevFixture(t, root, "content/project-nav.yaml", "- label: Accueil\n  link: /\n")
	writeDevFixture(t, root, "mpress.yaml", "site:\n  title: Test project\n  languages:\n    - en\n    - fr\nbuild:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n  navFile: ../project-nav.yaml\n")
	baseline := watchFingerprint(t, server)

	writeDevFixture(t, root, "project-nav.yaml", "- label: Start\n  link: /\n")
	outside := watchFingerprint(t, server)
	if outside == baseline {
		t.Fatal("configured navigation outside the content directory was not watched")
	}

	writeDevFixture(t, root, "content/project-nav.yaml", "- label: Début\n  link: /\n")
	if localized := watchFingerprint(t, server); localized == outside {
		t.Fatal("localized navigation was not watched")
	}
}

func TestWatchLoopRebuildsAfterInitialScanError(t *testing.T) {
	scanError := fmt.Errorf("scan unavailable")
	results := []struct {
		value string
		err   error
	}{
		{"", scanError},
		{"", scanError},
		{"recovered", nil},
		{"recovered", nil},
	}
	var calls atomic.Int32
	scan := func() (string, error) {
		index := int(calls.Add(1)) - 1
		if index >= len(results) {
			index = len(results) - 1
		}
		return results[index].value, results[index].err
	}
	var rebuilds atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		watchLoop(ctx, time.Millisecond, scan, func() { rebuilds.Add(1) })
	}()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("watch loop did not stop after cancellation")
		}
	}()

	deadline := time.Now().Add(2 * time.Second)
	for rebuilds.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if rebuilds.Load() != 1 {
		t.Fatalf("recovery rebuilds = %d, want 1", rebuilds.Load())
	}
	time.Sleep(20 * time.Millisecond)
	if rebuilds.Load() != 1 {
		t.Fatalf("recovery rebuilt more than once: %d", rebuilds.Load())
	}
}

func TestWatchLoopCoalescesBurst(t *testing.T) {
	sequence := []string{"baseline", "first", "second", "second", "second"}
	var calls atomic.Int32
	scan := func() (string, error) {
		index := int(calls.Add(1)) - 1
		if index >= len(sequence) {
			index = len(sequence) - 1
		}
		return sequence[index], nil
	}
	var rebuilds atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		watchLoop(ctx, time.Millisecond, scan, func() { rebuilds.Add(1) })
	}()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("watch loop did not stop after cancellation")
		}
	}()

	deadline := time.Now().Add(2 * time.Second)
	for rebuilds.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if rebuilds.Load() != 1 {
		t.Fatalf("burst rebuilds = %d, want 1", rebuilds.Load())
	}
	time.Sleep(20 * time.Millisecond)
	if rebuilds.Load() != 1 {
		t.Fatalf("burst rebuilt repeatedly: %d", rebuilds.Load())
	}
}

func TestWatchLoopCancellationStopsWorker(t *testing.T) {
	var scans atomic.Int32
	var rebuilds atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		watchLoop(ctx, time.Millisecond, func() (string, error) {
			scans.Add(1)
			return "stable", nil
		}, func() { rebuilds.Add(1) })
	}()

	deadline := time.Now().Add(2 * time.Second)
	for scans.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watch loop did not stop after cancellation")
	}
	stopped := scans.Load()
	time.Sleep(20 * time.Millisecond)
	if scans.Load() != stopped || rebuilds.Load() != 0 {
		t.Fatalf("watch loop kept working after cancellation: scans %d -> %d, rebuilds %d", stopped, scans.Load(), rebuilds.Load())
	}
}

func BenchmarkWatchScan1000Docs8MiBAsset(b *testing.B) {
	root := b.TempDir()
	writeDevFixture(b, root, "mpress.yaml", "build:\n  contentDir: content\n  staticDir: static\n  outputDir: site\n  navFile: _nav.yaml\n")
	for i := 0; i < 1000; i++ {
		path := filepath.Join("content", fmt.Sprintf("section-%02d", i%20), fmt.Sprintf("page-%04d.md", i))
		writeDevFixture(b, root, path, "---\ntitle: Page\n---\n\n# Heading\n\nBody text for the page.\n")
	}
	asset := make([]byte, 8<<20)
	for index := range asset {
		asset[index] = byte(index % 251)
	}
	writeDevFixture(b, root, "static/large.bin", string(asset))
	server, err := NewServer(root, Options{})
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = server.Close() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := server.scanWatchState(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestWatchRebuildsAfterInitialScanErrorRecovery(t *testing.T) {
	server, root := newWatchFixture(t)
	server.rebuild(false)

	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.md")
	if err := os.WriteFile(secret, []byte("# External\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "content", "escape.md")
	devSymlink(t, secret, link)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.watch(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})

	// The initial baseline scan fails on the unsafe alias.
	time.Sleep(300 * time.Millisecond)
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(6 * time.Second)
	for time.Now().Before(deadline) {
		if server.currentState().Revision > 1 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("watch did not rebuild after recovering from an initial scan error: %+v", server.currentState())
}

func TestWatchFingerprintIncludesFilesUnderConfiguredInputDirectories(t *testing.T) {
	server, root := newWatchFixture(t)
	for _, name := range []string{"static/node_modules/widget.js", "content/node_modules/guide.md"} {
		before := watchFingerprint(t, server)
		writeDevFixture(t, root, name, "authored input")
		if watchFingerprint(t, server) == before {
			t.Fatalf("configured input %s was excluded because of its directory name", name)
		}
	}
}

func TestWatchFingerprintSeparatesBinaryFileContents(t *testing.T) {
	server, root := newWatchFixture(t)
	separator := "\x00file\x00static/b.bin\x00"
	writeDevFixture(t, root, "static/a.bin", "X")
	writeDevFixture(t, root, "static/b.bin", separator+"Y")
	before := watchFingerprint(t, server)
	writeDevFixture(t, root, "static/a.bin", "X"+separator)
	writeDevFixture(t, root, "static/b.bin", "Y")
	if watchFingerprint(t, server) == before {
		t.Fatal("binary contents were confused with the boundary between files")
	}
}
