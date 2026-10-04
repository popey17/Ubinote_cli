package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"ubinote-cli/internal/api"
	"ubinote-cli/internal/ui"
)

type mode int

const (
	modeLogin mode = iota
	modeBrowse
	modeEdit
	modeConfirmDelete
)

type noteItem struct {
	id    string
	title string
}

func (n noteItem) Title() string       { return n.title }
func (n noteItem) Description() string { return n.id }
func (n noteItem) FilterValue() string { return n.title }

type model struct {
	client   *api.Client
	mode     mode
	login    loginModel
	list     list.Model
	viewport viewport.Model
	editor   editorModel
	notes    []api.Note
	current  *api.Note
	status   string
	errMsg   string
	width    int
	height   int
	ready    bool
}

type notesLoadedMsg struct {
	notes []api.Note
	err   error
}

type noteLoadedMsg struct {
	note api.Note
	err  error
}

type noteDeletedMsg struct {
	err error
}

func Run(client *api.Client) error {
	m := newApp(client)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

func RunEditor(client *api.Client, id, title, body string) error {
	ed := newEditor(id, title, body, 80, 24)
	ed.standalone = true
	ed.focus = 1
	ed.title.Blur()
	ed.body.Focus()
	p := tea.NewProgram(standaloneEditor{client: client, editor: ed}, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return err
	}
	if se, ok := final.(standaloneEditor); ok {
		if se.editor.canceled() {
			return fmt.Errorf("cancelled")
		}
	}
	return nil
}

type standaloneEditor struct {
	client *api.Client
	editor editorModel
}

func (s standaloneEditor) Init() tea.Cmd { return s.editor.Init() }

func (s standaloneEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.editor.layout(msg.Width, msg.Height)
		return s, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return s, tea.Quit
		}
	}
	var cmd tea.Cmd
	prevDirty := s.editor.dirty
	s.editor, cmd = s.editor.Update(msg, s.client)
	_ = prevDirty
	if s.editor.quit {
		return s, tea.Quit
	}
	// detect cancel exit without save when esc and not dirty handled inside editor
	if !s.editor.dirty && !s.editor.saved && !s.editor.confirmEsc {
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "esc" {
			s.editor.quit = true
			return s, tea.Quit
		}
	}
	return s, cmd
}

func (s standaloneEditor) View() string { return s.editor.View() }

func newApp(client *api.Client) model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = ui.Selected
	delegate.Styles.SelectedDesc = ui.Dim
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Notes"
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	m := model{
		client: client,
		login:  newLoginModel(),
		list:   l,
	}
	if client.Token() == "" {
		m.mode = modeLogin
	} else {
		m.mode = modeBrowse
	}
	return m
}

func (m model) Init() tea.Cmd {
	if m.mode == modeLogin {
		return m.login.Init()
	}
	return m.loadNotes()
}

func (m model) loadNotes() tea.Cmd {
	return func() tea.Msg {
		notes, err := m.client.ListNotes(context.Background())
		return notesLoadedMsg{notes: notes, err: err}
	}
}

