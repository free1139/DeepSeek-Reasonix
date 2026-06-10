package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"reasonix/internal/evidence"
	"reasonix/internal/provider"
)

func TestTodoWriteAcceptsLevels(t *testing.T) {
	args := json.RawMessage(`{"todos":[` +
		`{"content":"Phase","status":"in_progress","level":0},` +
		`{"content":"sub","status":"pending","level":1}]}`)
	if _, err := (todoWrite{}).Execute(context.Background(), args); err != nil {
		t.Fatalf("levels 0/1 should be accepted: %v", err)
	}
}

func TestTodoWriteRejectsBadLevel(t *testing.T) {
	args := json.RawMessage(`{"todos":[{"content":"x","status":"pending","level":2}]}`)
	_, err := (todoWrite{}).Execute(context.Background(), args)
	if err == nil || !strings.Contains(err.Error(), "level") {
		t.Fatalf("level 2 should be rejected with a level error, got %v", err)
	}
}

func TestTodoWriteRejectsNewCompletedWithoutCompleteStepReceipt(t *testing.T) {
	ledger := evidence.NewLedger()
	ledger.Record(evidence.Receipt{
		ToolName: "todo_write",
		Success:  true,
		Todos:    []evidence.TodoItem{{Content: "Add parser", Status: "in_progress"}},
	})
	ctx := evidence.WithLedger(context.Background(), ledger)
	args := json.RawMessage(`{"todos":[{"content":"Add parser","status":"completed"}]}`)

	_, err := (todoWrite{}).Execute(ctx, args)
	if err == nil || !strings.Contains(err.Error(), "complete_step") {
		t.Fatalf("new completion without complete_step should be rejected, got %v", err)
	}
}

func TestTodoWriteAcceptsNewCompletedWithCompleteStepReceipt(t *testing.T) {
	ledger := evidence.NewLedger()
	ledger.Record(evidence.Receipt{
		ToolName: "todo_write",
		Success:  true,
		Todos:    []evidence.TodoItem{{Content: "Add parser", Status: "in_progress"}},
	})
	ledger.Record(evidence.Receipt{ToolName: "complete_step", Success: true, Step: "Add parser"})
	ctx := evidence.WithLedger(context.Background(), ledger)
	args := json.RawMessage(`{"todos":[{"content":"Add parser","status":"completed"}]}`)

	if _, err := (todoWrite{}).Execute(ctx, args); err != nil {
		t.Fatalf("matching complete_step should authorize new completion: %v", err)
	}
}

func TestTodoWriteAllowsInitialCompletedWithoutBaseline(t *testing.T) {
	ctx := evidence.WithLedger(context.Background(), evidence.NewLedger())
	args := json.RawMessage(`{"todos":[{"content":"Add parser","status":"completed"}]}`)

	if _, err := (todoWrite{}).Execute(ctx, args); err != nil {
		t.Fatalf("initial completed todo without baseline should preserve existing behavior: %v", err)
	}
}

func TestTodoWriteAcceptsCrossTurnCompleteStepViaSession(t *testing.T) {
	// Simulate a cross-turn scenario: the ledger has only the prior turn's
	// todo_write baseline, with no complete_step receipt (the ledger was reset).
	// But the session messages contain a complete_step call from the prior turn
	// WITH its tool result, proving it was executed. The cross-turn fallback
	// should find it and authorize the completion.
	ledger := evidence.NewLedger()
	ledger.Record(evidence.Receipt{
		ToolName: "todo_write",
		Success:  true,
		Todos:    []evidence.TodoItem{{Content: "Fix parser", Status: "in_progress"}},
	})

	sessionMsgs := []provider.Message{
		{
			Role: provider.RoleAssistant,
			ToolCalls: []provider.ToolCall{
				{
					ID:        "step-1",
					Name:      "complete_step",
					Arguments: `{"step":"Fix parser","result":"done","evidence":[{"kind":"manual","summary":"verified"}]}`,
				},
			},
		},
		{
			Role:       provider.RoleTool,
			ToolCallID: "step-1",
			Name:       "complete_step",
			Content:    `Step "Fix parser" signed off with 1 evidence item(s) [manual].`,
		},
	}

	ctx := evidence.WithLedger(context.Background(), ledger)
	ctx = evidence.WithSessionMessages(ctx, sessionMsgs)
	args := json.RawMessage(`{"todos":[{"content":"Fix parser","status":"completed"}]}`)

	if _, err := (todoWrite{}).Execute(ctx, args); err != nil {
		t.Fatalf("cross-turn complete_step from session should authorize new completion: %v", err)
	}
}

