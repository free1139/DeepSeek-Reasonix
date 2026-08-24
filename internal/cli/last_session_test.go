package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLastSessionReadMissing covers the no-pointer case: --continue must
// fall through to the mtime fast path, not return an error.
func TestLastSessionReadMissing(t *testing.T) {
	dir := t.TempDir()
	if path, ok := readLastSession(dir); ok || path != "" {
		t.Fatalf("missing pointer: got (%q, %v), want (\"\", false)", path, ok)
	}
}

// TestLastSessionRoundTrip exercises the happy path: write then read returns
// the same path, and a second write overwrites the previous one.
func TestLastSessionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "20260101-120000.000000000-deepseek-flash.jsonl")
	if err := writeLastSession(dir, first); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, ok := readLastSession(dir)
	if !ok || got != first {
		t.Fatalf("read after write: got (%q, %v), want (%q, true)", got, ok, first)
	}
	second := filepath.Join(dir, "20260102-120000.000000000-deepseek-flash.jsonl")
	if err := writeLastSession(dir, second); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	got, ok = readLastSession(dir)
	if !ok || got != second {
		t.Fatalf("read after rewrite: got (%q, %v), want (%q, true)", got, ok, second)
	}
}

// TestLastSessionReadIgnoresTrailingLines covers the on-disk shape: the
// pointer file may grow extra diagnostics later, but only the first non-empty
// line is consumed.
func TestLastSessionReadIgnoresTrailingLines(t *testing.T) {
	dir := t.TempDir()
	first := "/abs/path/session-a.jsonl"
	body := first + "\n# written by chatREPL\n# mtime=2026-01-02T12:00:00Z\n"
	if err := os.WriteFile(filepath.Join(dir, "last-session"), []byte(body), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	got, ok := readLastSession(dir)
	if !ok || got != first {
		t.Fatalf("got (%q, %v), want (%q, true)", got, ok, first)
	}
}

// TestLastSessionReadRejectsEmptyOrWhitespace ensures the helper treats a
// blank pointer as missing, not as a valid empty path.
func TestLastSessionReadRejectsEmptyOrWhitespace(t *testing.T) {
	for _, body := range []string{"", "\n", "   \n\t\n", "  \n# comment\n"} {
		t.Run(body, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "last-session"), []byte(body), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}
			if path, ok := readLastSession(dir); ok || path != "" {
				t.Fatalf("body=%q should yield (\"\", false), got (%q, %v)", body, path, ok)
			}
		})
	}
}

// TestLastSessionWriteSkipsMissingInputs is a defensive smoke test: passing
// an empty sessionDir or sessionPath must be a no-op, not a panic.
func TestLastSessionWriteSkipsMissingInputs(t *testing.T) {
	if err := writeLastSession("", "/abs/path/session.jsonl"); err != nil {
		t.Fatalf("empty dir: %v", err)
	}
	if err := writeLastSession(t.TempDir(), ""); err != nil {
		t.Fatalf("empty path: %v", err)
	}
}

// TestLastSessionPathLivesNextToTranscripts pins the on-disk location so the
// per-workspace pointer does not leak into the global state root.
func TestLastSessionPathLivesNextToTranscripts(t *testing.T) {
	dir := t.TempDir()
	got := lastSessionPath(dir)
	want := filepath.Join(dir, "last-session")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if lastSessionPath("") != "" {
		t.Fatalf("empty dir must yield empty path")
	}
}

// TestLastSessionContinuesPreferPointer over the mtime scan when both are
// present: with a healthy pointer file plus a newer mtime transcript on
// disk, readLastSession wins. This mirrors the --continue primary path.
func TestLastSessionContinuesPreferPointer(t *testing.T) {
	dir := buildDirMTime(t, 5, false, false)
	pointer := filepath.Join(dir, "20260101-120000.000000000-deepseek-flash.jsonl")
	if err := writeLastSession(dir, pointer); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, ok := readLastSession(dir)
	if !ok || got != pointer {
		t.Fatalf("pointer not preferred: got (%q, %v), want (%q, true)", got, ok, pointer)
	}
}