func (m model) loadNote(id string) tea.Cmd {
	return func() tea.Msg {
		note, err := m.client.GetNote(context.Background(), id)
		return noteLoadedMsg{note: note, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.layout()
		if m.mode == modeBrowse && m.current != nil {
			m.renderCurrent()
		}
		return m, nil

	case loginAttemptMsg:
		return m, loginWithClient(m.client, msg.email, msg.password)

	case loginSuccessMsg:
		m.client.SetToken(msg.token)
		m.mode = modeBrowse
		m.errMsg = ""
		return m, m.loadNotes()

	case loginFailMsg:
		var cmd tea.Cmd
		m.login, cmd = m.login.Update(msg)
		return m, cmd

	case notesLoadedMsg:
		if msg.err != nil {
			if apiErrUnauthorized(msg.err) {
				m.mode = modeLogin
				m.errMsg = "session expired"
				return m, nil
			}
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.notes = msg.notes
		m.errMsg = ""
		items := make([]list.Item, len(msg.notes))
		for i, n := range msg.notes {
			items[i] = noteItem{id: n.ID, title: n.Title}
		}
		m.list.SetItems(items)
		if len(msg.notes) > 0 {
			m.list.Select(0)
			return m, m.loadNote(msg.notes[0].ID)
		}
		m.current = nil
		m.viewport.SetContent(ui.Dim.Render("No notes yet. Press n to create one."))
		return m, nil

	case noteLoadedMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		n := msg.note
		m.current = &n
		m.renderCurrent()
		return m, nil

	case noteSavedMsg:
		m.mode = modeBrowse
		m.status = "saved"
		return m, m.loadNotes()

	case noteSaveFailMsg:
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg, m.client)
		return m, cmd

	case noteDeletedMsg:
		m.mode = modeBrowse
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return m, nil
		}
		m.status = "deleted"
		m.current = nil
		return m, m.loadNotes()

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.mode {
		case modeLogin:
			if msg.String() == "q" {
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.login, cmd = m.login.Update(msg)
			return m, cmd
		case modeConfirmDelete:
			switch msg.String() {
			case "y", "Y":
				id := ""
				if m.current != nil {
					id = m.current.ID
				}
				m.mode = modeBrowse
				return m, func() tea.Msg {
					return noteDeletedMsg{err: m.client.DeleteNote(context.Background(), id)}
				}
			case "n", "N", "esc":
				m.mode = modeBrowse
				return m, nil
			}
			return m, nil
		case modeEdit:
			var cmd tea.Cmd
			wasConfirm := m.editor.confirmEsc
			m.editor, cmd = m.editor.Update(msg, m.client)
			// esc without dirty / after discard → back to browse
			if msg.String() == "esc" && !m.editor.dirty && !m.editor.confirmEsc && !wasConfirm {
				m.mode = modeBrowse
				return m, nil
			}
			if wasConfirm && !m.editor.confirmEsc && !m.editor.dirty && msg.String() == "y" {
				m.mode = modeBrowse
				return m, nil
			}
			return m, cmd
		case modeBrowse:
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "e":
				if m.current == nil {
					return m, nil
				}
				m.editor = newEditor(m.current.ID, m.current.Title, m.current.Body, m.rightWidth(), m.height-2)
				m.editor.focus = 1
				m.editor.title.Blur()
				m.editor.body.Focus()
				m.mode = modeEdit
				return m, m.editor.Init()
			case "n":
				m.editor = newEditor("", "Untitled", "", m.rightWidth(), m.height-2)
				m.editor.focus = 0
				m.mode = modeEdit
				return m, m.editor.Init()
			case "d":
				if m.current == nil {
					return m, nil
				}
				m.mode = modeConfirmDelete
				return m, nil
			case "enter", "right":
				if item, ok := m.list.SelectedItem().(noteItem); ok {
					return m, m.loadNote(item.id)
				}
			}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			// auto-load on selection change via list
			if item, ok := m.list.SelectedItem().(noteItem); ok {
				if m.current == nil || m.current.ID != item.id {
					return m, tea.Batch(cmd, m.loadNote(item.id))
				}
			}
			return m, cmd
		}
	}

	// fallback updates
	switch m.mode {
	case modeLogin:
		var cmd tea.Cmd
		m.login, cmd = m.login.Update(msg)
		return m, cmd
	case modeEdit:
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg, m.client)
		return m, cmd
	case modeBrowse:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		m.viewport, _ = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *model) layout() {
	listW := m.width / 3
	if listW < 24 {
		listW = 24
	}
	if listW > m.width/2 {
		listW = m.width / 2
	}
	rightW := m.width - listW - 2
	h := m.height - 2
	m.list.SetSize(listW, h)
	m.viewport = viewport.New(rightW, h)
	if m.mode == modeEdit {
		m.editor.layout(rightW, h)
	}
}

func (m model) rightWidth() int {
	listW := m.width / 3
	if listW < 24 {
		listW = 24
	}
	return max(20, m.width-listW-2)
}

func (m *model) renderCurrent() {
	if m.current == nil {
		m.viewport.SetContent("")
		return
	}
	body := m.current.Body
	if strings.TrimSpace(body) == "" {
		body = "_Nothing here yet._"
	}
	width := m.viewport.Width
	if width < 20 {
		width = 40
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-2),
	)
	content := body
	if err == nil {
		if out, err2 := r.Render(body); err2 == nil {
			content = out
		}
	}
	header := ui.Title.Render(m.current.Title) + "\n\n"
	m.viewport.SetContent(header + content)
	m.viewport.GotoTop()
}

func (m model) View() string {
	if !m.ready && m.mode != modeLogin {
		return "loading…"
	}
	switch m.mode {
	case modeLogin:
		return m.login.View()
	case modeConfirmDelete:
		title := ""
		if m.current != nil {
			title = m.current.Title
		}
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			ui.Panel.Render(fmt.Sprintf("Delete %q?\n\ny confirm · n cancel", title)))
	case modeEdit:
		left := m.list.View()
		right := m.editor.View()
		return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right) + "\n" + m.footer()
	default:
		left := m.list.View()
		right := ui.Panel.Width(m.viewport.Width).Height(m.viewport.Height).Render(m.viewport.View())
		body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
		return body + "\n" + m.footer()
	}
}

func (m model) footer() string {
	if m.errMsg != "" {
		return ui.Error.Render(m.errMsg)
	}
	if m.status != "" {
		return ui.Status.Render(m.status)
	}
	if m.mode == modeEdit {
		return ui.Help.Render("ctrl+s save · esc cancel · ctrl+p format · tab focus")
	}
	return ui.Help.Render("↑↓ navigate · enter open · e edit · n new · d delete · / filter · q quit")
}

func apiErrUnauthorized(err error) bool {
	return errors.Is(err, api.ErrUnauthorized)
}
