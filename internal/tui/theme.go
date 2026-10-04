package tui

import (
	"github.com/charmbracelet/lipgloss"

	"ubinote-cli/internal/ui"
)

// Welcome + login styles — same light-blue palette as notes.
var (
	themeTitle  = lipgloss.NewStyle().Foreground(ui.Blue).Bold(true)
	themeQuit   = lipgloss.NewStyle().Foreground(ui.Red).Bold(true)
	themeOpt    = lipgloss.NewStyle().Foreground(ui.BlueHi)
	themeHover  = lipgloss.NewStyle().Foreground(ui.BlueHi).Bold(true)
	themePrompt = lipgloss.NewStyle().Foreground(ui.Text)
	themeBody   = lipgloss.NewStyle().Foreground(ui.Body)
	themeMuted  = lipgloss.NewStyle().Foreground(ui.Muted)
	themeInput  = lipgloss.NewStyle().Foreground(ui.Text)
	themeCursor = lipgloss.NewStyle().Foreground(ui.Cursor).Background(ui.Cursor)
	themeBorder = ui.Border
)

const ubinoteBanner = `
██╗   ██╗██████╗ ██╗███╗   ██╗ ██████╗ ████████╗███████╗
██║   ██║██╔══██╗██║████╗  ██║██╔═══██╗╚══██╔══╝██╔════╝
██║   ██║██████╔╝██║██╔██╗ ██║██║   ██║   ██║   █████╗  
██║   ██║██╔══██╗██║██║╚██╗██║██║   ██║   ██║   ██╔══╝  
╚██████╔╝██████╔╝██║██║ ╚████║╚██████╔╝   ██║   ███████╗
 ╚═════╝ ╚═════╝ ╚═╝╚═╝  ╚═══╝ ╚═════╝    ╚═╝   ╚══════╝`

type hitZone struct {
	Y0, Y1 int
	X0, X1 int // optional; X1 < 0 means full width
	Action string
}

func hitTest(zones []hitZone, x, y int) string {
	for _, z := range zones {
		if y < z.Y0 || y > z.Y1 {
			continue
		}
		if z.X1 >= 0 && (x < z.X0 || x > z.X1) {
			continue
		}
		return z.Action
	}
	return ""
}
