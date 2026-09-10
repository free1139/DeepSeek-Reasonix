package agent

// agentConfig is everything New fixes for an Agent's lifetime; nothing writes
// it afterwards, which agent_config_test.go enforces. Embedded rather than
// nested so access stays flat, as perTurnState already does. Separating it lets
// the struct-state ratchet count reachable states instead of size: an immutable
// field combines with nothing. Anything that changes after New goes on Agent.
type agentConfig struct {
	maxSteps           int
	maxStepsKey        string
	reasoningByteLimit int
	maxOutputTokens    int
	temperature        float64
	usageSource        string
	modelRef           string
	// workspaceID is a prompt-cache lineage component, so it must not move
	// while an agent lives — a change would silently rekey the cache.
	workspaceID string
	// classifierTaskText is the host-trusted task text for delivery intent
	// classification, set by sub-agent spawners whose Run input carries host
	// framing. Empty means classify the raw input verbatim.
	classifierTaskText string
	// writeWorkspaceRoot scopes write reservations when writeScheduler is set.
	writeWorkspaceRoot string
	// subagentDepth caps delegation; at maxSubagentDepth the recursive
	// agent/skill tools are excluded.
	subagentDepth    int
	maxSubagentDepth int
	// contextWindow and compactRatio decide when at most one provider-visible
	// checkpoint is installed; recentKeep and archiveDir shape what it keeps.
	contextWindow          int
	compactRatio           float64
	recentKeep             int
	archiveDir             string
	legacyAnchorSafetyGate bool
	// readCoordinatorShadow fixes the internal read-coordinator rollout switch
	// for the whole run; see Options.ReadPipeline.
	readCoordinatorShadow bool
	// legacyImplicitFullReads restores the pre-intent read default for rollback.
	legacyImplicitFullReads bool
}

// ReadPipelineOptions carries the internal read-pipeline rollback switches. The
// new behavior is the default; each switch exists so an operator can fall back
// for diagnosis, is host-local, and is fixed for the whole run.
//
// LegacyEvidenceGates inverts the default: the writer-declared evidence gate
// is OFF by default and only re-enables when an explicit Options value forces
// it on. The gate's hook (operation_evidence.go) is keyed on a per-call
// "has the model seen this content" check that adds an extra round-trip to
// every write; with it off, writes no longer wait on read evidence.
type ReadPipelineOptions struct {
	// LegacyCoordinator restores the legacy incomplete-read execution owner.
	LegacyCoordinator bool
	// LegacyEvidenceGates turns the writer-declared evidence check off. It is
	// the host-local rollback switch for the evidence gate; see the type
	// comment for the default direction.
	LegacyEvidenceGates bool
	// LegacyImplicitFullReads restores the old rule that a read with no window
	// promised the whole file. It exists for diagnosis and rollback only.
	LegacyImplicitFullReads bool
}

// defaultLegacyEvidenceGates reports whether the writer-declared evidence gate
// should be disabled for this run. The boot path may force it on (set
// opts.ReadPipeline.LegacyEvidenceGates = false) when an operator wants the
// pre-2026 strict-read-before-write behavior back.
func defaultLegacyEvidenceGates() bool {
	return true
}
