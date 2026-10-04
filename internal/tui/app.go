package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/popey17/Ubinote_cli/internal/api"
	"github.com/popey17/Ubinote_cli/internal/config"
	"github.com/popey17/Ubinote_cli/internal/ui"
)

type mode int

const (
	modeWelcome mode = iota
	modeLogin
	modeBrowse
	modeEdit
	modeConfirmDelete
	modeConfirmLogout
)

type loggedOutMsg struct{}

type noteItem struct {
	id    string
	title string
}

func (n noteItem) Title() string       { return n.title }
func (n noteItem) Description() string { return "" }
func (n noteItem) FilterValue() string { return n.title }

type model struct {
	client       *api.Client
	mode         mode
	welcome      welcomeModel
	login        loginModel
	list         list.Model
	viewport     viewport.Model
	editor       editorModel
	notes        []api.Note
	notesByID    map[string]api.Note
	current      *api.Note
	status       string
	errMsg       string
	width        int
	height       int
	ready        bool
	renderer     *glamour.TermRenderer
	renderW      int
	renderCache  map[string]string // noteID -> rendered body for current width
	notesFetched bool              // true after first list attempt finishes
	notesLoading bool
	loadGen      int // invalidates delayed "waking up" warnings
}

type notesLoadedMsg struct {
	notes []api.Note
	err   error
}

type noteDeletedMsg struct {
	err error
}

type clearStatusMsg struct{}

type slowLoadMsg struct {
	gen int
}

const (
	statusLoadingNotes = "loading notes…"
	statusRefreshing   = "refreshing…"
	statusWakingUp     = "server is waking up — this can take a moment…"
	slowLoadAfter      = 3 * time.Second
)

func flashStatus(text string) (string, tea.Cmd) {
	return text, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func warnIfSlow(gen int) tea.Cmd {
	return tea.Tick(slowLoadAfter, func(time.Time) tea.Msg {
		return slowLoadMsg{gen: gen}
	})
}

func Run(client *api.Client) error {
	m := newApp(client)
	// Mouse clicks/wheel only — cell motion floods Update and makes list nav feel laggy.
	// Mouse enabled for welcome/login clicks; browse ignores motion (see Update).
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
	delegate.ShowDescription = false
	delegate.SetHeight(1)
	delegate.SetSpacing(1)
	// Underline only — no background highlight.
	delegate.Styles.SelectedTitle = ui.Selected.Padding(0, 1)
	delegate.Styles.SelectedDesc = ui.SelectedDesc
	delegate.Styles.NormalTitle = ui.NormalItem.Padding(0, 1)
	delegate.Styles.NormalDesc = ui.Dim
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Notes"
	l.Styles.Title = ui.HeaderBar
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 0)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.Blue)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.BlueHi)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(false)

	apiURL := client.BaseURL()
	m := model{
		client:      client,
		welcome:     newWelcomeModel(apiURL),
		login:       newLoginModel(apiURL),
		list:        l,
		notesByID:   map[string]api.Note{},
		renderCache: map[string]string{},
	}
	if client.Token() == "" {
		m.mode = modeWelcome
	} else {
		m.mode = modeBrowse
	}
	return m
}

func (m model) Init() tea.Cmd {
	switch m.mode {
	case modeWelcome:
		return m.welcome.Init()
	case modeLogin:
		return m.login.Init()
	default:
		return m.startLoadNotes()
	}
}

func (m model) startLoadNotes() tea.Cmd {
	return func() tea.Msg {
		return notesLoadStartMsg{}
	}
}

type notesLoadStartMsg struct{}