func TestTodoWriteAcceptsCrossTurnCompleteStepViaSessionByIndex(t *testing.T) {
	// Same as above, but using a numeric step index ("1") instead of a content
	// string and including a tool result. The cross-turn fallback must handle both.
	ledger := evidence.NewLedger()
	ledger.Record(evidence.Receipt{
		ToolName: "todo_write",
		Success:  true,
		Todos:    []evidence.TodoItem{{Content: "Add tests", Status: "in_progress"}, {Content: "Write docs", Status: "pending"}},
	})

	sessionMsgs := []provider.Message{
		{
			Role: provider.RoleAssistant,
			ToolCalls: []provider.ToolCall{
				{
					ID:        "step-1",
					Name:      "complete_step",
					Arguments: `{"step":"1","result":"done","evidence":[{"kind":"manual","summary":"verified"}]}`,
				},
			},
		},
		{
			Role:       provider.RoleTool,
			ToolCallID: "step-1",
			Name:       "complete_step",
			Content:    `Step "1" signed off with 1 evidence item(s) [manual].`,
		},
	}

	ctx := evidence.WithLedger(context.Background(), ledger)
	ctx = evidence.WithSessionMessages(ctx, sessionMsgs)
	args := json.RawMessage(`{"todos":[{"content":"Add tests","status":"completed"},{"content":"Write docs","status":"in_progress"}]}`)

	if _, err := (todoWrite{}).Execute(ctx, args); err != nil {
		t.Fatalf("cross-turn numeric complete_step from session should authorize completion: %v", err)
	}
}

func TestTodoWriteRejectsCrossTurnMissingCompleteStep(t *testing.T) {
	// Cross-turn without any complete_step in session should still be rejected.
	ledger := evidence.NewLedger()
	ledger.Record(evidence.Receipt{
		ToolName: "todo_write",
		Success:  true,
		Todos:    []evidence.TodoItem{{Content: "Fix parser", Status: "in_progress"}},
	})

	ctx := evidence.WithLedger(context.Background(), ledger)
	ctx = evidence.WithSessionMessages(ctx, []provider.Message{})
	args := json.RawMessage(`{"todos":[{"content":"Fix parser","status":"completed"}]}`)

	_, err := (todoWrite{}).Execute(ctx, args)
	if err == nil || !strings.Contains(err.Error(), "complete_step") {
		t.Fatalf("cross-turn without complete_step should still be rejected, got %v", err)
	}
}

func TestTodoWriteIgnoresFailedCompleteStepReceipt(t *testing.T) {
	ledger := evidence.NewLedger()
	ledger.Record(evidence.Receipt{
		ToolName: "todo_write",
		Success:  true,
		Todos:    []evidence.TodoItem{{Content: "Add parser", Status: "in_progress"}},
	})
	ledger.Record(evidence.Receipt{ToolName: "complete_step", Success: false, Step: "Add parser"})
	ctx := evidence.WithLedger(context.Background(), ledger)
	args := json.RawMessage(`{"todos":[{"content":"Add parser","status":"completed"}]}`)

	_, err := (todoWrite{}).Execute(ctx, args)
	if err == nil || !strings.Contains(err.Error(), "complete_step") {
		t.Fatalf("failed complete_step should not authorize new completion, got %v", err)
	}
}
