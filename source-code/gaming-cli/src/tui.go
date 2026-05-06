package src

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Stany TUI ────────────────────────────────────────────────────────────────

type tuiState int

const (
	stateMenu    tuiState = iota
	stateInfo
	stateLoading
	stateDone
	stateError
)

// ── Opcje menu ────────────────────────────────────────────────────────────────

type menuItem struct {
	label    string
	sublabel string
	icon     string
	action   menuAction
}

type menuAction int

const (
	actionSwitchGame menuAction = iota
	actionSwitchDesktop
	actionInfo
	actionGamescope
	actionQuit
)

var menuItems = []menuItem{
	{icon: "🎮", label: "Tryb gry",     sublabel: "Gamescope + Steam Big Picture",  action: actionSwitchGame},
	{icon: "🖥 ", label: "Tryb pulpitu", sublabel: "Powrót do KDE Plasma",           action: actionSwitchDesktop},
	{icon: "📊", label: "Info",          sublabel: "Informacje o środowisku gaming", action: actionInfo},
	{icon: "🔧", label: "Gamescope",     sublabel: "Uruchom gamescope standalone",   action: actionGamescope},
	{icon: "✖ ", label: "Wyjdź",         sublabel: "Zamknij gaming TUI",             action: actionQuit},
}

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	cursor      int
	state       tuiState
	currentMode string
	info        EnvInfo
	spinner     spinner.Model
	loadingMsg  string
	doneMsg     string
	errMsg      string
	width       int
	height      int
}

// ── Wiadomości ────────────────────────────────────────────────────────────────

type switchDoneMsg struct{ err error }
type infoReadyMsg struct{ info EnvInfo }

// ── Init ──────────────────────────────────────────────────────────────────────

func InitialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)

	mode, _ := CurrentMode()
	return Model{
		cursor:      0,
		state:       stateMenu,
		currentMode: mode,
		spinner:     s,
		width:       80,
		height:      24,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

// ── Update ─────────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

		case tea.WindowSizeMsg:
			m.width = msg.Width
			m.height = msg.Height
			return m, nil

		case spinner.TickMsg:
			if m.state == stateLoading {
				var cmd tea.Cmd
				m.spinner, cmd = m.spinner.Update(msg)
				return m, cmd
			}
			return m, nil

		case switchDoneMsg:
			if msg.err != nil {
				m.state = stateError
				m.errMsg = msg.err.Error()
			} else {
				mode, _ := CurrentMode()
				m.currentMode = mode
				m.state = stateDone
			}
			return m, nil

		case infoReadyMsg:
			m.info = msg.info
			m.state = stateInfo
			return m, nil

		case tea.KeyMsg:
			return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {

		case stateMenu:
			switch msg.String() {
				case "up", "k":
					if m.cursor > 0 {
						m.cursor--
					} else {
						m.cursor = len(menuItems) - 1
					}
				case "down", "j":
					if m.cursor < len(menuItems)-1 {
						m.cursor++
					} else {
						m.cursor = 0
					}
				case "enter", " ":
					return m.execAction(menuItems[m.cursor].action)
				case "q", "ctrl+c", "esc":
					return m, tea.Quit
			}

				case stateInfo:
					switch msg.String() {
						case "q", "esc", "backspace":
							m.state = stateMenu
					}

						case stateDone, stateError:
							switch msg.String() {
								case "q", "esc", "enter", "backspace":
									m.state = stateMenu
							}
	}

	return m, nil
}

func (m Model) execAction(action menuAction) (tea.Model, tea.Cmd) {
	switch action {
		case actionQuit:
			return m, tea.Quit

		case actionSwitchGame:
			m.state = stateLoading
			m.loadingMsg = "Przełączam na tryb gry…"
			m.doneMsg = "Tryb gry aktywny! 🎮"
			return m, tea.Batch(
				m.spinner.Tick,
		       func() tea.Msg {
			       err := SwitchToGame()
			       return switchDoneMsg{err: err}
		       },
			)

		case actionSwitchDesktop:
			m.state = stateLoading
			m.loadingMsg = "Przełączam na KDE Plasma…"
			m.doneMsg = "Tryb pulpitu aktywny! 🖥"
			return m, tea.Batch(
				m.spinner.Tick,
		       func() tea.Msg {
			       err := SwitchToDesktop()
			       return switchDoneMsg{err: err}
		       },
			)

		case actionInfo:
			m.state = stateLoading
			m.loadingMsg = "Zbieram informacje…"
			return m, tea.Batch(
				m.spinner.Tick,
		       func() tea.Msg {
			       info := GatherInfo()
			       return infoReadyMsg{info: info}
		       },
			)

		case actionGamescope:
			// Wyjdź z TUI i uruchom gamescope interaktywnie
			return m, tea.Sequence(
				tea.Quit,
			  func() tea.Msg { return nil },
			)
	}
	return m, nil
}

// ── View ───────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	switch m.state {
		case stateMenu:
			return m.viewMenu()
		case stateInfo:
			return m.viewInfo()
		case stateLoading:
			return m.viewLoading()
		case stateDone:
			return m.viewDone()
		case stateError:
			return m.viewError()
	}
	return ""
}

// ── Widok Menu ─────────────────────────────────────────────────────────────────

