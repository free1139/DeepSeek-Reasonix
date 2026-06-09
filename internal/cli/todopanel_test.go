package cli

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"reasonix/internal/control"
	"reasonix/internal/event"
)

// TestRenderTodoPanelNesting proves a level-1 sub-step renders indented under
// its level-0 phase in the pinned task panel.
func TestRenderTodoPanelNesting(t *testing.T) {
	m := newTestChatTUI()
	m.width = 60
	m.todoArgs = `{"todos":[` +
		`{"content":"Phase A","status":"in_progress","level":0},` +
		`{"content":"sub one","status":"pending","level":1}]}`

	out := ansi.Strip(m.renderTodoPanel())
	if !strings.Contains(out, "Phase A") {
		t.Fatalf("panel missing phase:\n%s", out)
	}
	if !strings.Contains(out, "      ○ sub one") {
		t.Fatalf("sub-step not indented under its phase:\n%s", out)
	}
}

func TestRenderTodoPanelScrollsToInProgressTodo(t *testing.T) {
	m := newTestChatTUI()
	m.width = 72
	m.todoArgs = `{"todos":[` +
		`{"content":"Item 01","status":"completed"},` +
		`{"content":"Item 02","status":"completed"},` +
		`{"content":"Item 03","status":"completed"},` +
		`{"content":"Item 04","status":"completed"},` +
		`{"content":"Item 05","status":"completed"},` +
		`{"content":"Item 06","status":"completed"},` +
		`{"content":"Item 07","status":"completed"},` +
		`{"content":"Item 08","status":"completed"},` +
		`{"content":"Item 09","status":"in_progress","activeForm":"Working item 09"},` +
		`{"content":"Item 10","status":"pending"}]}`

	out := ansi.Strip(m.renderTodoPanel())
	if !strings.Contains(out, "Working item 09") {
		t.Fatalf("panel should keep the in-progress todo visible:\n%s", out)
	}
	if strings.Contains(out, "Item 01") {
		t.Fatalf("panel should window around the active todo instead of pinning the first rows:\n%s", out)
	}
}

// TestTodoPanelIncreasesBottomRows proves the todo panel consumes rows in the
// bottom region and shrinks the transcript viewport accordingly.
func TestTodoPanelIncreasesBottomRows(t *testing.T) {
	ctrl := control.New(control.Options{})
	m := newChatTUI(ctrl, "", make(chan event.Event, 1), 80)
	m0, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m0.(chatTUI)

	initialBottom := m.bottomRows()
	initialViewH := m.viewport.Height()
	_ = initialBottom

	// Add a single pending todo
	m.todoArgs = `{"todos":[{"content":"Step 1","status":"in_progress"}]}`
	m0, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m0.(chatTUI)

	if got := m.bottomRows(); got <= initialBottom {
		t.Fatalf("bottomRows with todo panel = %d, want > %d", got, initialBottom)
	}
	if got := m.viewport.Height(); got >= initialViewH {
		t.Fatalf("viewport height with todo panel = %d, want < %d", got, initialViewH)
	}
	if !m.viewport.AtBottom() {
		t.Fatal("viewport should be at bottom after WindowSizeMsg (content unchanged, just resized)")
	}
}

// TestTodoPanelClearsWhenAllDone proves the panel disappears and bottomRows drops
// back once every item is completed.
func TestTodoPanelClearsWhenAllDone(t *testing.T) {
	m := newTestChatTUI()
	m.width = 60

	// All items completed — panel should be empty
	m.todoArgs = `{"todos":[{"content":"Step 1","status":"completed"},{"content":"Step 2","status":"completed"}]}`
	if got := m.renderTodoPanel(); got != "" {
		t.Fatalf("renderTodoPanel with all-completed = %q, want empty", got)
	}
}

// TestTodoPanelAllPendingDoesNotClear proves the panel stays visible when every
// item is in "pending" state (no completed count equals total, no in_progress).
func TestTodoPanelAllPendingDoesNotClear(t *testing.T) {
	m := newTestChatTUI()
	m.width = 60

	m.todoArgs = `{"todos":[{"content":"Step 1","status":"pending"},{"content":"Step 2","status":"pending"}]}`
	if got := m.renderTodoPanel(); got == "" {
		t.Fatal("renderTodoPanel should render with all-pending items, got empty")
	}
	// Must also show 0/2 header
	out := ansi.Strip(m.renderTodoPanel())
	if !strings.Contains(out, "0/2") {
		t.Fatalf("panel should show 0/2 for all-pending items:\n%s", out)
	}
}

