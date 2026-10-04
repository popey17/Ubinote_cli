package ui

import "github.com/charmbracelet/lipgloss"

// Shared light-blue palette for welcome, login, and notes.
var (
	Blue     = lipgloss.Color("#87cefa") // light sky blue
	BlueHi   = lipgloss.Color("#b0e0ff")
	Green    = lipgloss.Color("#87cefa") // options use same accent family
	Red      = lipgloss.Color("#ff6b6b")
	Text     = lipgloss.Color("#e8f4ff")
	Body     = lipgloss.Color("#c5d9eb")
	Muted    = lipgloss.Color("#6b8499")
	Cursor   = lipgloss.Color("#87cefa")
	Border   = lipgloss.Color("#87cefa")
	Accent   = Blue
	AccentHi = BlueHi
	Amber    = Blue // legacy name used by theme/app
)

var (
	Brand = lipgloss.NewStyle().
		Foreground(Blue).
		Bold(true)

	BrandMark = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true)

	Title = lipgloss.NewStyle().
		Foreground(Blue).
		Bold(true)

	Subtitle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	Dim = lipgloss.NewStyle().
		Foreground(Muted)

	Error = lipgloss.NewStyle().
		Foreground(Red).
		Bold(true)

	Status = lipgloss.NewStyle().
		Foreground(Muted)

	Help = lipgloss.NewStyle().
		Foreground(Muted)

	Label = lipgloss.NewStyle().
		Foreground(BlueHi).
		Bold(true)

	Panel = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(Border).
		Padding(1, 2)

	Card = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(Blue).
		Padding(1, 3).
		Width(48)

	// Selected note — underline only, no background highlight.
	Selected = lipgloss.NewStyle().
			Foreground(BlueHi).
			Bold(true).
			Underline(true)

	SelectedDesc = lipgloss.NewStyle().
			Foreground(Muted)

	NormalItem = lipgloss.NewStyle().
			Foreground(Text)

	HeaderBar = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true).
			Padding(0, 1)

	FooterBar = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 1)

	AccentHiStyle = lipgloss.NewStyle().
			Foreground(BlueHi).
			Bold(true)
)
