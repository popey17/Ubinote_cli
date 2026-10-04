package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/popey17/Ubinote_cli/internal/config"
)

type loginModel struct {
	email    textinput.Model
	password textinput.Model
	focus    int // 0 email, 1 password, 2 submit, 3 toggle, 4 back
	errMsg   string
	loading  bool
	slow     bool // true after delayed wake-up warning
	loadGen  int
	register bool
	width    int
	height   int
	apiURL   string
	hover    string
}

type loginSlowMsg struct {
	gen int
}

type loginSuccessMsg struct {
	token string
}

type loginFailMsg struct {
	err error
}

type loginBackMsg struct{}

func newLoginModel(apiURL string) loginModel {
	email := textinput.New()
	email.Placeholder = "you@example.com"
	email.CharLimit = 256
	email.Width = 40
	email.Prompt = ""
	email.TextStyle = themeInput
	email.PlaceholderStyle = themeMuted
	email.Cursor.Style = themeCursor
	email.Focus()

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 256
	password.Width = 40
	password.Prompt = ""
	password.TextStyle = themeInput
	password.PlaceholderStyle = themeMuted
	password.Cursor.Style = themeCursor

	return loginModel{email: email, password: password, apiURL: apiURL}
}

func (m loginModel) Init() tea.Cmd { return textinput.Blink }

func loginZones(padTop int, register bool) []hitZone {
	bannerLines := len(strings.Split(strings.TrimSpace(ubinoteBanner), "\n"))
	y := padTop
	y += bannerLines
	y++ // credit
	y++ // blank
	y++ // mode line
	y++ // blank
	// Email label
	y++
	emailY := y // input line (border makes this approximate; use wider band)
	y += 3      // bordered input ~3 rows
	y++         // blank
	y++         // password label
	passY := y
	y += 3
	y++ // blank
	submitY := y
	y++
	toggleY := y
	y++
	backY := y
	_ = register
	return []hitZone{
		{Y0: emailY - 1, Y1: emailY + 1, X0: 0, X1: -1, Action: "email"},
		{Y0: passY - 1, Y1: passY + 1, X0: 0, X1: -1, Action: "password"},
		{Y0: submitY, Y1: submitY, X0: 0, X1: -1, Action: "submit"},
		{Y0: toggleY, Y1: toggleY, X0: 0, X1: -1, Action: "toggle"},
		{Y0: backY, Y1: backY, X0: 0, X1: -1, Action: "back"},
	}
}

func (m loginModel) Update(msg tea.Msg) (loginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case loginFailMsg:
		m.loading = false
		m.slow = false
		m.loadGen++
		m.errMsg = msg.err.Error()
		return m, nil

	case loginSlowMsg:
		if msg.gen != m.loadGen || !m.loading {
			return m, nil
		}
		m.slow = true
		return m, nil

	case loginSuccessMsg:
		// Handled by app; clear local loading state if it arrives here.
		m.loading = false
		m.slow = false
		return m, nil

	case tea.MouseMsg:
		if m.loading {
			return m, nil
		}
		action := hitTest(loginZones(1, m.register), msg.X, msg.Y)
		switch msg.Action {
		case tea.MouseActionMotion:
			m.hover = action
			return m, nil
		case tea.MouseActionPress:
			if msg.Button != tea.MouseButtonLeft || action == "" {
				return m, nil
			}
			switch action {
			case "email":
				m.focus = 0
				m.password.Blur()
				return m, m.email.Focus()
			case "password":
				m.focus = 1
				m.email.Blur()
				return m, m.password.Focus()
			case "submit":
				return m.submit()
			case "toggle":
				m.register = !m.register
				m.errMsg = ""
				return m, nil
			case "back":
				return m, func() tea.Msg { return loginBackMsg{} }
			}
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch msg.String() {
		case "esc", "ctrl+c":
			return m, func() tea.Msg { return loginBackMsg{} }
		case "q":
			if m.focus > 1 {
				return m, func() tea.Msg { return loginBackMsg{} }
			}
		case "ctrl+r", "ctrl+l":
			m.register = !m.register
			m.errMsg = ""
			return m, nil
		case "tab", "down":
			m.focus = (m.focus + 1) % 5
			return m, m.syncFocus()
		case "shift+tab", "up":
			m.focus = (m.focus + 4) % 5
			return m, m.syncFocus()
		case "enter":
			switch m.focus {
			case 0:
				m.focus = 1
				return m, m.syncFocus()
			case 1, 2:
				return m.submit()
			case 3:
				m.register = !m.register
				m.errMsg = ""
				return m, nil
			case 4:
				return m, func() tea.Msg { return loginBackMsg{} }
			}
		}
	}

	if m.focus > 1 {
		return m, nil
	}
	var cmd tea.Cmd
	if m.focus == 0 {
		m.email, cmd = m.email.Update(msg)
	} else {
		m.password, cmd = m.password.Update(msg)
	}
	return m, cmd
}

