package engine

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func testEngine() *Engine {
	cfg := DefaultConfig()
	cfg.DisableThoughtLogging = true
	return NewEngine(cfg)
}

// TestSequentialThinkingEngine verifies the thinking engine correctly links thoughts across steps and branches.
func TestSequentialThinkingEngine(t *testing.T) {
	eng := testEngine()
	ctx := context.Background()

	// Step 1
	out, err := eng.ProcessThought(ctx, "", ThoughtData{
		Thought:           "Starting my analysis on testing.",
		NextThoughtNeeded: true,
		ThoughtNumber:     1,
		TotalThoughts:     3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.ThoughtHistoryLength != 1 {
		t.Errorf("expected history length 1, got %d", out.ThoughtHistoryLength)
	}
	if out.RetainedHistoryLength != 1 {
		t.Errorf("expected retained history length 1, got %d", out.RetainedHistoryLength)
	}
	if out.TotalThoughts != 3 {
		t.Errorf("TotalThoughts: got %d, want 3", out.TotalThoughts)
	}

	// Step 2 with Branch
	branchID := "test-branch"
	fromThought := 1
	out2, err := eng.ProcessThought(ctx, "", ThoughtData{
		Thought:           "This is a separate possibility based on step 1.",
		NextThoughtNeeded: false,
		ThoughtNumber:     2,
		TotalThoughts:     3,
		BranchFromThought: &fromThought,
		BranchID:          &branchID,
	})
	if err != nil {
		t.Fatalf("unexpected error on branch: %v", err)
	}

	if len(out2.Branches) != 1 || out2.Branches[0] != branchID {
		t.Errorf("expected branch '%s', got %v", branchID, out2.Branches)
	}
	if out2.ThoughtHistoryLength != 2 {
		t.Errorf("expected history length 2, got %d", out2.ThoughtHistoryLength)
	}

	if got := eng.Stats().ThoughtsProcessed; got != 2 {
		t.Errorf("ThoughtsProcessed: got %d, want 2", got)
	}
}

// TestConfigFromEnv verifies env overrides and rejection of invalid values.
func TestConfigFromEnv(t *testing.T) {
	t.Setenv("SEQ_THINKING_MAX_HISTORY", "5")
	t.Setenv("SEQ_THINKING_LRU_SIZE", "-3") // invalid: ignored, default retained
	t.Setenv("SEQ_THINKING_MAX_BRANCHES", "notanumber")
	t.Setenv("DISABLE_THOUGHT_LOGGING", "TRUE")

	c := ConfigFromEnv()
	if c.MaxHistory != 5 {
		t.Errorf("MaxHistory: got %d, want 5", c.MaxHistory)
	}
	if c.LRUSize != defaultLRUSize {
		t.Errorf("LRUSize should keep default on invalid input, got %d", c.LRUSize)
	}
	if c.MaxBranches != defaultMaxBranches {
		t.Errorf("MaxBranches should keep default on parse error, got %d", c.MaxBranches)
	}
	if !c.DisableThoughtLogging {
		t.Error("DisableThoughtLogging should be true (case-insensitive)")
	}
}

// TestValidation rejects out-of-bounds input.
func TestValidation(t *testing.T) {
	eng := testEngine()
	ctx := context.Background()
	big := strings.Repeat("x", defaultMaxThoughtBytes+1)
	longBranch := strings.Repeat("b", maxBranchIDBytes+1)
	from := 1

	cases := []struct {
		name  string
		input ThoughtData
	}{
		{"zero thoughtNumber", ThoughtData{Thought: "a", ThoughtNumber: 0, TotalThoughts: 1}},
		{"zero totalThoughts", ThoughtData{Thought: "a", ThoughtNumber: 1, TotalThoughts: 0}},
		{"over max total", ThoughtData{Thought: "a", ThoughtNumber: 1, TotalThoughts: defaultMaxTotalThoughts + 1}},
		{"oversized thought", ThoughtData{Thought: big, ThoughtNumber: 1, TotalThoughts: 1}},
		{"oversized branchId", ThoughtData{Thought: "a", ThoughtNumber: 1, TotalThoughts: 1, BranchFromThought: &from, BranchID: &longBranch}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := eng.ProcessThought(ctx, "", tc.input); err == nil {
				t.Errorf("expected validation error for %s", tc.name)
			}
		})
	}
	if got := eng.Stats().ThoughtsProcessed; got != 0 {
		t.Errorf("rejected inputs must not increment ThoughtsProcessed, got %d", got)
	}
}

// TestBoundedHistory verifies the ring caps retained history while totalSeen keeps climbing.
func TestBoundedHistory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DisableThoughtLogging = true
	cfg.MaxHistory = 10
	eng := NewEngine(cfg)
	ctx := context.Background()

	var last *OutputData
	for i := 1; i <= 25; i++ {
		out, err := eng.ProcessThought(ctx, "sess", ThoughtData{
			Thought:           "t",
			NextThoughtNeeded: true,
			ThoughtNumber:     i,
			TotalThoughts:     30,
		})
		if err != nil {
			t.Fatalf("unexpected error at %d: %v", i, err)
		}
		last = out
	}
	if last.ThoughtHistoryLength != 25 {
		t.Errorf("ThoughtHistoryLength should be monotonic (25), got %d", last.ThoughtHistoryLength)
	}
	if last.RetainedHistoryLength != 10 {
		t.Errorf("RetainedHistoryLength should be capped at 10, got %d", last.RetainedHistoryLength)
	}
	state, _ := eng.sessions.Get("sess")
	state.mu.Lock()
	retained := len(state.history)
	state.mu.Unlock()
	if retained != 10 {
		t.Errorf("retained history should be capped at 10, got %d", retained)
	}
}

