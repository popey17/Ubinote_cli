package ui

import "github.com/charmbracelet/lipgloss"

var (
	Accent = lipgloss.Color("#7aa2f7")
	Text   = lipgloss.Color("#c0caf5")
	Muted  = lipgloss.Color("#565f89")
	Green  = lipgloss.Color("#9ece6a")
	Red    = lipgloss.Color("#f7768e")
	Border = lipgloss.Color("#414868")
)

var (
	Title = lipgloss.NewStyle().
		Foreground(Accent).
		Bold(true)

	Dim = lipgloss.NewStyle().
		Foreground(Muted)

	Error = lipgloss.NewStyle().
		Foreground(Red)

	Status = lipgloss.NewStyle().
		Foreground(Muted)

	Help = lipgloss.NewStyle().
		Foreground(Muted)

	Panel = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(Border).
		Padding(0, 1)

	Selected = lipgloss.NewStyle().
		Foreground(Accent).
		Bold(true)

	NormalItem = lipgloss.NewStyle().
			Foreground(Text)
)