func (m *loginModel) syncFocus() tea.Cmd {
	switch m.focus {
	case 0:
		m.password.Blur()
		return m.email.Focus()
	case 1:
		m.email.Blur()
		return m.password.Focus()
	default:
		m.email.Blur()
		m.password.Blur()
		return nil
	}
}

func (m loginModel) submit() (loginModel, tea.Cmd) {
	email := strings.TrimSpace(m.email.Value())
	pass := m.password.Value()
	if email == "" || pass == "" {
		m.errMsg = "email and password required"
		return m, nil
	}
	m.loading = true
	m.slow = false
	m.errMsg = ""
	m.loadGen++
	gen := m.loadGen
	return m, tea.Batch(
		doAuth(email, pass, m.register),
		tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return loginSlowMsg{gen: gen}
		}),
	)
}

func doAuth(email, password string, register bool) tea.Cmd {
	return func() tea.Msg {
		return loginAttemptMsg{email: email, password: password, register: register}
	}
}

type loginAttemptMsg struct {
	email, password string
	register        bool
}

func (m loginModel) btn(action, label string) string {
	active := m.hover == action || (m.hover == "" && focusAction(m.focus) == action)
	if active {
		return themeHover.Render("▸ " + label)
	}
	return themeOpt.Render("  " + label)
}

func focusAction(focus int) string {
	switch focus {
	case 2:
		return "submit"
	case 3:
		return "toggle"
	case 4:
		return "back"
	default:
		return ""
	}
}

func (m loginModel) View() string {
	w, h := m.width, m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	mode := "Sign in"
	toggle := "Create an account instead"
	submit := "Sign in"
	if m.register {
		mode = "Create account"
		toggle = "Sign in instead"
		submit = "Create account"
	}

	emailFocused := m.focus == 0 || m.hover == "email"
	passFocused := m.focus == 1 || m.hover == "password"

	var b strings.Builder
	b.WriteString(themeTitle.Render(strings.TrimSpace(ubinoteBanner)))
	b.WriteByte('\n')
	b.WriteString(themeTitle.Render("                                  - markdown notes for your terminal"))
	b.WriteString("\n\n")
	b.WriteString(themeQuit.Render(mode + "  ·  esc back"))
	b.WriteString("\n\n")

	b.WriteString(m.fieldLabel("Email", emailFocused))
	b.WriteByte('\n')
	b.WriteString(m.fieldBox(m.email.View(), emailFocused))
	b.WriteString("\n\n")
	b.WriteString(m.fieldLabel("Password", passFocused))
	b.WriteByte('\n')
	b.WriteString(m.fieldBox(m.password.View(), passFocused))
	b.WriteString("\n\n")

	b.WriteString(m.btn("submit", submit))
	b.WriteByte('\n')
	b.WriteString(m.btn("toggle", toggle))
	b.WriteByte('\n')
	b.WriteString(m.btn("back", "Back to main menu"))
	b.WriteString("\n\n")

	switch {
	case m.loading && m.slow:
		b.WriteString(themeQuit.Render("server is waking up — this can take a moment…"))
	case m.loading:
		b.WriteString(themeMuted.Render("connecting…"))
	case m.errMsg != "":
		b.WriteString(themeQuit.Render("✗ " + m.errMsg))
	default:
		b.WriteString(themeMuted.Render("mouse: click fields/buttons  ·  tab/enter  ·  " + m.apiURL))
	}

	padded := lipgloss.NewStyle().Padding(1, 2).Render(b.String())
	return lipgloss.NewStyle().Width(w).Height(h).Render(padded)
}

func (m loginModel) fieldLabel(label string, focused bool) string {
	if focused {
		return themeHover.Render("▸ " + label)
	}
	return themeOpt.Render("  " + label)
}

func (m loginModel) fieldBox(content string, focused bool) string {
	border := lipgloss.Color("#555555")
	if focused {
		border = themeBorder
	}
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Width(44).
		Render(content)
}

func persistToken(token string) error {
	return config.SaveCredentials(config.Credentials{Token: token})
}

func loginWithClient(client loginClient, email, password string, register bool) tea.Cmd {
	return func() tea.Msg {
		if register {
			if _, _, err := client.Register(context.Background(), email, password); err != nil {
				return loginFailMsg{err: err}
			}
		}
		tok, err := client.Login(context.Background(), email, password)
		if err != nil {
			return loginFailMsg{err: err}
		}
		if err := persistToken(tok); err != nil {
			return loginFailMsg{err: err}
		}
		return loginSuccessMsg{token: tok}
	}
}

type loginClient interface {
	Login(ctx context.Context, email, password string) (string, error)
	Register(ctx context.Context, email, password string) (id, emailOut string, err error)
	SetToken(token string)
}
