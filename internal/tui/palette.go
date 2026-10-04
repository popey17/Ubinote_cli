package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"ubinote-cli/internal/format"
)

type formatItem struct {
	action format.Action
	label  string
}

func (i formatItem) Title() string       { return i.label }
func (i formatItem) Description() string { return string(i.action) }
func (i formatItem) FilterValue() string { return i.label + " " + string(i.action) }

func newFormatPalette(width, height int) list.Model {
	items := make([]list.Item, 0, len(format.AllActions()))
	for _, a := range format.AllActions() {
		items = append(items, formatItem{action: a, label: format.ActionLabel(a)})
	}
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	l := list.New(items, delegate, width, height)
	l.Title = "Format"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7")).Bold(true)
	return l
}
