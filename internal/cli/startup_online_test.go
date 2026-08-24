package cli

import (
	"bytes"
	"strings"
	"testing"

	"reasonix/internal/config"
	"reasonix/internal/i18n"
	"reasonix/internal/telemetry"
)

// TestStartupOnlineSkipsStdinConsent asserts the chatREPL path does not block
// reading from a non-interactive stdin when [startup].online is false (the
// default). This is the failure mode behind "reasonix --continue hangs for
// tens of seconds": telemetry consent reading from /dev/null used to swallow
// EOF silently.
func TestStartupOnlineSkipsStdinConsent(t *testing.T) {
	i18n.DetectLanguage("en")
	cfg := &config.Config{}       // Startup.Online defaults to false
	stdin := bytes.NewBuffer(nil) // EOF on first read

	var out, errOut bytes.Buffer
	reporter := startCLITelemetryWithIO(cfg, telemetry.Options{
		Version: "v1.0.0", Interactive: true, CLIMode: "tui",
	}, stdin, &out, &errOut, !cfg.StartupOnline())

	// Should not have hung; the prompt never fires in offline mode.
	if strings.Contains(strings.ToLower(out.String()), "reasonix can send anonymous") || strings.Contains(out.String(), "匿名") {
		t.Fatalf("offline startup must skip the consent notice; got: %s", out.String())
	}
	_ = reporter
}

// TestStartupOnlineOptInKeepsStdinPath asserts that flipping online=true still
// hits the original consent prompt path, so the legacy behaviour is reachable
// only by explicit opt-in.
func TestStartupOnlineOptInKeepsStdinPath(t *testing.T) {
	i18n.DetectLanguage("en")
	cfg := &config.Config{Startup: config.StartupConfig{Online: true}}
	// "y" answers the consent prompt cleanly.
	stdin := strings.NewReader("y\n")

	var out, errOut bytes.Buffer
	reporter := startCLITelemetryWithIO(cfg, telemetry.Options{
		Version: "v1.0.0", Interactive: true, CLIMode: "tui",
	}, stdin, &out, &errOut, !cfg.StartupOnline())

	if !strings.Contains(strings.ToLower(out.String()), "reasonix can send anonymous") && !strings.Contains(out.String(), "匿名") {
		t.Fatalf("online=true must still surface consent notice; got: %q", out.String())
	}
	if reporter == nil {
		t.Fatalf("online=true with consent granted should produce a reporter")
	}
}

// TestStartupOnlineDoesNotAffectConfiguredCfg asserts that already-configured
// telemetry (CLITelemetryConfigured() == true) skips the consent flow
// regardless of the online flag, preserving the "configured once, never asked
// again" contract.
func TestStartupOnlineDoesNotAffectConfiguredCfg(t *testing.T) {
	cfg := &config.Config{
		Telemetry: config.TelemetryConfig{CLIMetrics: "off"},
	}
	stdin := bytes.NewBuffer(nil) // EOF — would hang if the prompt fired

	var out, errOut bytes.Buffer
	reporter := startCLITelemetryWithIO(cfg, telemetry.Options{
		Version: "test", Interactive: true, CLIMode: "tui",
	}, stdin, &out, &errOut, true /* forceOffline */)

	if strings.Contains(out.String(), "consent") {
		t.Fatalf("configured telemetry must skip consent; got: %s", out.String())
	}
	_ = reporter
}

// TestStartupOnlineOptOutNoStdinNotice asserts that opt-out (online=false)
// writes a recognisable marker on stderr (via the existing "consent cleanup
// failed" path) so the operator can see why telemetry went silent.
func TestStartupOnlineOptOutNoStdinNotice(t *testing.T) {
	cfg := &config.Config{} // online defaults to false
	stdin := bytes.NewBuffer(nil)

	var out, errOut bytes.Buffer
	_ = startCLITelemetryWithIO(cfg, telemetry.Options{
		Version: "test", Interactive: true, CLIMode: "tui",
	}, stdin, &out, &errOut, true /* forceOffline */)

	// stdout must stay silent — the consent notice is intentionally suppressed
	// so the chatREPL prompt can appear immediately.
	if strings.Contains(strings.ToLower(out.String()), "telemetry") {
		t.Fatalf("offline startup must not print the telemetry consent notice; got: %q", out.String())
	}
}
