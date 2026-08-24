package cli

import (
	"os"
	"path/filepath"
	"strings"
)

const lastSessionFilename = "last-session"

// lastSessionPath is the per-session-dir pointer file storing the absolute
// path of the session most recently selected by --continue or by an in-chat
// switch. It lives next to the .jsonl transcripts, not at the state root, so
// a per-workspace session dir carries its own last-session pointer and a
// project reset does not inherit the global one.
func lastSessionPath(sessionDir string) string {
	if sessionDir == "" {
		return ""
	}
	return filepath.Join(sessionDir, lastSessionFilename)
}

// readLastSession returns the absolute session path stored in the pointer
// file, or ("", false) when the file is missing, empty, or unreadable. The
// caller is responsible for any further validation of the path; this helper
// does not stat the target session so a transient disk hiccup on the
// pointer file falls straight back to mtime without burning extra IO.
func readLastSession(sessionDir string) (string, bool) {
	path := lastSessionPath(sessionDir)
	if path == "" {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	first := strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0])
	if first == "" {
		return "", false
	}
	return first, true
}

// writeLastSession overwrites the pointer file with the absolute path of
// sessionPath plus a trailing newline. Permission matches the surrounding
// transcript metadata (0o600). Errors are returned to the caller; chat
// shutdown treats them as advisory and swallows them.
func writeLastSession(sessionDir, sessionPath string) error {
	path := lastSessionPath(sessionDir)
	if path == "" || sessionPath == "" {
		return nil
	}
	return os.WriteFile(path, []byte(sessionPath+"\n"), 0o600)
}
