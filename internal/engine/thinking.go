package engine

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru/v2"

	"github.com/maccavelli/mcplib"
)

// Tuning defaults. These bound per-session and per-process memory so a long or
// adversarial run cannot exhaust memory.
const (
	defaultLRUSize              = 1000
	defaultMaxHistory           = 1000
	defaultMaxBranches          = 64
	defaultMaxBranchThoughtRefs = 128
	defaultMaxThoughtBytes      = 16384
	defaultMaxTotalThoughts     = 10000
	maxBranchIDBytes            = 256
	maxLogThoughtBytes          = 200
)

// sessionCreateMu serializes first-time session insertion to prevent orphaned
// SessionState instances under concurrent get-or-create for the same key.
var sessionCreateMu sync.Mutex

// Config holds the engine's tunable limits. Sourced from DefaultConfig and
// optionally overridden via environment variables (see ConfigFromEnv); this
// replaces the previous viper-based configuration.
type Config struct {
	LRUSize               int  // distinct sessions retained in the LRU cache
	MaxHistory            int  // retained thoughts per session (ring, drop-oldest)
	MaxBranches           int  // distinct branch IDs retained per session
	MaxBranchThoughtRefs  int  // thought-number refs tracked per branch ID
	MaxThoughtBytes       int  // maximum accepted size of a single Thought
	MaxTotalThoughts      int  // upper bound for ThoughtNumber / TotalThoughts
	DisableThoughtLogging bool // suppress per-thought INFO logging
}

// DefaultConfig returns the built-in limits.
func DefaultConfig() Config {
	return Config{
		LRUSize:               defaultLRUSize,
		MaxHistory:            defaultMaxHistory,
		MaxBranches:           defaultMaxBranches,
		MaxBranchThoughtRefs:  defaultMaxBranchThoughtRefs,
		MaxThoughtBytes:       defaultMaxThoughtBytes,
		MaxTotalThoughts:      defaultMaxTotalThoughts,
		DisableThoughtLogging: false,
	}
}

// ConfigFromEnv layers environment overrides on top of DefaultConfig. Invalid
// or non-positive integer values are ignored (the default is retained).
func ConfigFromEnv() Config {
	c := DefaultConfig()
	c.DisableThoughtLogging = strings.EqualFold(os.Getenv("DISABLE_THOUGHT_LOGGING"), "true")
	intEnv("SEQ_THINKING_LRU_SIZE", &c.LRUSize)
	intEnv("SEQ_THINKING_MAX_HISTORY", &c.MaxHistory)
	intEnv("SEQ_THINKING_MAX_BRANCHES", &c.MaxBranches)
	intEnv("SEQ_THINKING_MAX_BRANCH_REFS", &c.MaxBranchThoughtRefs)
	intEnv("SEQ_THINKING_MAX_THOUGHT_BYTES", &c.MaxThoughtBytes)
	intEnv("SEQ_THINKING_MAX_TOTAL_THOUGHTS", &c.MaxTotalThoughts)
	return c
}

