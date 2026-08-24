package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	modalFallbackWidth  = 32
	modalFallbackHeight = 9
)

type modalLayout struct {
	width, height               int
	contentWidth, contentHeight int
	fallback                    bool
}

// modalBounds centralizes the shallow geometry shared by overlays. Terminals
// below the usable threshold trade chrome for a text-only full-pane fallback.
func modalBounds(viewportWidth, viewportHeight, preferredWidth, preferredHeight int) modalLayout {
	viewportWidth = max(1, viewportWidth)
	viewportHeight = max(1, viewportHeight)
	fallback := viewportWidth < modalFallbackWidth || viewportHeight < modalFallbackHeight
	if fallback {
		return modalLayout{width: viewportWidth, height: viewportHeight, contentWidth: viewportWidth, contentHeight: viewportHeight, fallback: true}
	}
	// Lipgloss Width includes the frame chrome, so retain the shared outer
	// modal budget and derive the usable inner width from it.
	outerWidth := min(preferredWidth, viewportWidth)
	outerHeight := min(preferredHeight, viewportHeight)
	return modalLayout{
		width:         outerWidth,
		height:        outerHeight,
		contentWidth:  max(1, outerWidth-6),
		contentHeight: max(1, outerHeight-4),
	}
}

// renderModalFrame applies the common active frame or the text-only fallback.
// Dynamic rows are width-bounded before lipgloss receives them.
func renderModalFrame(rows []string, layout modalLayout) string {
	fitted := make([]string, 0, len(rows))
	for _, row := range rows {
		fitted = append(fitted, ansi.Truncate(row, layout.contentWidth, ""))
	}
	if layout.fallback && len(fitted) > layout.contentHeight {
		fitted = fitted[:layout.contentHeight]
	}
	body := strings.Join(fitted, "\n")
	if layout.fallback {
		return lipgloss.NewStyle().Width(layout.width).Height(layout.height).Render(body)
	}
	return lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color(paneBorderColor(true))).
		Padding(1, 2).
		Width(layout.width).
		Height(layout.contentHeight).
		Render(body)
}

// confirmationBody reserves the final action row and scrolls only the wrapped
// question when it cannot fit. The caller retains the offset as modal state.
func confirmationBody(title, question, actions string, layout modalLayout, offset int) ([]string, int) {
	wrappedText := wrapModalWords(question, layout.contentWidth)
	wrapped := make([]descriptionLine, len(wrappedText))
	for i, line := range wrappedText {
		wrapped[i].text = line
	}
	questionRows := max(1, layout.contentHeight-4) // title, two gaps, actions
	limit := max(0, len(wrapped)-questionRows)
	offset = clamp(offset, 0, limit)
	end := min(len(wrapped), offset+questionRows)
	body := []string{title, ""}
	for _, line := range wrapped[offset:end] {
		body = append(body, line.text)
	}
	body = append(body, "", fitConfirmationActions(actions, layout.contentWidth))
	return body, limit
}

// wrapModalWords keeps ordinary words intact when a confirmation copy wraps.
func wrapModalWords(text string, width int) []string {
	width = max(1, width)
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, len(words))
	line := ""
	for _, word := range words {
		if line != "" && lipgloss.Width(line+" "+word) > width {
			lines = append(lines, line)
			line = word
			continue
		}
		if line == "" && lipgloss.Width(word) > width {
			for lipgloss.Width(word) > width {
				part := truncatePlain(word, max(1, width-lipgloss.Width("…")))
				lines = append(lines, part+"…")
				word = strings.TrimPrefix(word, part)
			}
			line = word
			continue
		}
		if line == "" {
			line = word
		} else {
			line += " " + word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func fitConfirmationActions(actions string, width int) string {
	if lipgloss.Width(actions) <= width {
		return actions
	}
	for _, compact := range []string{"y confirm  •  n / Esc", "y / n", "y/n"} {
		if lipgloss.Width(compact) <= width {
			return compact
		}
	}
	return ansi.Truncate(actions, width, "")
}

// updateConfirmationScroll recognizes the shared confirmation-message motions.
func (m *model) updateConfirmationScroll(key string, limit int) bool {
	key = normalizeArrow(key)
	if key == "g" {
		if m.confirmationGArmed {
			m.confirmationScroll = 0
			m.confirmationGArmed = false
			return true
		}
		m.confirmationGArmed = true
		return true
	}
	m.confirmationGArmed = false
	switch key {
	case "G":
		m.confirmationScroll = limit
	case "j":
		m.confirmationScroll = clamp(m.confirmationScroll+1, 0, limit)
	case "k":
		m.confirmationScroll = clamp(m.confirmationScroll-1, 0, limit)
	case "ctrl+d":
		m.confirmationScroll = clamp(m.confirmationScroll+max(1, limit/2), 0, limit)
	case "ctrl+u":
		m.confirmationScroll = clamp(m.confirmationScroll-max(1, limit/2), 0, limit)
	default:
		return false
	}
	return true
}

func overlayModalLayout(background, modal string, width int, fallback bool) string {
	if !fallback {
		return overlayModal(background, modal, width)
	}
	backgroundLines := strings.Split(background, "\n")
	modalLines := strings.Split(modal, "\n")
	for i := range backgroundLines {
		line := ""
		if i < len(modalLines) {
			line = ansi.Truncate(modalLines[i], width, "")
		}
		backgroundLines[i] = line + strings.Repeat(" ", max(0, width-lipgloss.Width(line)))
	}
	return strings.Join(backgroundLines, "\n")
}
