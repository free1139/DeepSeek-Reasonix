package config

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestStartupOnlineDefaultsToFalse(t *testing.T) {
	if got := Default().StartupOnline(); got {
		t.Fatalf("Default().StartupOnline() = true, want false (offline by default)")
	}
	if got := (&Config{}).StartupOnline(); got {
		t.Fatalf("empty Config.StartupOnline() = true, want false")
	}
	if got := (*Config)(nil).StartupOnline(); got {
		t.Fatalf("nil Config.StartupOnline() = true, want false")
	}
}

func TestStartupOnlineExplicit(t *testing.T) {
	cfg := &Config{Startup: StartupConfig{Online: true}}
	if !cfg.StartupOnline() {
		t.Fatalf("explicit online=true should report true")
	}
	cfg.Startup.Online = false
	if cfg.StartupOnline() {
		t.Fatalf("explicit online=false should report false")
	}
}

// TestStartupOnlineFromTOMLMissingSection confirms legacy configs without a
// [startup] section keep the safe offline default.
func TestStartupOnlineFromTOMLMissingSection(t *testing.T) {
	const legacy = `
default_model = "deepseek-flash"

[telemetry]
cli_metrics = "off"
`
	var cfg Config
	if _, err := toml.Decode(legacy, &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.StartupOnline() {
		t.Fatalf("legacy config should resolve StartupOnline=false")
	}
}

func TestStartupOnlineFromTOMLOnlineTrue(t *testing.T) {
	const src = `
[startup]
online = true
`
	var cfg Config
	if _, err := toml.Decode(src, &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !cfg.StartupOnline() {
		t.Fatalf("explicit [startup] online=true should report true")
	}
}

func TestStartupOnlineFromTOMLOnlineFalse(t *testing.T) {
	const src = `
[startup]
online = false
`
	var cfg Config
	if _, err := toml.Decode(src, &cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.StartupOnline() {
		t.Fatalf("explicit [startup] online=false should report false")
	}
}

func TestStartupOnlineFromTOMLUnknownFieldIgnored(t *testing.T) {
	const src = `
[startup]
online = true
unknown_future_field = "x"
`
	var cfg Config
	if _, err := toml.Decode(src, &cfg); err != nil {
		t.Fatalf("decode rejected unknown future field: %v", err)
	}
	if !cfg.StartupOnline() {
		t.Fatalf("online should still be true after decode with unknown field")
	}
}

func TestRenderStartupSectionMentionsOnline(t *testing.T) {
	out := RenderTOML(Default())
	if !strings.Contains(out, "[startup]") {
		t.Fatalf("rendered default config should mention [startup]; got:\n%s", out)
	}
	if !strings.Contains(out, "online") {
		t.Fatalf("rendered default config should mention online; got:\n%s", out)
	}
}
