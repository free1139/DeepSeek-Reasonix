package cli

import (
	"time"

	"reasonix/internal/agent"
)

// reclaimCLIRecoveryBranches performs the same conservative recovery-copy
// hygiene as Desktop. Failures are intentionally silent: cleanup is optional,
// and any lease, concurrent save, I/O error, or failed revalidation either
// preserves the live branch or leaves a durable hidden stage for startup repair.
//
// When wait is false the scan and trash run in a detached goroutine and the
// caller returns immediately. The startup path passes false so a 100k-entry
// session dir does not block first-frame; /resume, /continue, /resume <n>, and
// the picker pass true to keep their existing synchronous semantics, since
// those surfaces already expect a small bounded wait before the picker shows.
func reclaimCLIRecoveryBranches(dir string, wait bool) {
	if !wait {
		go reclaimCLIRecoveryBranches(dir, true)
		return
	}
	candidates, err := agent.ReclaimableRecoveryBranches(dir, time.Now(), agent.RecoveryGCGracePeriod)
	if err != nil {
		return
	}
	for _, path := range candidates {
		_ = agent.TrashReclaimableRecoveryBranch(path, dir)
	}
}