func (m Model) viewMenu() string {
	var sb strings.Builder

	// --- Banner ---
	banner := renderBanner()
	sb.WriteString(banner)
	sb.WriteString("\n")

	// --- Tryb badge ---
	modeBar := "  Aktywny tryb: " + ModeBadge(m.currentMode)
	sb.WriteString(modeBar)
	sb.WriteString("\n")
	sb.WriteString("  " + Divider(m.width-4))
	sb.WriteString("\n\n")

	// --- Menu items ---
	for i, item := range menuItems {
		isSelected := i == m.cursor

		icon := StyleCmdArg.Render(item.icon)

		var label, sub string
		if isSelected {
			cursor := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("❯")
			label = lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(item.label)
			sub   = lipgloss.NewStyle().Foreground(ColorSecondary).Render("  " + item.sublabel)
			row   := fmt.Sprintf("  %s %s %s", cursor, icon, label)
			sb.WriteString(lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1B4B")).
			Width(m.width - 4).
			PaddingLeft(2).
			Render(row))
			sb.WriteString("\n")
			sb.WriteString(lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1B4B")).
			Width(m.width - 4).
			PaddingLeft(2).
			Render(sub))
		} else {
			cursor := "  "
			label = StyleCmdName.Render(item.label)
			sub   = StyleCmdDesc.Render("  " + item.sublabel)
			row   := fmt.Sprintf("  %s %s %s", cursor, icon, label)
			sb.WriteString("  " + row)
			sb.WriteString("\n")
			sb.WriteString("  " + sub)
		}
		sb.WriteString("\n\n")
	}

	// --- Footer ---
	sb.WriteString("  " + Divider(m.width-4) + "\n")
	sb.WriteString("  " + StyleCmdDesc.Render("↑/↓ lub k/j — nawigacja   enter — wybierz   q — wyjdź"))
	sb.WriteString("\n")

	return sb.String()
}

// ── Widok Info ──────────────────────────────────────────────────────────────────

func (m Model) viewInfo() string {
	var sb strings.Builder
	info := m.info

	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")

	title := lipgloss.NewStyle().
	Foreground(ColorAccent).Bold(true).
	Render("  📊 Informacje o środowisku")
	sb.WriteString(title + "\n")
	sb.WriteString("  " + Divider(m.width-4) + "\n\n")

	rows := []struct{ label, value string }{
		{"Dystrybucja", info.PrettyName},
		{"Debian", info.DebianVer},
		{"Jądro", info.Kernel},
		{"GPU", info.GPU},
		{"Gamescope", info.Gamescope},
		{"Steam", info.Steam},
		{"Tryb", info.CurrentMode},
	}

	for _, r := range rows {
		label := StyleLabel.Render(r.label)
		sep   := StyleDivider.Render(" : ")

		var val string
		switch {
			case strings.Contains(r.value, "NIE ZAINSTALOWANY"):
				val = StyleValueBad.Render(r.value)
			case strings.Contains(r.value, "game-mode"):
				val = StyleValueGood.Render("🎮 " + r.value)
			case strings.Contains(r.value, "desktop-mode"):
				val = StyleValue.Render("🖥  " + r.value)
			case strings.Contains(r.value, "zainstalowany"):
				val = StyleValueGood.Render(r.value)
			default:
				val = StyleValue.Render(r.value)
		}

		sb.WriteString("  " + label + sep + val + "\n")
	}

	sb.WriteString("\n  " + Divider(m.width-4) + "\n")
	sb.WriteString("  " + StyleCmdDesc.Render("esc / backspace — powrót do menu"))
	sb.WriteString("\n")
	return sb.String()
}

// ── Widok Loading ──────────────────────────────────────────────────────────────

func (m Model) viewLoading() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")

	spin := m.spinner.View()
	msg  := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(m.loadingMsg)
	sb.WriteString(fmt.Sprintf("  %s  %s\n", spin, msg))
	return sb.String()
}

// ── Widok Done ─────────────────────────────────────────────────────────────────

func (m Model) viewDone() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")

	icon := StyleSuccess.Render("✔")
	msg  := StyleSuccess.Render(m.doneMsg)
	sb.WriteString(fmt.Sprintf("  %s  %s\n\n", icon, msg))
	sb.WriteString("  " + StyleCmdDesc.Render("enter / esc — powrót do menu") + "\n")
	return sb.String()
}

// ── Widok Error ────────────────────────────────────────────────────────────────

func (m Model) viewError() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")

	icon := StyleError.Render("✖")
	header := StyleError.Render("Wystąpił błąd:")
	sb.WriteString(fmt.Sprintf("  %s  %s\n", icon, header))
	sb.WriteString("  " + StyleValueBad.Render(m.errMsg) + "\n\n")
	sb.WriteString("  " + StyleCmdDesc.Render("enter / esc — powrót do menu") + "\n")
	return sb.String()
}

// ── Banner ─────────────────────────────────────────────────────────────────────

func renderBanner() string {
	line1 := StyleBannerTitle.Render("  HackerOS Gaming Edition")
	line2 := StyleBannerSub.Render("  gaming-cli") +
	StyleDivider.Render(" ─ ") +
	StyleBannerTag.Render("v0.0.1  ·  Debian Testing (Forky)  ·  PC / Laptop")
	return line1 + "\n" + line2
}

// ── Uruchomienie TUI ───────────────────────────────────────────────────────────

func RunTUI() {
	m := InitialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI błąd: %v\n", err)
		os.Exit(1)
	}
}
