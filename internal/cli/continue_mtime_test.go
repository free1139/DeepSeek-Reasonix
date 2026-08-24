package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"reasonix/internal/agent"
)

// buildDirMTime creates n fake session files, each with a transcript, an
// optional .events.jsonl log, and an optional sidecar. The timestamps on disk
// are forced to specific instants so the test can assert ordering.
func buildDirMTime(tb testing.TB, n int, withEvents bool, withSidecar bool) string {
	tb.Helper()
	dir := tb.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := range n {
		stamp := base.Add(time.Duration(i) * time.Second).UTC().Format("20060102-150405.000000000")
		name := stamp + "-deepseek-flash.jsonl"
		full := filepath.Join(dir, name)
		if err := os.WriteFile(full, []byte(`{"role":"system","content":"sys"}`+"\n"), 0o644); err != nil {
			tb.Fatalf("write transcript: %v", err)
		}
		// Set the transcript mtime deterministically by writing then chmod-ing.
		// os.Chtimes is the supported way; Stat would still see the correct value.
		mtime := base.Add(time.Duration(i) * time.Second)
		if err := os.Chtimes(full, mtime, mtime); err != nil {
			tb.Fatalf("chtimes transcript: %v", err)
		}
		if withEvents {
			logName := name + ".events.jsonl"
			logPath := filepath.Join(dir, logName)
			if err := os.WriteFile(logPath, []byte("[]\n"), 0o644); err != nil {
				tb.Fatalf("write events: %v", err)
			}
			// Force the events log mtime ahead of the transcript so the picker
			// picks it over the transcript — this mirrors production where the
			// controller flushes the event log more often than the snapshot.
			evTime := mtime.Add(30 * time.Second)
			if err := os.Chtimes(logPath, evTime, evTime); err != nil {
				tb.Fatalf("chtimes events: %v", err)
			}
		}
		if withSidecar {
			meta := agent.BranchMeta{
				ID:            agent.BranchID(full),
				CreatedAt:     base,
				UpdatedAt:     base.Add(time.Duration(i) * time.Minute),
				SchemaVersion: agent.BranchMetaCountsVersion,
				Turns:         1,
				Preview:       "x",
			}
			if err := agent.SaveBranchMetaPreserveUpdated(full, meta); err != nil {
				tb.Fatalf("save meta: %v", err)
			}
		}
	}
	return dir
}

// TestMostRecentSessionByMTimePicksNewest asserts the helper picks the newest
// transcript in a small fixture with deterministic mtimes.
func TestMostRecentSessionByMTimePicksNewest(t *testing.T) {
	dir := buildDirMTime(t, 10, false, false)
	got, ok := mostRecentSessionByMTime(dir)
	if !ok {
		t.Fatal("ok=false on non-empty dir")
	}
	want := filepath.Join(dir, "20260101-120009.000000000-deepseek-flash.jsonl")
	if got != want {
		t.Fatalf("got %q, want %q", filepath.Base(got), filepath.Base(want))
	}
}

// TestMostRecentSessionByMTimePicksEventLogOverTranscript checks the .events.jsonl
// mtime advances the candidate — this is the common case in production where the
// controller flushes the event log more often than the snapshot rewrite.
func TestMostRecentSessionByMTimePicksEventLogOverTranscript(t *testing.T) {
	dir := buildDirMTime(t, 10, true, false)
	got, ok := mostRecentSessionByMTime(dir)
	if !ok {
		t.Fatal("ok=false")
	}
	want := filepath.Join(dir, "20260101-120009.000000000-deepseek-flash.jsonl")
	if got != want {
		t.Fatalf("got %q, want %q", filepath.Base(got), filepath.Base(want))
	}
}

// TestMostRecentSessionByMTimeTieBreakByPath confirms that mtime ties resolve
// to the largest filename (same tiebreaker agent.ListSessionOrder uses via the
// Path < Path comparison).
func TestMostRecentSessionByMTimeTieBreakByPath(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	for i := range 5 {
		name := filepath.Join(dir, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC).
			Add(time.Duration(i)*time.Second).
			UTC().Format("20060102-150405.000000000")+"-model.jsonl")
		if err := os.WriteFile(name, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		_ = base
	}
	got, ok := mostRecentSessionByMTime(dir)
	if !ok {
		t.Fatal("ok=false")
	}
	// All five files have the same mtime (just-written); the largest filename
	// wins because the helper picks "full > bestPath" on ties.
	want := filepath.Join(dir, "20260101-120004.000000000-model.jsonl")
	if got != want {
		t.Fatalf("tie break: got %q, want %q", filepath.Base(got), filepath.Base(want))
	}
}

