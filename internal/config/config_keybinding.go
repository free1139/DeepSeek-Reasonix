package config

// KeybindingsConfig allows overriding transcript-scroll keys. Empty fields fall
// back to the defaults below; set a field to change it.
//
// Terminal considerations:
//   - Alt-prefixed keys (alt+up etc.) are unreliable because terminals cannot
//     distinguish Alt from Esc followed by a letter — bubbletea never sees them.
//   - Ctrl+arrow keys conflict with macOS system shortcuts (Mission Control).
//   - Ctrl+letter (ctrl+h/j) and PgUp/PgDn are reliable on all terminals.
type KeybindingsConfig struct {
	ScrollUp   string `toml:"scroll_up"`   // default "ctrl+h"
	ScrollDown string `toml:"scroll_down"` // default "ctrl+j"
	PageUp     string `toml:"page_up"`     // default "pgup"
	PageDown   string `toml:"page_down"`   // default "pgdown"
	GotoTop    string `toml:"goto_top"`    // default "ctrl+home"
	GotoBottom string `toml:"goto_bottom"` // default "ctrl+end"
}

// FillDefaults replaces every empty field with its working default so
// callers can just compare against the field without a second default lookup.
func (k *KeybindingsConfig) FillDefaults() {
	if k.ScrollUp == "" {
		k.ScrollUp = "ctrl+h"
	}
	if k.ScrollDown == "" {
		k.ScrollDown = "ctrl+j"
	}
	if k.PageUp == "" {
		k.PageUp = "pgup"
	}
	if k.PageDown == "" {
		k.PageDown = "pgdown"
	}
	if k.GotoTop == "" {
		k.GotoTop = "ctrl+home"
	}
	if k.GotoBottom == "" {
		k.GotoBottom = "ctrl+end"
	}
}
