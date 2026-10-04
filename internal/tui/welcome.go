package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type welcomeDoneMsg struct{}

type welcomePane int

const (
	welcomeMenu welcomePane = iota
	welcomeHelp
	welcomeInfo
)

type welcomeModel struct {
	width       int
	height      int
	apiURL      string
	pane        welcomePane
	errMsg      string
	hover       string
	selectedIdx int // 0=l 1=h 2=i 3=q
}

func newWelcomeModel(apiURL string) welcomeModel {
	return welcomeModel{apiURL: apiURL, pane: welcomeMenu, selectedIdx: 0}
}

func (m welcomeModel) Init() tea.Cmd { return nil }

// Fixed layout rows (with padTop=1) so mouse hit-testing matches the paint.
func welcomeZones(padTop int) []hitZone {
	bannerLines := len(strings.Split(strings.TrimSpace(ubinoteBanner), "\n"))
	y := padTop
	y += bannerLines // banner
	y++              // credit
	y++              // blank
	quitY := y
	y++ // quit
	y++ // blank
	lY, hY, iY := y, y+1, y+2
	return []hitZone{
		{Y0: quitY, Y1: quitY, X0: 0, X1: -1, Action: "q"},
		{Y0: lY, Y1: lY, X0: 0, X1: -1, Action: "l"},
		{Y0: hY, Y1: hY, X0: 0, X1: -1, Action: "h"},
		{Y0: iY, Y1: iY, X0: 0, X1: -1, Action: "i"},
	}
}

func (m welcomeModel) Update(msg tea.Msg) (welcomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.MouseMsg:
		if m.pane != welcomeMenu {
			if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
				m.pane = welcomeMenu
				m.errMsg = ""
			}
			return m, nil
		}
		action := hitTest(welcomeZones(1), msg.X, msg.Y)
		switch msg.Action {
		case tea.MouseActionMotion:
			if action != "" {
				m.hover = action
				m.selectedIdx = actionIndex(action)
			} else {
				m.hover = ""
			}
			return m, nil
		case tea.MouseActionPress:
			if msg.Button == tea.MouseButtonLeft && action != "" {
				return m.runAction(action)
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.pane == welcomeHelp || m.pane == welcomeInfo {
			switch msg.String() {
			case "esc", "enter", "q", "b", " ":
				m.pane = welcomeMenu
				m.errMsg = ""
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
			m.hover = optionAction(m.selectedIdx)
			return m, nil
		case "down", "j":
			if m.selectedIdx < 3 {
				m.selectedIdx++
			}
			m.hover = optionAction(m.selectedIdx)
			return m, nil
		case "enter", " ":
			return m.runAction(optionAction(m.selectedIdx))
		case "l", "h", "i", "q":
			return m.runAction(msg.String())
		}
	}
	return m, nil
}

func optionAction(idx int) string {
	switch idx {
	case 0:
		return "l"
	case 1:
		return "h"
	case 2:
		return "i"
	default:
		return "q"
	}
}

func actionIndex(action string) int {
	switch action {
	case "l":
		return 0
	case "h":
		return 1
	case "i":
		return 2
	case "q":
		return 3
	default:
		return 0
	}
}

func (m welcomeModel) runAction(cmd string) (welcomeModel, tea.Cmd) {
	m.errMsg = ""
	switch cmd {
	case "l", "login":
		m.selectedIdx = 0
		return m, func() tea.Msg { return welcomeDoneMsg{} }
	case "h", "help":
		m.selectedIdx = 1
		m.pane = welcomeHelp
		return m, nil
	case "i", "info":
		m.selectedIdx = 2
		m.pane = welcomeInfo
		return m, nil
	case "q", "quit":
		m.selectedIdx = 3
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m welcomeModel) styleOption(action, label string) string {
	active := m.hover == action || (m.hover == "" && optionAction(m.selectedIdx) == action)
	if active {
		return themeHover.Render("▸ " + label)
	}
	return themeOpt.Render("  " + label)
}

func (m welcomeModel) View() string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	switch m.pane {
	case welcomeHelp:
		return m.overlayView(w, h, "Help", helpText())
	case welcomeInfo:
		return m.overlayView(w, h, "Info", infoText(m.apiURL))
	}

	padTop, padLeft := 1, 2
	var b strings.Builder
	b.WriteString(themeTitle.Render(strings.TrimSpace(ubinoteBanner)))
	b.WriteByte('\n')
	b.WriteString(themeTitle.Render("                                  - markdown notes for your terminal"))
	b.WriteString("\n\n")
	b.WriteString(themeQuit.Render("Click an option  ·  q to quit"))
	b.WriteString("\n\n")
	b.WriteString(m.styleOption("l", "Login to your notes"))
	b.WriteByte('\n')
	b.WriteString(m.styleOption("h", "Help — keys & shortcuts"))
	b.WriteByte('\n')
	b.WriteString(m.styleOption("i", "Info — about ubinote"))
	b.WriteString("\n\n")
	b.WriteString(themeMuted.Render("mouse: click  ·  keys: ↑↓ enter  ·  q quit"))

	padded := lipgloss.NewStyle().Padding(padTop, padLeft).Render(b.String())
	return lipgloss.NewStyle().Width(w).Height(h).Render(padded)
}

func (m welcomeModel) overlayView(w, h int, title, body string) string {
	card := lipgloss.JoinVertical(lipgloss.Left,
		themeTitle.Render(title),
		"",
		themeBody.Render(body),
		"",
		themeMuted.Render("click or press enter/esc to go back"),
	)
	boxed := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(themeBorder).
		Padding(1, 2).
		Render(card)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, boxed)
}

func helpText() string {
	return strings.TrimSpace(`
Main menu
  click / l   login
  click / h   help
  click / i   info
  click / q   quit

After login (notes)
  ↑↓          navigate notes
  e           edit note
  n           new note
  d           delete note
  r           refresh from API
  L           logout
  /           filter list
  q           quit

Editor
  ctrl+s      save
  esc         cancel
  ctrl+p      format palette
`)
}

func infoText(apiURL string) string {
	return strings.TrimSpace(`
ubinote — personal Markdown notes in the terminal.

Same account and notes as the web app.
Talks to your personal_note API over HTTP.

API: ` + apiURL + `

Stack: Go · cobra · Bubble Tea · glamour
`)
}