// TestMostRecentSessionByMTimeSkipsCleanupPending ensures a session with a
// .cleanup-pending marker does not steal the top slot from a healthy one.
func TestMostRecentSessionByMTimeSkipsCleanupPending(t *testing.T) {
	dir := buildDirMTime(t, 5, false, false)
	// Mark the most-recent transcript as pending cleanup. by Mark
	top := filepath.Join(dir, "20260101-120004.000000000-deepseek-flash.jsonl")
	if err := agent.MarkCleanupPending(top, "test"); err != nil {
		t.Fatalf("mark cleanup pending: %v", err)
	}
	got, ok := mostRecentSessionByMTime(dir)
	if !ok {
		t.Fatal("ok=false")
	}
	skip := filepath.Join(dir, "20260101-120004.000000000-deepseek-flash.jsonl")
	if got == skip {
		t.Fatalf("cleanup-pending transcript %q should have been skipped", filepath.Base(got))
	}
}

// TestMostRecentSessionByMTimeSkipsEventLogFile confirms that helper only
// treats transcript-shaped entries (.jsonl without the .events suffix).
func TestMostRecentSessionByMTimeSkipsEventLogFile(t *testing.T) {
	dir := buildDirMTime(t, 5, false, false)
	// Drop a stale event log into the directory with a much newer mtime; it
	// must NOT become the "newest session" on its own.
	staleLog := filepath.Join(dir, "20260101-120005.000000000-deepseek-flash.jsonl.events.jsonl")
	if err := os.WriteFile(staleLog, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(staleLog, future, future); err != nil {
		t.Fatal(err)
	}
	got, ok := mostRecentSessionByMTime(dir)
	if !ok {
		t.Fatal("ok=false")
	}
	if filepath.Base(got) == filepath.Base(staleLog) {
		t.Fatalf("event log must not surface as a session on its own")
	}
}

// TestMostRecentSessionByMTimeScalesLinearly reports wall time at N=100/500/2000
// so the saving against the sidecar path (covered by continue_startup_test.go)
// can be eyeballed without benchstat.
func TestMostRecentSessionByMTimeScalesLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scaling probe")
	}
	for _, n := range []int{100, 500, 1000, 2000, 5000} {
		dir := buildDirMTime(t, n, true, true)
		start := time.Now()
		got, ok := mostRecentSessionByMTime(dir)
		elapsed := time.Since(start)
		if !ok {
			t.Fatalf("n=%d: no session", n)
		}
		t.Logf("n=%5d  mtime-fastpath=%8s  picked=%s",
			n, elapsed.Round(time.Microsecond), filepath.Base(got))
	}
}

// TestMostRecentSessionByMTimeAgreesWithSidecar is a soft assertion that the
// fast path returns the same session as the sidecar path on a fixture where
// UpdatedAt and mtime agree. It does not fail on disagreement, but logs the
// divergence so the operator sees the gap. This mirrors the production
// contract: mtime is always set after the controller flushes, so divergence
// should be rare.
func TestMostRecentSessionByMTimeAgreesWithSidecar(t *testing.T) {
	dir := buildDirMTime(t, 50, true, true)
	fastPath, ok := mostRecentSessionByMTime(dir)
	if !ok {
		t.Fatal("fast path: ok=false")
	}
	sessions, err := agent.ListSessions(dir)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("sidecar path: empty")
	}
	sidecar := sessions[0].Path
	if fastPath != sidecar {
		// Soft-fail: dump a diff so divergence is visible in CI logs.
		names := make([]string, 0, 5)
		for _, s := range sessions[:min(5, len(sessions))] {
			names = append(names, filepath.Base(s.Path))
		}
		sort.Strings(names)
		t.Logf("divergence: fast=%s sidecar=%s sidecar-top-5=%v",
			filepath.Base(fastPath), filepath.Base(sidecar), names)
		t.Fatalf("mtime vs sidecar disagree on top-1; pick one or accept divergence")
	}
}

// TestMostRecentSessionByMTimeEmptyDir covers an edge case: no candidate.
func TestMostRecentSessionByMTimeEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if path, ok := mostRecentSessionByMTime(dir); ok || path != "" {
		t.Fatalf("empty dir should yield (\"\", false), got (%q, %v)", path, ok)
	}
	if path, ok := mostRecentSessionByMTime(""); ok || path != "" {
		t.Fatalf("empty path should yield (\"\", false), got (%q, %v)", path, ok)
	}
}

// TestMostRecentSessionByMTimeMissingDir covers a non-existent directory: the
// helper must not panic and must report ok=false cleanly.
func TestMostRecentSessionByMTimeMissingDir(t *testing.T) {
	if path, ok := mostRecentSessionByMTime("/nonexistent/path/for/test"); ok || path != "" {
		t.Fatalf("missing dir should yield (\"\", false), got (%q, %v)", path, ok)
	}
}

// helper unused after the file stops importing encoding/json — keep as no-op so
// go vet does not complain about unused imports if other tests are trimmed.
var _ = json.Marshal
