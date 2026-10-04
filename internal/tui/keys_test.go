package tui_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ubinote-cli/internal/format"
	"ubinote-cli/internal/tui"
)

func TestBindingForBold(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyCtrlB}
	a, ok := tui.BindingFor(msg)
	if !ok || a != format.Bold {
		t.Fatalf("got %v %v", a, ok)
	}
}

func TestBindingForHeading(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}, Alt: true}
	a, ok := tui.BindingFor(msg)
	if !ok || a != format.H1 {
		t.Fatalf("got %v %v", a, ok)
	}
}

func TestBindingForUnknown(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	if _, ok := tui.BindingFor(msg); ok {
		t.Fatal("expected no binding")
	}
}
