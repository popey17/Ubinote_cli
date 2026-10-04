package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/popey17/Ubinote_cli/internal/api"
	"github.com/popey17/Ubinote_cli/internal/format"
	"github.com/popey17/Ubinote_cli/internal/ui"
)

type editorModel struct {
	title      textinput.Model
	body       textarea.Model
	focus      int // 0 title, 1 body
	noteID     string
	isNew      bool
	dirty      bool
	saving     bool
	confirmEsc bool
	errMsg     string
	status     string
	width      int
	height     int
	palette    list.Model
	showPal    bool
	standalone bool
	saved      bool
	quit       bool
}

type noteSavedMsg struct {
	note api.Note
}

type noteSaveFailMsg struct {
	err error
}

func newEditor(noteID, title, body string, width, height int) editorModel {
	ti := textinput.New()
	ti.Placeholder = "Title"
	ti.CharLimit = 200
	ti.SetValue(title)
	ti.Focus()

	ta := textarea.New()
	ta.Placeholder = "Write Markdown…"
	ta.SetValue(body)
	ta.CharLimit = 0
	ta.ShowLineNumbers = false

	m := editorModel{
		title:  ti,
		body:   ta,
		focus:  0,
		noteID: noteID,
		isNew:  noteID == "",
		width:  width,
		height: height,
	}
	m.palette = newFormatPalette(40, 12)
	m.layout(width, height)
	return m
}

func (m *editorModel) layout(width, height int) {
	m.width = width
	m.height = height
	m.title.Width = max(20, width-8)
	bodyH := max(5, height-10)
	m.body.SetWidth(max(20, width-4))
	m.body.SetHeight(bodyH)
	m.palette.SetSize(min(48, width-4), min(14, height-4))
}

func (m editorModel) Init() tea.Cmd { return textinput.Blink }

func (m editorModel) Update(msg tea.Msg, client *api.Client) (editorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case noteSavedMsg:
		m.saving = false
		m.dirty = false
		m.saved = true
		m.noteID = msg.note.ID
		m.isNew = false
		m.status = "saved"
		if m.standalone {
			m.quit = true
			return m, tea.Quit
		}
		return m, nil
	case noteSaveFailMsg:
		m.saving = false
		m.errMsg = msg.err.Error()
		return m, nil
	case tea.KeyMsg:
		if m.showPal {
			switch msg.String() {
			case "esc", "ctrl+p":
				m.showPal = false
				return m, nil
			case "enter":
				if item, ok := m.palette.SelectedItem().(formatItem); ok {
					m = m.applyFormat(item.action)
				}
				m.showPal = false
				return m, nil
			}
			var cmd tea.Cmd
			m.palette, cmd = m.palette.Update(msg)
			return m, cmd
		}

		if m.confirmEsc {
			switch msg.String() {
			case "y", "Y":
				m.confirmEsc = false
				m.dirty = false
				if m.standalone {
					m.quit = true
					return m, tea.Quit
				}
				return m, nil
			case "n", "N", "esc":
				m.confirmEsc = false
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+s":
			title := strings.TrimSpace(m.title.Value())
			body := m.body.Value()
			if title == "" || strings.TrimSpace(body) == "" {
				m.errMsg = "title and body are required"
				return m, nil
			}
			m.saving = true
			m.errMsg = ""
			return m, saveNote(client, m.noteID, m.isNew, title, body)
		case "esc":
			if m.dirty {
				m.confirmEsc = true
				return m, nil
			}
			if m.standalone {
				m.quit = true
				return m, tea.Quit
			}
			return m, nil
		case "ctrl+p":
			m.showPal = true
			return m, nil
		case "tab":
			m.focus = 1 - m.focus
			if m.focus == 0 {
				m.body.Blur()
				m.title.Focus()
			} else {
				m.title.Blur()
				m.body.Focus()
			}
			return m, nil
		}

		if m.focus == 1 {
			if action, ok := BindingFor(msg); ok {
				m = m.applyFormat(action)
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	if m.focus == 0 {
		prev := m.title.Value()
		m.title, cmd = m.title.Update(msg)
		if m.title.Value() != prev {
			m.dirty = true
		}
	} else {
		prev := m.body.Value()
		m.body, cmd = m.body.Update(msg)
		if m.body.Value() != prev {
			m.dirty = true
		}
	}
	return m, cmd
}

func (m editorModel) applyFormat(action format.Action) editorModel {
	val := m.body.Value()
	// bubbles textarea has no selection API; apply at cursor
	start, end := cursorRange(m.body)
	res := format.Apply(action, val, start, end)
	m.body.SetValue(res.Value)
	m.dirty = true
	m.status = "applied " + format.ActionLabel(action)
	m.focus = 1
	m.title.Blur()
	m.body.Focus()
	return m
}

func cursorRange(ta textarea.Model) (start, end int) {
	// Approximate absolute offset from line info and value.
	val := ta.Value()
	info := ta.LineInfo()
	line := ta.Line()
	lines := strings.Split(val, "\n")
	if line < 0 {
		line = 0
	}
	if line >= len(lines) {
		return len(val), len(val)
	}
	offset := 0
	for i := 0; i < line && i < len(lines); i++ {
		offset += len(lines[i]) + 1
	}
	col := info.CharOffset
	if col < 0 {
		col = 0
	}
	if col > len(lines[line]) {
		col = len(lines[line])
	}
	start = offset + col
	if start > len(val) {
		start = len(val)
	}
	return start, start
}

func saveNote(client *api.Client, id string, isNew bool, title, body string) tea.Cmd {
	return func() tea.Msg {
		var (
			note api.Note
			err  error
		)
		if isNew {
			note, err = client.CreateNote(context.Background(), title, body)
		} else {
			note, err = client.UpdateNote(context.Background(), id, title, body)
		}
		if err != nil {
			return noteSaveFailMsg{err: err}
		}
		return noteSavedMsg{note: note}
	}
}

func (m editorModel) View() string {
	if m.showPal {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			ui.Panel.Render(m.palette.View()))
	}

	var b strings.Builder
	mode := "edit"
	if m.isNew {
		mode = "new"
	}
	b.WriteString(ui.Title.Render("ubinote · "+mode) + "\n")
	b.WriteString(m.title.View() + "\n\n")
	b.WriteString(m.body.View() + "\n")
	if m.confirmEsc {
		b.WriteString(ui.Error.Render("Discard changes? y/n"))
	} else if m.errMsg != "" {
		b.WriteString(ui.Error.Render(m.errMsg))
	} else if m.status != "" {
		b.WriteString(ui.Status.Render(m.status))
	} else {
		b.WriteString(ui.Help.Render("ctrl+s save · esc cancel · ctrl+p format · tab focus"))
	}
	return b.String()
}

// canceled reports whether the user left without saving (standalone).
func (m editorModel) canceled() bool {
	return m.quit && !m.saved
}
