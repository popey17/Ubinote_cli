package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/popey17/Ubinote_cli/internal/format"
)

func BindingFor(msg tea.KeyMsg) (format.Action, bool) {
	switch msg.String() {
	case "ctrl+b":
		return format.Bold, true
	case "ctrl+i":
		return format.Italic, true
	case "ctrl+k":
		return format.Link, true
	case "alt+1":
		return format.H1, true
	case "alt+2":
		return format.H2, true
	case "alt+3":
		return format.H3, true
	case "ctrl+shift+c":
		return format.CodeBlock, true
	case "alt+c":
		return format.Code, true
	case "alt+u":
		return format.UL, true
	case "alt+o":
		return format.OL, true
	case "alt+t":
		return format.Check, true
	case "ctrl+]":
		return format.Indent, true
	case "ctrl+[":
		return format.Outdent, true
	case "alt+q":
		return format.Quote, true
	default:
		return "", false
	}
}