// TestBranchCap verifies distinct branches are capped per session.
func TestBranchCap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DisableThoughtLogging = true
	cfg.MaxBranches = 3
	eng := NewEngine(cfg)
	ctx := context.Background()
	from := 1

	var out *OutputData
	for i := range 3 {
		id := string(rune('A' + i))
		o, err := eng.ProcessThought(ctx, "sess", ThoughtData{
			Thought:           "t",
			NextThoughtNeeded: true,
			ThoughtNumber:     i + 1,
			TotalThoughts:     20,
			BranchFromThought: &from,
			BranchID:          &id,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		out = o
	}
	if len(out.Branches) != 3 {
		t.Errorf("branches should be capped at 3, got %d", len(out.Branches))
	}

	overflow := "D"
	_, err := eng.ProcessThought(ctx, "sess", ThoughtData{
		Thought:           "t",
		NextThoughtNeeded: true,
		ThoughtNumber:     4,
		TotalThoughts:     20,
		BranchFromThought: &from,
		BranchID:          &overflow,
	})
	if err == nil {
		t.Fatal("expected error when branch cap exceeded")
	}
}

// TestSessionIsolation verifies distinct session keys keep independent state.
func TestSessionIsolation(t *testing.T) {
	eng := testEngine()
	ctx := context.Background()

	// Session A records a contradiction without resolution -> would block A's next thought.
	_, err := eng.ProcessThought(ctx, "connA", ThoughtData{
		Thought:               "A1",
		NextThoughtNeeded:     true,
		ContradictionDetected: true,
		ThoughtNumber:         1,
		TotalThoughts:         2,
	})
	if err != nil {
		t.Fatalf("A1 unexpected error: %v", err)
	}

	// Session B (different connection key) must NOT be blocked by A's contradiction.
	out, err := eng.ProcessThought(ctx, "connB", ThoughtData{
		Thought:           "B1",
		NextThoughtNeeded: true,
		ThoughtNumber:     1,
		TotalThoughts:     1,
	})
	if err != nil {
		t.Fatalf("B1 should not be blocked by A's state: %v", err)
	}
	if out.ThoughtHistoryLength != 1 {
		t.Errorf("session B should have its own history of 1, got %d", out.ThoughtHistoryLength)
	}
}

// TestResumedEmptyMetric verifies the resumed-empty counter increments.
func TestResumedEmptyMetric(t *testing.T) {
	eng := testEngine()
	ctx := context.Background()
	// First thought arrives with number > 1 and no prior history.
	_, err := eng.ProcessThought(ctx, "sess", ThoughtData{
		Thought:           "resumed",
		NextThoughtNeeded: true,
		ThoughtNumber:     5,
		TotalThoughts:     5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := eng.Stats().ResumedEmpty; got != 1 {
		t.Errorf("ResumedEmpty: got %d, want 1", got)
	}
}

// TestContextCancellation verifies a cancelled context is rejected before work.
func TestContextCancellation(t *testing.T) {
	eng := testEngine()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := eng.ProcessThought(ctx, "", ThoughtData{Thought: "x", ThoughtNumber: 1, TotalThoughts: 1}); err == nil {
		t.Error("expected error on cancelled context")
	}
}

// TestRecordHandlerError verifies the handler-error counter.
func TestRecordHandlerError(t *testing.T) {
	eng := testEngine()
	eng.RecordHandlerError()
	eng.RecordHandlerError()
	if got := eng.Stats().HandlerErrors; got != 2 {
		t.Errorf("HandlerErrors: got %d, want 2", got)
	}
}

// TestFormatThoughtTruncation verifies long bodies are truncated and length-annotated.
func TestFormatThoughtTruncation(t *testing.T) {
	eng := testEngine()
	const extra = 50
	body := strings.Repeat("z", maxLogThoughtBytes+extra)
	got := eng.formatThought(ThoughtData{Thought: body, ThoughtNumber: 1, TotalThoughts: 1})
	if strings.Contains(got, body) {
		t.Error("formatThought should not contain the full body")
	}
	if !strings.Contains(got, fmt.Sprintf("+%d bytes", extra)) {
		t.Errorf("expected truncation annotation, got %q", got)
	}
	if !strings.Contains(got, fmt.Sprintf("(%dB)", maxLogThoughtBytes+extra)) {
		t.Errorf("expected byte-length annotation, got %q", got)
	}
}

//go:fix inline

func TestDialecticRequiresResolution(t *testing.T) {
	eng := testEngine()
	ctx := context.Background()

	_, err := eng.ProcessThought(ctx, "sess", ThoughtData{
		Thought:               "contradiction found",
		NextThoughtNeeded:     true,
		ContradictionDetected: true,
		ThoughtNumber:         1,
		TotalThoughts:         3,
	})
	if err != nil {
		t.Fatalf("first thought unexpected error: %v", err)
	}

	_, err = eng.ProcessThought(ctx, "sess", ThoughtData{
		Thought:           "continuing without resolution",
		NextThoughtNeeded: true,
		ThoughtNumber:     2,
		TotalThoughts:     3,
	})
	if err == nil {
		t.Fatal("expected dialectic error when resolutionStrategy is missing")
	}
}

func TestDialecticCannotConcludeWithContradiction(t *testing.T) {
	eng := testEngine()
	ctx := context.Background()

	_, err := eng.ProcessThought(ctx, "sess", ThoughtData{
		Thought:               "open contradiction",
		NextThoughtNeeded:     false,
		ContradictionDetected: true,
		ThoughtNumber:         1,
		TotalThoughts:         1,
	})
	if err == nil {
		t.Fatal("expected dialectic error when concluding with open contradiction")
	}
}

func TestRevisionSpliceReplacesRetainedThought(t *testing.T) {
	eng := testEngine()
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		if _, err := eng.ProcessThought(ctx, "sess", ThoughtData{
			Thought:           fmt.Sprintf("thought-%d", i),
			NextThoughtNeeded: true,
			ThoughtNumber:     i,
			TotalThoughts:     3,
		}); err != nil {
			t.Fatalf("setup thought %d: %v", i, err)
		}
	}

	revises := 2
	out, err := eng.ProcessThought(ctx, "sess", ThoughtData{
		Thought:           "revised-thought-2",
		NextThoughtNeeded: true,
		ThoughtNumber:     4,
		TotalThoughts:     4,
		IsRevision:        new(true),
		RevisesThought:    &revises,
	})
	if err != nil {
		t.Fatalf("revision unexpected error: %v", err)
	}
	if out.RetainedHistoryLength != 3 {
		t.Errorf("retained history should stay 3 after revision, got %d", out.RetainedHistoryLength)
	}
	if out.ThoughtHistoryLength != 4 {
		t.Errorf("lifetime counter should be 4 after revision, got %d", out.ThoughtHistoryLength)
	}

	state, _ := eng.sessions.Get("sess")
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.history[1].Thought != "revised-thought-2" {
		t.Errorf("slot 2 should be revised, got %q", state.history[1].Thought)
	}
}

func TestRevisionFailsWhenEvicted(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DisableThoughtLogging = true
	cfg.MaxHistory = 2
	eng := NewEngine(cfg)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		if _, err := eng.ProcessThought(ctx, "sess", ThoughtData{
			Thought:           "t",
			NextThoughtNeeded: true,
			ThoughtNumber:     i,
			TotalThoughts:     3,
		}); err != nil {
			t.Fatalf("setup thought %d: %v", i, err)
		}
	}

	revises := 1
	_, err := eng.ProcessThought(ctx, "sess", ThoughtData{
		Thought:           "too late",
		NextThoughtNeeded: true,
		ThoughtNumber:     4,
		TotalThoughts:     4,
		IsRevision:        new(true),
		RevisesThought:    &revises,
	})
	if err == nil {
		t.Fatal("expected error revising evicted thought")
	}
}

func TestRevisionRequiresRevisesThought(t *testing.T) {
	eng := testEngine()
	ctx := context.Background()

	_, err := eng.ProcessThought(ctx, "sess", ThoughtData{
		Thought:           "orphan revision",
		NextThoughtNeeded: true,
		ThoughtNumber:     1,
		TotalThoughts:     1,
		IsRevision:        new(true),
	})
	if err == nil {
		t.Fatal("expected error when isRevision is true without revisesThought")
	}
}

func TestBranchThoughtRefCap(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DisableThoughtLogging = true
	cfg.MaxBranchThoughtRefs = 2
	eng := NewEngine(cfg)
	ctx := context.Background()
	from := 1
	branchID := "alpha"

	for i := 1; i <= 2; i++ {
		if _, err := eng.ProcessThought(ctx, "sess", ThoughtData{
			Thought:           "t",
			NextThoughtNeeded: true,
			ThoughtNumber:     i,
			TotalThoughts:     5,
			BranchFromThought: &from,
			BranchID:          &branchID,
		}); err != nil {
			t.Fatalf("branch ref %d unexpected error: %v", i, err)
		}
	}

	_, err := eng.ProcessThought(ctx, "sess", ThoughtData{
		Thought:           "t",
		NextThoughtNeeded: true,
		ThoughtNumber:     3,
		TotalThoughts:     5,
		BranchFromThought: &from,
		BranchID:          &branchID,
	})
	if err == nil {
		t.Fatal("expected error when per-branch ref cap exceeded")
	}
}

func TestLRUEviction(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DisableThoughtLogging = true
	cfg.LRUSize = 2
	eng := NewEngine(cfg)
	ctx := context.Background()

	for _, sid := range []string{"s1", "s2", "s3"} {
		if _, err := eng.ProcessThought(ctx, sid, ThoughtData{
			Thought:           "t",
			NextThoughtNeeded: false,
			ThoughtNumber:     1,
			TotalThoughts:     1,
		}); err != nil {
			t.Fatalf("session %s: %v", sid, err)
		}
	}
	if got := eng.Stats().Evictions; got != 1 {
		t.Errorf("Evictions: got %d, want 1", got)
	}
}

func TestBranchesSorted(t *testing.T) {
	eng := testEngine()
	ctx := context.Background()
	from := 1

	for _, id := range []string{"charlie", "alpha", "bravo"} {
		branchID := id
		if _, err := eng.ProcessThought(ctx, "sess", ThoughtData{
			Thought:           "t",
			NextThoughtNeeded: true,
			ThoughtNumber:     1,
			TotalThoughts:     3,
			BranchFromThought: &from,
			BranchID:          &branchID,
		}); err != nil {
			t.Fatalf("branch %s: %v", id, err)
		}
	}

	state, _ := eng.sessions.Get("sess")
	state.mu.Lock()
	defer state.mu.Unlock()
	out := eng.buildOutput(state, ThoughtData{})
	if len(out.Branches) != 3 {
		t.Fatalf("expected 3 branches, got %d", len(out.Branches))
	}
	if out.Branches[0] != "alpha" || out.Branches[1] != "bravo" || out.Branches[2] != "charlie" {
		t.Errorf("branches not sorted: %v", out.Branches)
	}
}

func TestNewEngine_Fallback(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LRUSize = -1 // Invalid size will cause lru.NewWithEvict to fail
	eng := NewEngine(cfg)
	if eng == nil || eng.sessions == nil {
		t.Fatal("expected fallback to default size to succeed")
	}
}

func TestFormatThoughtPrefixes(t *testing.T) {
	eng := testEngine()
	base := ThoughtData{Thought: "text", ThoughtNumber: 1, TotalThoughts: 1}

	revThought := 1
	revInput := base
	revInput.IsRevision = new(bool)
	*revInput.IsRevision = true
	revInput.RevisesThought = &revThought

	if got := eng.formatThought(revInput); !strings.Contains(got, "🔄 Revision") {
		t.Errorf("missing revision prefix: %s", got)
	}

	branchID := "b1"
	branchInput := base
	branchInput.BranchFromThought = &revThought
	branchInput.BranchID = &branchID
	if got := eng.formatThought(branchInput); !strings.Contains(got, "🌿 Branch") {
		t.Errorf("missing branch prefix: %s", got)
	}

	contraInput := base
	contraInput.ContradictionDetected = true
	if got := eng.formatThought(contraInput); !strings.Contains(got, "⚠️ Contradiction") {
		t.Errorf("missing contradiction prefix: %s", got)
	}

	resStr := "fix it"
	resInput := base
	resInput.ResolutionStrategy = &resStr
	if got := eng.formatThought(resInput); !strings.Contains(got, "💡 Resolution") {
		t.Errorf("missing resolution prefix: %s", got)
	}
}
