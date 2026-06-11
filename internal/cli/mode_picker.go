package cli

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// modePicker is the in-chat overlay for the /audit command. It shows the three
// mode options (Auto / Plan / YOLO) and lets the user pick one with ↑/↓ + Enter,
// equivalent to cycling via Shift+Tab. chatTUI routes keystrokes through
// handleModePickerKey while m.modePicker is set, and renders it via
// renderModePicker in the pinned bottom region.
type modePicker struct {
	sel int // 0=Auto, 1=Plan, 2=YOLO
}

const (
	modeAuto = iota
	modePlan
	modeYOLO
)

var modeLabels = []string{"Auto", "Plan", "YOLO"}

// modeColors matches the status-line mode tags from View().
var modeColors = []string{
	modeAuto: statusAutoColor.hex,
	modePlan: statusPlanColor.hex,
	modeYOLO: statusYoloColor.hex,
}

// openModePicker populates the picker with the three mode options and
// pre-selects the currently active mode.
func (m *chatTUI) openModePicker() {
	sel := modeAuto
	switch {
	case m.ctrl.Bypass():
		sel = modeYOLO
	case m.planMode:
		sel = modePlan
	}
	m.modePicker = &modePicker{sel: sel}
}

// handleModePickerKey routes keys while the mode picker is open.
func (m chatTUI) handleModePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.modePicker = nil
		return m, nil
	case "up", "k":
		if m.modePicker.sel > 0 {
			m.modePicker.sel--
		}
		return m, nil
	case "down", "j":
		if m.modePicker.sel < modeYOLO {
			m.modePicker.sel++
		}
		return m, nil
	case "enter":
		m.applyModePickerSelection()
		return m, nil
	}
	return m, nil
}

// applyModePickerSelection sets the TUI mode to the selected option and closes
// the picker.
func (m *chatTUI) applyModePickerSelection() {
	sel := m.modePicker.sel
	m.modePicker = nil
	m.setModeByArg(modeLabels[sel])
}

// setModeByArg switches the TUI mode based on the argument (auto/plan/yolo),
// case-insensitive. Unknown values are ignored silently.
func (m *chatTUI) setModeByArg(arg string) {
	switch strings.ToLower(arg) {
	case "auto":
		m.planMode = false
		m.ctrl.SetPlanMode(false)
		m.ctrl.SetBypass(false)
		m.notice("mode: Auto")
	case "plan":
		m.planMode = true
		m.ctrl.ClearGoal()
		m.ctrl.SetPlanMode(true)
		m.ctrl.SetBypass(false)
		m.notice("mode: Plan")
	case "yolo":
		m.planMode = false
		m.ctrl.SetPlanMode(false)
		m.ctrl.SetBypass(true)
		m.notice("mode: YOLO")
	}
}

// renderModePicker renders the mode-selection card when the picker is open.
func (m chatTUI) renderModePicker() string {
	if m.modePicker == nil {
		return ""
	}
	w := max(m.width, 10)
	var b strings.Builder
	b.WriteString(accent("Mode") + "\n")
	for i, label := range modeLabels {
		cur := i == m.modePicker.sel
		prefix := "  "
		if cur {
			prefix = accent("❯ ")
		}
		// Color the label to match the mode tag style
		body := lipgloss.NewStyle().
			Background(lipgloss.Color(modeColors[i])).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1).
			Render(label)
		b.WriteString(prefix + body + "\n")
	}
	b.WriteString(dim("↑/↓ navigate · Enter select · Esc dismiss"))
	return choicePanelStyle.Width(w).Render(b.String())
}