// TestTodoPanelAtBottomScroll proves that when the todo panel appears and the
// viewport was tracking the bottom, it stays at the bottom after the viewport
// shrinks. This is the real code path: todo_write tool result sets
// transcriptDirty so SetContent re-feeds the viewport and GotoBottom fires.
func TestTodoPanelAtBottomScroll(t *testing.T) {
	ctrl := control.New(control.Options{})
	m := newChatTUI(ctrl, "", make(chan event.Event, 1), 80)
	m0, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m0.(chatTUI)

	// Fill the transcript so the viewport overflows.
	for i := range 50 {
		m.transcript = append(m.transcript, fmt.Sprintf("line %d", i))
	}
	wrapped := wrapTranscript(strings.Join(m.transcript, "\n"), 79)
	m.wrappedLines = strings.Split(wrapped, "\n")
	m.viewport.SetContent(wrapped)
	m.viewport.GotoBottom()
	if !m.viewport.AtBottom() {
		t.Fatal("viewport should be at bottom after GotoBottom")
	}

	// Add a todo panel *and* mark transcript dirty (simulating the normal
	// collapseToolOutput → transcriptDirty flow when a tool result arrives).
	m.todoArgs = `{"todos":[{"content":"Fix parser","status":"in_progress"}]}`
	m.transcriptDirty = true

	// A second WindowSizeMsg of the same size is a safe no-op through update()
	// that still exercises the Update wrapper's resize + re-feed logic.
	m0, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m0.(chatTUI)

	if !m.viewport.AtBottom() {
		msg := fmt.Sprintf("viewport should stay at bottom after todo panel appears (h=%d, yoff=%d, total=%d)",
			m.viewport.Height(), m.viewport.YOffset(), m.viewport.TotalLineCount())
		t.Fatal(msg)
	}
	// The YOffset must have increased to compensate for the shorter viewport.
	total := m.viewport.TotalLineCount()
	vh := m.viewport.Height()
	if want := total - vh; m.viewport.YOffset() != want {
		t.Fatalf("viewport YOffset = %d, want %d (total %d - height %d)", m.viewport.YOffset(), want, total, vh)
	}
}

// TestTodoPanelScrolledUpOffsetPreserved proves that when the user has scrolled
// above the bottom, the YOffset stays unchanged when the todo panel appears —
// the viewport simply shrinks from the bottom, leaving the visible region
// unchanged.
func TestTodoPanelScrolledUpOffsetPreserved(t *testing.T) {
	ctrl := control.New(control.Options{})
	m := newChatTUI(ctrl, "", make(chan event.Event, 1), 80)
	m0, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m0.(chatTUI)

	for i := range 50 {
		m.transcript = append(m.transcript, fmt.Sprintf("line %d", i))
	}
	wrapped := wrapTranscript(strings.Join(m.transcript, "\n"), 79)
	m.wrappedLines = strings.Split(wrapped, "\n")
	m.viewport.SetContent(wrapped)
	m.viewport.GotoBottom()

	// Scroll up a few lines so the viewport is NO LONGER at bottom.
	m.viewport.ScrollUp(10)
	wasAtBottom := m.viewport.AtBottom()
	if wasAtBottom {
		t.Fatal("viewport should NOT be at bottom after ScrollUp")
	}
	prevYOff := m.viewport.YOffset()

	// Add a todo panel but do NOT set transcriptDirty — the content is
	// unchanged, so only the viewport height shrinks.
	m.todoArgs = `{"todos":[{"content":"Fix parser","status":"in_progress"}]}`
	m.transcriptDirty = false

	m0, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m0.(chatTUI)

	// The YOffset should be unchanged; the viewport just lost rows at the bottom.
	if got := m.viewport.YOffset(); got != prevYOff {
		t.Fatalf("viewport YOffset after todo panel = %d, want %d (unchanged)", got, prevYOff)
	}
}
