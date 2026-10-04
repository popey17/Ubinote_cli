package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"ubinote-cli/internal/config"
	"ubinote-cli/internal/ui"
)

type loginModel struct {
	email    textinput.Model
	password textinput.Model
	focus    int
	errMsg   string
	loading  bool
}

type loginSuccessMsg struct {
	token string
}

type loginFailMsg struct {
	err error
}

func newLoginModel() loginModel {
	email := textinput.New()
	email.Placeholder = "email"
	email.CharLimit = 256
	email.Width = 40
	email.Focus()

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '•'
	password.CharLimit = 256
	password.Width = 40

	return loginModel{email: email, password: password}
}

func (m loginModel) Init() tea.Cmd { return textinput.Blink }

func (m loginModel) Update(msg tea.Msg) (loginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case loginFailMsg:
		m.loading = false
		m.errMsg = msg.err.Error()
		return m, nil
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch msg.String() {
		case "tab", "shift+tab", "up", "down":
			m.focus = 1 - m.focus
			if m.focus == 0 {
				m.password.Blur()
				m.email.Focus()
			} else {
				m.email.Blur()
				m.password.Focus()
			}
			return m, nil
		case "enter":
			if m.focus == 0 {
				m.email.Blur()
				m.password.Focus()
				m.focus = 1
				return m, nil
			}
			email := strings.TrimSpace(m.email.Value())
			pass := m.password.Value()
			if email == "" || pass == "" {
				m.errMsg = "email and password required"
				return m, nil
			}
			m.loading = true
			m.errMsg = ""
			return m, doLogin(email, pass)
		}
	}

	var cmd tea.Cmd
	if m.focus == 0 {
		m.email, cmd = m.email.Update(msg)
	} else {
		m.password, cmd = m.password.Update(msg)
	}
	return m, cmd
}

func doLogin(email, password string) tea.Cmd {
	return func() tea.Msg {
		// client is injected via package-level during Run — see app.go loginCmd
		return loginAttemptMsg{email: email, password: password}
	}
}

type loginAttemptMsg struct {
	email, password string
}

func (m loginModel) View() string {
	var b strings.Builder
	b.WriteString(ui.Title.Render("ubinote login"))
	b.WriteString("\n\n")
	b.WriteString(m.email.View())
	b.WriteString("\n")
	b.WriteString(m.password.View())
	b.WriteString("\n\n")
	if m.loading {
		b.WriteString(ui.Status.Render("signing in…"))
	} else if m.errMsg != "" {
		b.WriteString(ui.Error.Render(m.errMsg))
	} else {
		b.WriteString(ui.Help.Render("tab focus · enter submit · q quit"))
	}
	return lipgloss.NewStyle().Padding(1, 2).Render(b.String())
}

func persistToken(token string) error {
	return config.SaveCredentials(config.Credentials{Token: token})
}

func loginWithClient(client loginClient, email, password string) tea.Cmd {
	return func() tea.Msg {
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
	SetToken(token string)
}
