package model

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

type logMsg string

const klArt = `
██╗  ██╗██╗     
██║ ██╔╝██║     
█████╔╝ ██║     
██╔═██╗ ██║     
██║  ██╗███████╗
╚═╝  ╚═╝╚══════╝ v1.0.0
`

var (
	headerGradient = lipgloss.Blend1D(lipgloss.Width(klArt), lipgloss.Color("#FF6AC1"), lipgloss.Color("#5AC8FA"))
	infoGradient   = lipgloss.Blend1D(20, lipgloss.Color("#04B575"), lipgloss.Color("#5AC8FA"))

	headerStyle = lipgloss.NewStyle().
			Bold(true)

	infoStyle = lipgloss.
			NewStyle().
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#04B575"))

	autoscrollStyle = lipgloss.
			NewStyle().
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	formatStyle = lipgloss.
			NewStyle().
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)

	filterStyle = lipgloss.
			NewStyle().
			Bold(true).
			PaddingLeft(1).
			PaddingRight(1)
)

func applyGradient(text string, baseStyle lipgloss.Style, colors []color.Color) string {
	var b strings.Builder
	s := lipgloss.NewStyle()

	for i, r := range text {
		j := i % len(colors)
		b.WriteString(s.Foreground(colors[j]).Render(string(r)))
	}

	return baseStyle.Render(b.String())
}

func (m Model) headerView() string {
	header := applyGradient(klArt, lipgloss.NewStyle(), headerGradient)

	info := infoStyle.Render(fmt.Sprintf("%s/%s", m.kl.Namespace(), m.kl.Service()))

	if f := m.kl.Filter(); f != "" {
		filter := filterStyle.Render(fmt.Sprintf("Filter: %s", f))
		info = lipgloss.JoinHorizontal(lipgloss.Left, info, filter)
	}

	format := applyGradient(fmt.Sprintf("[f] Format: %v", m.format), formatStyle, infoGradient)
	info = lipgloss.JoinHorizontal(lipgloss.Left, info, format)

	autoscroll := applyGradient(fmt.Sprintf("[s] Autoscroll: %v", m.autoscroll), autoscrollStyle, infoGradient)
	info = lipgloss.JoinHorizontal(lipgloss.Left, info, autoscroll)

	if m.viewport.AtBottom() {
		arrowDown := autoscrollStyle.Render("▼")
		info = lipgloss.JoinHorizontal(lipgloss.Left, info, arrowDown)
	}

	return header + "\n" + info
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	return m.headerView() + "\n\n" + m.viewport.View()
}