// intEnv parses a positive integer from the named env var into dst, leaving dst
// unchanged on absence, parse error, or a non-positive value.
func intEnv(name string, dst *int) {
	v, ok := os.LookupEnv(name)
	if !ok {
		return
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n <= 0 {
		slog.Warn("ignoring invalid engine config env var", "name", name, "value", v)
		return
	}
	*dst = n
}

// ThoughtData represents a single structured thinking step for the sequential thought process.
type ThoughtData struct {
	mcplib.UniversalBaseInput
	Thought               string  `json:"thought" jsonschema:"[REQUIRED] Your current thinking step"`
	SelfCritique          string  `json:"selfCritique,omitzero" jsonschema:"Recommended self-correction and evaluation of the current thought logic (optional)"`
	ContradictionDetected bool    `json:"contradictionDetected,omitzero" jsonschema:"Must be set to true if SelfCritique identifies a paradox, flaw, or assumption"`
	ResolutionStrategy    *string `json:"resolutionStrategy,omitzero" jsonschema:"Mandatory step outlining how to fix the logic ONLY if the previous thought had a contradiction"`
	NextThoughtNeeded     bool    `json:"nextThoughtNeeded" jsonschema:"[REQUIRED] Whether another thought step is needed"`
	ThoughtNumber         int     `json:"thoughtNumber" jsonschema:"[REQUIRED] Current thought number (numeric value, e.g., 1, 2, 3)" jsonschema_extras:"minimum=1"`
	TotalThoughts         int     `json:"totalThoughts" jsonschema:"[REQUIRED] Estimated total thoughts needed (numeric value, e.g., 5, 10)" jsonschema_extras:"minimum=1"`
	IsRevision            *bool   `json:"isRevision,omitzero" jsonschema:"Set true to replace a prior retained thought in-place (requires revisesThought)"`
	RevisesThought        *int    `json:"revisesThought,omitzero" jsonschema:"Required when isRevision is true: the thoughtNumber to replace in retained history" jsonschema_extras:"minimum=1"`
	BranchFromThought     *int    `json:"branchFromThought,omitzero" jsonschema:"Branching point thought number" jsonschema_extras:"minimum=1"`
	BranchID              *string `json:"branchId,omitzero" jsonschema:"Branch identifier"`
	NeedsMoreThoughts     *bool   `json:"needsMoreThoughts,omitzero" jsonschema:"Advisory telemetry only; not enforced by engine"`
	ActionableInsight     *bool   `json:"actionableInsight,omitzero" jsonschema:"Advisory telemetry only; not enforced by engine"`
}

// OutputData represents the final structured output after processing a thought.
type OutputData struct {
	ThoughtNumber         int      `json:"thoughtNumber"`
	TotalThoughts         int      `json:"totalThoughts"`
	NextThoughtNeeded     bool     `json:"nextThoughtNeeded"`
	Branches              []string `json:"branches"`
	ThoughtHistoryLength  int      `json:"thoughtHistoryLength"`  // lifetime thoughts processed in session (monotonic)
	RetainedHistoryLength int      `json:"retainedHistoryLength"` // current ring buffer size
}

// Stats is a point-in-time snapshot of engine counters for observability.
type Stats struct {
	ThoughtsProcessed uint64 `json:"thoughtsProcessed"`
	HandlerErrors     uint64 `json:"handlerErrors"`
	Evictions         uint64 `json:"evictions"`
	ResumedEmpty      uint64 `json:"resumedEmpty"`
	ActiveSessions    int    `json:"activeSessions"`
}

type metrics struct {
	thoughtsProcessed atomic.Uint64
	handlerErrors     atomic.Uint64
	evictions         atomic.Uint64
	resumedEmpty      atomic.Uint64
}

// SessionState represents the state for an isolated sequential thinking session.
type SessionState struct {
	mu        sync.Mutex
	history   []ThoughtData    // bounded ring (drop-oldest), newest at the tail
	totalSeen int              // monotonic count of thoughts ever processed for this session
	branches  map[string][]int // branchID -> ThoughtNumbers (references, not copies)
}

// Engine provides state management for sequential thinking steps and branches.
type Engine struct {
	cfg      Config
	sessions *lru.Cache[string, *SessionState]
	metrics  metrics
}

// NewEngine creates a new Engine instance with the supplied configuration.
func NewEngine(cfg Config) *Engine {
	e := &Engine{cfg: cfg}
	cache, err := lru.NewWithEvict[string, *SessionState](cfg.LRUSize, func(key string, _ *SessionState) {
		slog.Warn("session evicted from LRU cache; its history is discarded", "session", key)
		e.metrics.evictions.Add(1)
	})
	if err != nil {
		// LRUSize is validated positive by ConfigFromEnv; fall back defensively.
		var fallbackErr error
		cache, fallbackErr = lru.NewWithEvict[string, *SessionState](defaultLRUSize, func(key string, _ *SessionState) {
			slog.Warn("session evicted from LRU cache; its history is discarded", "session", key)
			e.metrics.evictions.Add(1)
		})
		if fallbackErr != nil {
			slog.Error("failed to create LRU cache even with default size", "error", fallbackErr)
		}
	}
	e.sessions = cache
	return e
}

// Stats returns a snapshot of the engine's observability counters.
func (e *Engine) Stats() Stats {
	return Stats{
		ThoughtsProcessed: e.metrics.thoughtsProcessed.Load(),
		HandlerErrors:     e.metrics.handlerErrors.Load(),
		Evictions:         e.metrics.evictions.Load(),
		ResumedEmpty:      e.metrics.resumedEmpty.Load(),
		ActiveSessions:    e.sessions.Len(),
	}
}

// RecordHandlerError increments the handler-error counter. Called by the tool
// handler when ProcessThought returns an error.
func (e *Engine) RecordHandlerError() {
	e.metrics.handlerErrors.Add(1)
}

func (e *Engine) formatThought(input ThoughtData) string {
	prefix := ""
	contextStr := ""

	if input.IsRevision != nil && *input.IsRevision && input.RevisesThought != nil {
		prefix = "🔄 Revision"
		contextStr = fmt.Sprintf(" (revising thought %d)", *input.RevisesThought)
	} else if input.BranchFromThought != nil && input.BranchID != nil {
		prefix = "🌿 Branch"
		contextStr = fmt.Sprintf(" (from thought %d, ID: %s)", *input.BranchFromThought, *input.BranchID)
	} else if input.ContradictionDetected {
		prefix = "⚠️ Contradiction"
	} else if input.ResolutionStrategy != nil && *input.ResolutionStrategy != "" {
		prefix = "💡 Resolution"
	} else {
		prefix = "💭 Thought"
	}

	// Avoid logging the full free-form thought body verbatim: it can contain
	// secrets the narrow log redactor won't catch, and it surfaces via the
	// get_internal_logs tool and logs resource. Truncate and annotate length.
	body := input.Thought
	if len(body) > maxLogThoughtBytes {
		body = body[:maxLogThoughtBytes] + fmt.Sprintf("…(+%d bytes)", len(input.Thought)-maxLogThoughtBytes)
	}

	return fmt.Sprintf("[%s %d/%d%s] (%dB) %s", prefix, input.ThoughtNumber, input.TotalThoughts, contextStr, len(input.Thought), body)
}

// validate enforces runtime input bounds. The jsonschema_extras "minimum"
// constraints are not honored by the SDK's schema reflector, so the only real
// enforcement happens here.
func (e *Engine) validate(input ThoughtData) error {
	if input.ThoughtNumber < 1 {
		return fmt.Errorf("invalid thoughtNumber %d: must be >= 1", input.ThoughtNumber)
	}
	if input.TotalThoughts < 1 {
		return fmt.Errorf("invalid totalThoughts %d: must be >= 1", input.TotalThoughts)
	}
	if input.ThoughtNumber > e.cfg.MaxTotalThoughts || input.TotalThoughts > e.cfg.MaxTotalThoughts {
		return fmt.Errorf("thought counts exceed limit %d", e.cfg.MaxTotalThoughts)
	}
	if len(input.Thought) > e.cfg.MaxThoughtBytes {
		return fmt.Errorf("thought exceeds %d bytes (got %d)", e.cfg.MaxThoughtBytes, len(input.Thought))
	}
	if input.BranchID != nil && len(*input.BranchID) > maxBranchIDBytes {
		return fmt.Errorf("branchId exceeds %d bytes", maxBranchIDBytes)
	}
	if input.IsRevision != nil && *input.IsRevision && input.RevisesThought == nil {
		return fmt.Errorf("isRevision=true requires revisesThought")
	}
	return nil
}

// applyRevision replaces a retained history slot when isRevision is set.
// Returns (true, nil) when a splice occurred, (false, nil) when not a revision.
func (e *Engine) applyRevision(state *SessionState, input ThoughtData) (bool, error) {
	if input.IsRevision == nil || !*input.IsRevision {
		return false, nil
	}
	target := *input.RevisesThought
	if target < 1 {
		return false, fmt.Errorf("invalid revisesThought %d: must be >= 1", target)
	}
	for i, t := range state.history {
		if t.ThoughtNumber == target {
			revised := input
			revised.ThoughtNumber = target
			state.history[i] = revised
			return true, nil
		}
	}
	return false, fmt.Errorf("revisesThought %d not found in retained history (may have been evicted)", target)
}

func (e *Engine) trackBranches(state *SessionState, input ThoughtData) error {
	if input.BranchFromThought == nil || input.BranchID == nil {
		return nil
	}
	branchID := *input.BranchID
	if _, exists := state.branches[branchID]; !exists {
		if len(state.branches) >= e.cfg.MaxBranches {
			return fmt.Errorf("branch limit %d reached for session", e.cfg.MaxBranches)
		}
		state.branches[branchID] = nil
	}
	refs := state.branches[branchID]
	if len(refs) >= e.cfg.MaxBranchThoughtRefs {
		return fmt.Errorf("branch %q thought reference limit %d reached", branchID, e.cfg.MaxBranchThoughtRefs)
	}
	state.branches[branchID] = append(refs, input.ThoughtNumber)
	return nil
}

func (e *Engine) buildOutput(state *SessionState, input ThoughtData) *OutputData {
	branchesKeys := make([]string, 0, len(state.branches))
	for k := range state.branches {
		branchesKeys = append(branchesKeys, k)
	}
	slices.Sort(branchesKeys)
	return &OutputData{
		ThoughtNumber:         input.ThoughtNumber,
		TotalThoughts:         input.TotalThoughts,
		NextThoughtNeeded:     input.NextThoughtNeeded,
		Branches:              branchesKeys,
		ThoughtHistoryLength:  state.totalSeen,
		RetainedHistoryLength: len(state.history),
	}
}

// ProcessThought records a single thought step, evaluates branching, and determines the next step.
// sessionKey is the per-connection fallback identity used when the input omits an explicit SessionID.
func (e *Engine) ProcessThought(ctx context.Context, sessionKey string, input ThoughtData) (*OutputData, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := e.validate(input); err != nil {
		return nil, err
	}

	sessionID := mcplib.StringValue(input.SessionID)
	if sessionID == "" {
		sessionID = sessionKey
	}
	if sessionID == "" {
		sessionID = "default_global"
	}

	if _, ok := e.sessions.Get(sessionID); !ok {
		sessionCreateMu.Lock()
		if _, ok := e.sessions.Get(sessionID); !ok {
			e.sessions.Add(sessionID, &SessionState{
				history:  make([]ThoughtData, 0, 16),
				branches: make(map[string][]int),
			})
		}
		sessionCreateMu.Unlock()
	}
	state, _ := e.sessions.Get(sessionID)

	out, err := e.recordLocked(ctx, state, sessionID, input)
	if err != nil {
		return nil, err
	}

	// Format + log outside the critical section: formatting and the slog
	// fan-out (JSON marshal, redaction regex, async write) are far more
	// expensive than the state mutation and must not serialize the session.
	if !e.cfg.DisableThoughtLogging && slog.Default().Enabled(ctx, slog.LevelInfo) {
		slog.Info(e.formatThought(input), "session", sessionID, "number", input.ThoughtNumber, "total", input.TotalThoughts)
	}

	e.metrics.thoughtsProcessed.Add(1)
	return out, nil
}

// recordLocked performs the per-session state mutation under state.mu and
// returns the response. Logging is deliberately left to the caller (outside
// the lock). input is passed by value, so the local TotalThoughts clamp does
// not leak back to the caller's copy.
func (e *Engine) recordLocked(ctx context.Context, state *SessionState, sessionID string, input ThoughtData) (*OutputData, error) {
	state.mu.Lock()
	defer state.mu.Unlock()

	if err := ctx.Err(); err != nil { // re-check: lock acquisition may have blocked
		return nil, err
	}

	if state.totalSeen == 0 && input.ThoughtNumber > 1 && (input.IsRevision == nil || !*input.IsRevision) {
		slog.Warn("resumed session has no retained history (new or evicted); continuing fresh",
			"session", sessionID, "thoughtNumber", input.ThoughtNumber)
		e.metrics.resumedEmpty.Add(1)
	}

	// Continuous Dialectic Gatekeeper: Forcing Socratic Evolution
	var previousContradiction bool
	if len(state.history) > 0 {
		lastThought := state.history[len(state.history)-1]
		if lastThought.ContradictionDetected {
			previousContradiction = true
		}
	}

	// Rule 1: A prior contradiction MUST be addressed sequentially organically.
	if previousContradiction {
		if input.ResolutionStrategy == nil || *input.ResolutionStrategy == "" {
			return nil, fmt.Errorf("CONTINUOUS DIALECTIC VIOLATION: The previous thought encountered a contradiction. You MUST supply a 'resolutionStrategy' in this thought to proceed")
		}
	}

	// Rule 2: A sequence CANNOT conclude while a contradiction is actively open.
	if input.ContradictionDetected && !input.NextThoughtNeeded {
		return nil, fmt.Errorf("CONTINUOUS DIALECTIC VIOLATION: You cannot set 'nextThoughtNeeded=false' while 'contradictionDetected' is true. You must dynamically generate another thought executing the resolution strategy")
	}

	if input.ThoughtNumber > input.TotalThoughts {
		input.TotalThoughts = input.ThoughtNumber
	}

	applied, err := e.applyRevision(state, input)
	if err != nil {
		return nil, err
	}
	if applied {
		state.totalSeen++
		return e.buildOutput(state, input), nil
	}

	// Append to the bounded ring history (drop oldest when at capacity).
	if e.cfg.MaxHistory > 0 && len(state.history) >= e.cfg.MaxHistory {
		copy(state.history, state.history[1:])
		state.history = state.history[:len(state.history)-1]
	}
	state.history = append(state.history, input)
	state.totalSeen++

	if err := e.trackBranches(state, input); err != nil {
		return nil, err
	}

	return e.buildOutput(state, input), nil
}
