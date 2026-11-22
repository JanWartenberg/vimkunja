package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// These menus share navigation without owning the underlying draft or selection.
func navigateManagement(key string, g *bool, current, last, visible int) (int, bool) {
	key = normalizeArrow(key)
	if *g {
		*g = false
		if key == "g" {
			return 0, true
		}
	}
	switch key {
	case "g":
		*g = true
	case "home":
		current = 0
	case "G", "end":
		current = last
	case "j":
		current++
	case "k":
		current--
	case "ctrl+d", "pgdown", "pagedown":
		current += max(1, visible/2)
	case "ctrl+u", "pgup", "pageup":
		current -= max(1, visible/2)
	default:
		return current, false
	}
	return clamp(current, 0, max(0, last)), true
}

func managementSize(w, h int) (int, int, bool) {
	if w >= 16 && h >= 8 {
		return w - 4, h - 4, true
	}
	return w, h, false
}

func frameManagement(content string, width int, framed bool) string {
	if !framed {
		return content
	}
	lines := strings.Split(content, "\n")
	padded := []string{strings.Repeat(" ", width+2)}
	for i, line := range lines {
		if i == len(lines)-1 {
			padded = append(padded, strings.Repeat("─", width+2))
		}
		padded = append(padded, " "+line+" ")
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(paneBorderColor(true))).Render(strings.Join(padded, "\n"))
}

func managementHint(hint string, width int) string {
	return renderFooterHints(width, "", strings.Split(hint, " · "), false)
}

// Resolve local Ex commands through the same aliases/prefixes as the main editor.
func (m *model) resolveManagementCommand(input string) (string, error) {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return "", nil
	}
	if len(fields) != 1 {
		return "", fmt.Errorf("command takes no arguments; :h explains available commands")
	}
	if name := normalizeCommandName(fields[0]); name == "bucketdelete" || name == "x" {
		return name, nil
	}
	command, _ := m.commands.resolve(fields[0])
	if command == nil {
		return "", fmt.Errorf("unknown or ambiguous command: %s; :h for help", fields[0])
	}
	return command.canonical, nil
}

func managementReferenceRows(entries []helpEntry, w int) []string {
	var rows []string
	keyWidth := 0
	for _, entry := range entries {
		keyWidth = max(keyWidth, ansi.StringWidth(entry.keys))
	}
	for _, entry := range entries {
		if w < 48 {
			rows = append(rows, wrapModalWords(entry.keys, w)...)
			for _, line := range wrapModalWords(entry.text, max(1, w-2)) {
				rows = append(rows, "  "+line)
			}
			continue
		}
		prefix := padPlainRight(entry.keys, keyWidth) + " — "
		for i, line := range wrapModalWords(entry.text, max(1, w-keyWidth-3)) {
			if i == 0 {
				rows = append(rows, prefix+line)
			} else {
				rows = append(rows, strings.Repeat(" ", keyWidth+3)+line)
			}
		}
	}
	return rows
}

func managementObjectLabel(name string, id int64, width int) string {
	suffix := fmt.Sprintf(" (#%d)", id)
	if width <= ansi.StringWidth(suffix) {
		return ansi.Truncate(fmt.Sprintf("#%d", id), max(0, width), "")
	}
	return ansi.Truncate(sanitizeTerminalText(name), width-ansi.StringWidth(suffix), "…") + suffix
}

func (m *model) managementProjectIdentity(id int64, width int) string {
	for _, p := range m.projects {
		if p.ID == id {
			return managementObjectLabel(p.Title, id, width)
		}
	}
	return managementObjectLabel("Project", id, width)
}