func (m model) loadNotes() tea.Cmd {
	return func() tea.Msg {
		notes, err := m.client.ListNotes(context.Background())
		return notesLoadedMsg{notes: notes, err: err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.welcome.width = msg.Width
		m.welcome.height = msg.Height
		m.login.width = msg.Width
		m.login.height = msg.Height
		m.layout()
		// Width change invalidates wrap cache.
		m.renderCache = map[string]string{}
		m.renderer = nil
		var cmds []tea.Cmd
		if m.mode == modeBrowse {
			if m.current != nil {
				m.renderCurrent()
			}
			// If Init fetch finished before we had a size, re-paint; if it never ran, fetch now.
			if !m.notesFetched && !m.notesLoading {
				cmds = append(cmds, m.startLoadNotes())
			}
		}
		return m, tea.Batch(cmds...)

	case welcomeDoneMsg:
		m.mode = modeLogin
		m.login = newLoginModel(m.client.BaseURL())
		m.login.width = m.width
		m.login.height = m.height
		return m, m.login.Init()

	case loginBackMsg:
		m.mode = modeWelcome
		m.welcome = newWelcomeModel(m.client.BaseURL())
		m.welcome.width = m.width
		m.welcome.height = m.height
		return m, m.welcome.Init()

	case loginAttemptMsg:
		return m, loginWithClient(m.client, msg.email, msg.password, msg.register)

	case clearStatusMsg:
		m.status = ""
		return m, nil

	case slowLoadMsg:
		if msg.gen != m.loadGen || !m.notesLoading {
			return m, nil
		}
		m.status = statusWakingUp
		return m, nil

	case notesLoadStartMsg:
		if m.notesLoading {
			return m, nil
		}
		m.notesLoading = true
		m.loadGen++
		gen := m.loadGen
		if m.status == "" || m.status == statusRefreshing {
			if m.status != statusRefreshing {
				m.status = statusLoadingNotes
			}
		}
		return m, tea.Batch(m.loadNotes(), warnIfSlow(gen))

	case loginSuccessMsg:
		m.login.loading = false
		m.login.slow = false
		m.client.SetToken(msg.token)
		m.mode = modeBrowse
		m.errMsg = ""
		m.notesFetched = false
		var statusCmd tea.Cmd
		m.status, statusCmd = flashStatus("welcome back")
		return m, tea.Batch(m.startLoadNotes(), statusCmd)

	case loginFailMsg:
		var cmd tea.Cmd
		m.login, cmd = m.login.Update(msg)
		return m, cmd

	case loginSlowMsg:
		var cmd tea.Cmd
		m.login, cmd = m.login.Update(msg)
		return m, cmd

	case notesLoadedMsg:
		m.notesLoading = false
		m.notesFetched = true
		m.loadGen++ // cancel pending slow-load warning
		if msg.err != nil {
			if apiErrUnauthorized(msg.err) {
				m.mode = modeWelcome
				m.errMsg = "session expired"
				m.status = ""
				return m, m.welcome.Init()
			}
			m.errMsg = msg.err.Error()
			switch m.status {
			case statusLoadingNotes, statusRefreshing, statusWakingUp:
				m.status = ""
			}
			return m, nil
		}
		m.notes = msg.notes
		m.notesByID = make(map[string]api.Note, len(msg.notes))
		m.renderCache = map[string]string{}
		m.errMsg = ""
		items := make([]list.Item, len(msg.notes))
		for i, n := range msg.notes {
			m.notesByID[n.ID] = n
			items[i] = noteItem{id: n.ID, title: n.Title}
		}
		// Ensure list/viewport have real dimensions before painting.
		if m.width > 0 && m.height > 0 {
			m.layout()
		}
		m.list.SetItems(items)
		var statusCmd tea.Cmd
		// Only announce "refreshed" for manual r; keep flash from save/delete/login.
		switch m.status {
		case statusRefreshing:
			m.status, statusCmd = flashStatus("refreshed")
		case statusLoadingNotes, statusWakingUp:
			m.status = ""
		case "welcome back", "saved", "deleted":
			// Restart the clear timer after list reload finishes.
			_, statusCmd = flashStatus(m.status)
		}
		keepID := ""
		if m.current != nil {
			keepID = m.current.ID
		}
		if len(msg.notes) > 0 {
			idx := 0
			if keepID != "" {
				for i, n := range msg.notes {
					if n.ID == keepID {
						idx = i
						break
					}
				}
			}
			m.list.Select(idx)
			m.selectNote(msg.notes[idx].ID)
			return m, statusCmd
		}
		m.current = nil
		m.viewport.SetContent(ui.Dim.Render("No notes yet. Press n to create one."))
		return m, statusCmd

	case noteSavedMsg:
		m.mode = modeBrowse
		m.notesFetched = false
		var statusCmd tea.Cmd
		m.status, statusCmd = flashStatus("saved")
		return m, tea.Batch(m.startLoadNotes(), statusCmd)

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
		m.notesFetched = false
		var statusCmd tea.Cmd
		m.status, statusCmd = flashStatus("deleted")
		m.current = nil
		return m, tea.Batch(m.startLoadNotes(), statusCmd)

	case loggedOutMsg:
		m.client.SetToken("")
		m.notes = nil
		m.notesByID = map[string]api.Note{}
		m.current = nil
		m.list.SetItems(nil)
		m.renderCache = map[string]string{}
		m.errMsg = ""
		m.status = ""
		m.mode = modeWelcome
		m.welcome = newWelcomeModel(m.client.BaseURL())
		m.welcome.width = m.width
		m.welcome.height = m.height
		m.login = newLoginModel(m.client.BaseURL())
		m.login.width = m.width
		m.login.height = m.height
		return m, m.welcome.Init()

	case tea.MouseMsg:
		// Keep note browsing snappy — ignore hover spam; allow wheel on viewport/list.
		if m.mode == modeBrowse || m.mode == modeEdit {
			if msg.Action == tea.MouseActionMotion {
				return m, nil
			}
			if tea.MouseEvent(msg).IsWheel() {
				var cmd tea.Cmd
				if m.mode == modeBrowse {
					m.list, cmd = m.list.Update(msg)
					m.viewport, _ = m.viewport.Update(msg)
				}
				return m, cmd
			}
			return m, nil
		}
		switch m.mode {
		case modeWelcome:
			var cmd tea.Cmd
			m.welcome, cmd = m.welcome.Update(msg)
			return m, cmd
		case modeLogin:
			var cmd tea.Cmd
			m.login, cmd = m.login.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.mode {
		case modeWelcome:
			var cmd tea.Cmd
			m.welcome, cmd = m.welcome.Update(msg)
			return m, cmd
		case modeLogin:
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
		case modeConfirmLogout:
			switch msg.String() {
			case "y", "Y":
				return m, doLogout(m.client)
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
			case "L":
				m.mode = modeConfirmLogout
				return m, nil
			case "r":
				m.status = statusRefreshing
				m.errMsg = ""
				m.notesFetched = false
				return m, m.startLoadNotes()

			case "enter", "right":
				if item, ok := m.list.SelectedItem().(noteItem); ok {
					m.selectNote(item.id)
				}
				return m, nil
			}
			prevID := ""
			if m.current != nil {
				prevID = m.current.ID
			}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			if item, ok := m.list.SelectedItem().(noteItem); ok && item.id != prevID {
				// List already includes body — select locally, no network round-trip.
				m.selectNote(item.id)
			}
			return m, cmd
		}
	}

	// fallback updates
	switch m.mode {
	case modeWelcome:
		var cmd tea.Cmd
		m.welcome, cmd = m.welcome.Update(msg)
		return m, cmd
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
	rightW := m.width - listW - 3
	// Reserve rows for app chrome (brand header + footer).
	h := max(5, m.height-3)
	m.list.SetSize(listW, h)
	m.viewport = viewport.New(max(20, rightW-2), h-2)
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

func (m *model) selectNote(id string) {
	n, ok := m.notesByID[id]
	if !ok {
		return
	}
	m.current = &n
	m.errMsg = ""
	m.renderCurrent()
}

func (m *model) ensureRenderer(width int) {
	if width < 20 {
		width = 40
	}
	if m.renderer != nil && m.renderW == width {
		return
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-2),
	)
	if err != nil {
		m.renderer = nil
		m.renderW = 0
		return
	}
	m.renderer = r
	m.renderW = width
	m.renderCache = map[string]string{}
}

func (m *model) renderCurrent() {
	if m.current == nil {
		m.viewport.SetContent("")
		return
	}
	header := ui.Title.Underline(true).Render(m.current.Title) + "\n\n"
	if cached, ok := m.renderCache[m.current.ID]; ok {
		m.viewport.SetContent(header + cached)
		m.viewport.GotoTop()
		return
	}

	body := m.current.Body
	if strings.TrimSpace(body) == "" {
		body = "_Nothing here yet._"
	}
	width := m.viewport.Width
	m.ensureRenderer(width)
	content := body
	if m.renderer != nil {
		if out, err := m.renderer.Render(body); err == nil {
			content = out
		}
	}
	m.renderCache[m.current.ID] = content
	m.viewport.SetContent(header + content)
	m.viewport.GotoTop()
}

func (m model) View() string {
	if !m.ready && m.mode != modeWelcome && m.mode != modeLogin {
		return lipgloss.Place(max(m.width, 40), max(m.height, 10), lipgloss.Center, lipgloss.Center,
			ui.Brand.Render("ubinote")+"\n"+ui.Dim.Render("loading…"))
	}
	switch m.mode {
	case modeWelcome:
		return m.welcome.View()
	case modeLogin:
		return m.login.View()
	case modeConfirmDelete:
		title := ""
		if m.current != nil {
			title = m.current.Title
		}
		card := ui.Card.Render(lipgloss.JoinVertical(lipgloss.Center,
			ui.Title.Render("Delete note?"),
			"",
			ui.Subtitle.Render(title),
			"",
			ui.Help.Render("y confirm  ·  n cancel"),
		))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
	case modeConfirmLogout:
		card := ui.Card.Render(lipgloss.JoinVertical(lipgloss.Center,
			ui.Title.Render("Log out?"),
			"",
			ui.Subtitle.Render("You’ll return to the welcome screen."),
			"",
			ui.Help.Render("y confirm  ·  n cancel"),
		))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, card)
	case modeEdit:
		return m.appChrome(
			lipgloss.JoinHorizontal(lipgloss.Top, m.list.View(), " ", m.editor.View()),
		)
	default:
		left := m.list.View()
		right := ui.Panel.
			Width(m.viewport.Width + 4).
			Height(m.viewport.Height + 2).
			BorderForeground(ui.Border).
			Render(m.viewport.View())
		return m.appChrome(lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right))
	}
}

func (m model) appChrome(body string) string {
	count := len(m.notes)
	brand := ui.Brand.Render("◆ ubinote")
	meta := ui.AccentHiStyle.Render(fmt.Sprintf("%d notes", count))
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		brand,
		strings.Repeat(" ", max(1, m.width-lipgloss.Width(brand)-lipgloss.Width(meta)-2)),
		meta,
	)
	rule := lipgloss.NewStyle().Foreground(ui.Blue).Render(strings.Repeat("─", max(10, m.width)))
	return header + "\n" + rule + "\n" + body + "\n" + m.footer()
}

func (m model) footer() string {
	bar := ui.FooterBar
	if m.errMsg != "" {
		return bar.Render(ui.Error.Render(m.errMsg))
	}
	if m.status != "" {
		return bar.Render(ui.Status.Render(m.status))
	}
	if m.mode == modeEdit {
		return bar.Render("ctrl+s save · esc cancel · ctrl+p format · tab focus")
	}
	return bar.Render("↑↓ navigate · e edit · n new · d delete · r refresh · L logout · / filter · q quit")
}

func doLogout(client *api.Client) tea.Cmd {
	return func() tea.Msg {
		if client.Token() != "" {
			_ = client.Logout(context.Background())
		}
		_ = config.ClearCredentials()
		return loggedOutMsg{}
	}
}

func apiErrUnauthorized(err error) bool {
	return errors.Is(err, api.ErrUnauthorized)
}
