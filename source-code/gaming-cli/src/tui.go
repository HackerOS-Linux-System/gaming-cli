package src

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Stany TUI ─────────────────────────────────────────────────────────────────

type tuiState int

const (
	stateMenu tuiState = iota
	stateInfo
	stateLoading
	stateDone
	stateError
	stateSteamSettings
)

// ── Opcje menu ────────────────────────────────────────────────────────────────

type menuItem struct {
	label    string
	sublabel string
	action   menuAction
}

type menuAction int

const (
	actionSwitchGame menuAction = iota
	actionSwitchDesktop
	actionInfo
	actionSteamSettings
	actionGamescope
	actionQuit
)

var menuItems = []menuItem{
	{"Tryb gry", "Gamescope + Steam Big Picture", actionSwitchGame},
	{"Tryb pulpitu", "Powrot do KDE Plasma", actionSwitchDesktop},
	{"Info", "Informacje o srodowisku gaming", actionInfo},
	{"Ustawienia Steam", "Konfiguracja Steam i gamescope", actionSteamSettings},
	{"Gamescope", "Uruchom gamescope standalone", actionGamescope},
	{"Wyjdz", "Zamknij gaming TUI", actionQuit},
}

// ── SteamSettings — konfiguracja Steam ────────────────────────────────────────

// SteamSettings przechowuje stan ustawień Steam w TUI.
// Odzwierciedla pola z gamescope-manager/src/Config.
type SteamSettings struct {
	BigPicture   bool
	VR           bool
	NoCEFSandbox bool
	TCPMode      bool
	NoVerify     bool
	FullDesktop  bool
	AllowHidCrpt bool
	DisableHWA   bool
	BetaChannel  int // indeks w betaChannels
	Language     int // indeks w steamLanguages
	ExtraFlags   string
}

// betaChannels — dostepne kanaly beta Steam
var betaChannels = []string{
	"brak",
	"beta",
	"steamdeckbeta",
	"beta-client",
}

// betaChannelKeys — wartosci przekazywane do konfiguracji (.hk / CLI)
var betaChannelKeys = []string{
	"",
	"beta",
	"steamdeckbeta",
	"beta-client",
}

// steamLanguages — dostepne jezyki Steam
var steamLanguages = []string{
	"(systemowy)",
	"polish",
	"english",
	"german",
	"french",
	"spanish",
	"russian",
	"czech",
	"hungarian",
	"turkish",
}

// steamLanguageKeys — wartosci przekazywane do Steam
var steamLanguageKeys = []string{
	"",
	"polish",
	"english",
	"german",
	"french",
	"spanish",
	"russian",
	"czech",
	"hungarian",
	"turkish",
}

// settingsField opisuje jeden wiersz w panelu ustawien
type settingsField int

const (
	sfBigPicture settingsField = iota
	sfVR
	sfNoCEFSandbox
	sfTCPMode
	sfNoVerify
	sfFullDesktop
	sfAllowHidCrpt
	sfDisableHWA
	sfBetaChannel
	sfLanguage
	sfExtraFlags
	sfSave
	sfCancel
	sfDefaults
	sfFieldCount // sentinel
)

// ── Opis pol formularza ustawien ─────────────────────────────────────────────

type fieldDef struct {
	label   string
	hint    string
	kind    string // "bool" | "select" | "text" | "action"
	options []string
}

var settingsFields = []fieldDef{
	{label: "Tryb Big Picture", hint: "Uruchamia Steam w trybie Steam Deck / TV (-tenfoot)", kind: "bool"},
	{label: "Tryb VR", hint: "Uruchamia Steam VR (-vr)", kind: "bool"},
	{label: "Wylacz sandbox CEF", hint: "Wylaczy sandbox przegladarki, wieksza wydajnosc (-no-cef-sandbox)", kind: "bool"},
	{label: "Uzyj TCP zamiast socket", hint: "Pomaga gdy masz problemy z polaczeniem (-nosharedmemory)", kind: "bool"},
	{label: "Pomij weryfikacje plikow", hint: "Nie sprawdzaj integralnosci plikow gier przy starcie (-noverifyfiles)", kind: "bool"},
	{label: "Pelna rozdzielczosc pulpitu", hint: "Rozciagnij Steam na caly ekran w trybie pulpitu (-fulldesktopres)", kind: "bool"},
	{label: "Allow HID Crypto", hint: "Umozliwia szyfrowane urzadzenia wejsciowe (-allow-hidcrypto)", kind: "bool"},
	{label: "Wylacz akceleracje sprzetowa UI", hint: "Wylacz GPU rendering w interfejsie Steam (-disable-gpu)", kind: "bool"},
	{label: "Kanal beta", hint: "Przejdz na kanal beta Steam", kind: "select", options: betaChannels},
	{label: "Jezyk", hint: "Jezyk interfejsu Steam", kind: "select", options: steamLanguages},
	{label: "Dodatkowe flagi Steam", hint: "Wpisz dowolne dodatkowe flagi Steam (np. -noverifyfiles)", kind: "text"},
	{label: "Zapisz", hint: "Zapisz konfiguracje do /etc/hackeros/gamescope-manager.hk", kind: "action"},
	{label: "Anuluj", hint: "Wyjdz bez zapisywania", kind: "action"},
	{label: "Domyslne", hint: "Przywroc domyslne ustawienia Steam", kind: "action"},
}

// ── Model ─────────────────────────────────────────────────────────────────────

type Model struct {
	cursor       int
	state        tuiState
	currentMode  string
	info         EnvInfo
	spinner      spinner.Model
	loadingMsg   string
	doneMsg      string
	errMsg       string
	width        int
	height       int
	steamCursor  settingsField
	steamSettings SteamSettings
	steamDirty   bool   // czy zmieniono ustawienia
	textEditing  bool   // czy edytujemy pole tekstowe
	textBuf      string // bufor edycji tekstu
}

// ── Wiadomosci ────────────────────────────────────────────────────────────────

type switchDoneMsg struct{ err error }
type infoReadyMsg struct{ info EnvInfo }
type steamSavedMsg struct{ err error }

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
		steamSettings: defaultSteamSettings(),
	}
}

func defaultSteamSettings() SteamSettings {
	return SteamSettings{
		BigPicture:   true,
		VR:           false,
		NoCEFSandbox: true,
		TCPMode:      false,
		NoVerify:     false,
		FullDesktop:  false,
		AllowHidCrpt: false,
		DisableHWA:   false,
		BetaChannel:  0,
		Language:     0,
		ExtraFlags:   "",
	}
}

func (m Model) Init() tea.Cmd { return nil }

// ── Update ────────────────────────────────────────────────────────────────────

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

		case steamSavedMsg:
			if msg.err != nil {
				m.state = stateError
				m.errMsg = "Blad zapisu ustawien Steam: " + msg.err.Error()
			} else {
				m.state = stateDone
				m.doneMsg = "Ustawienia Steam zapisane."
			}
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

								case stateSteamSettings:
									return m.handleSteamKey(msg)
	}

	return m, nil
}

// handleSteamKey obsługuje klawiaturę w panelu ustawień Steam.
func (m Model) handleSteamKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Jeśli edytujemy pole tekstowe
	if m.textEditing {
		switch msg.String() {
			case "enter":
				m.steamSettings.ExtraFlags = m.textBuf
				m.textEditing = false
				m.steamDirty = true
			case "esc":
				m.textEditing = false
			case "backspace", "ctrl+h":
				if len(m.textBuf) > 0 {
					m.textBuf = m.textBuf[:len(m.textBuf)-1]
				}
			default:
				if len(msg.Runes) > 0 {
					m.textBuf += string(msg.Runes)
				}
		}
		return m, nil
	}

	f := m.steamCursor
	fd := settingsFields[f]

	switch msg.String() {
		case "up", "k":
			if m.steamCursor > 0 {
				m.steamCursor--
			} else {
				m.steamCursor = sfFieldCount - 1
			}
		case "down", "j", "tab":
			if m.steamCursor < sfFieldCount-1 {
				m.steamCursor++
			} else {
				m.steamCursor = 0
			}
		case "left", "h":
			if fd.kind == "select" {
				m.adjustSelect(f, -1)
				m.steamDirty = true
			}
		case "right", "l":
			if fd.kind == "select" {
				m.adjustSelect(f, +1)
				m.steamDirty = true
			}
		case " ", "enter":
			switch fd.kind {
				case "bool":
					m.toggleBool(f)
					m.steamDirty = true
				case "select":
					m.adjustSelect(f, +1)
					m.steamDirty = true
				case "text":
					m.textEditing = true
					m.textBuf = m.steamSettings.ExtraFlags
				case "action":
					return m.execSteamAction(f)
			}
				case "esc", "q":
					m.state = stateMenu
	}

	return m, nil
}

func (m *Model) toggleBool(f settingsField) {
	switch f {
		case sfBigPicture:
			m.steamSettings.BigPicture = !m.steamSettings.BigPicture
		case sfVR:
			m.steamSettings.VR = !m.steamSettings.VR
		case sfNoCEFSandbox:
			m.steamSettings.NoCEFSandbox = !m.steamSettings.NoCEFSandbox
		case sfTCPMode:
			m.steamSettings.TCPMode = !m.steamSettings.TCPMode
		case sfNoVerify:
			m.steamSettings.NoVerify = !m.steamSettings.NoVerify
		case sfFullDesktop:
			m.steamSettings.FullDesktop = !m.steamSettings.FullDesktop
		case sfAllowHidCrpt:
			m.steamSettings.AllowHidCrpt = !m.steamSettings.AllowHidCrpt
		case sfDisableHWA:
			m.steamSettings.DisableHWA = !m.steamSettings.DisableHWA
	}
}

func (m *Model) adjustSelect(f settingsField, delta int) {
	switch f {
		case sfBetaChannel:
			n := (m.steamSettings.BetaChannel + delta + len(betaChannels)) % len(betaChannels)
			m.steamSettings.BetaChannel = n
		case sfLanguage:
			n := (m.steamSettings.Language + delta + len(steamLanguages)) % len(steamLanguages)
			m.steamSettings.Language = n
	}
}

func (m Model) execSteamAction(f settingsField) (tea.Model, tea.Cmd) {
	switch f {
		case sfSave:
			ss := m.steamSettings
			return m, func() tea.Msg {
				err := saveSteamSettings(ss)
				return steamSavedMsg{err: err}
			}
		case sfCancel:
			m.state = stateMenu
			return m, nil
		case sfDefaults:
			m.steamSettings = defaultSteamSettings()
			m.steamDirty = true
			return m, nil
	}
	return m, nil
}

// saveSteamSettings zapisuje ustawienia Steam przez gamescope-manager config set.
// Wywołuje sudo gamescope-manager config set dla każdej zmienionej wartości.
func saveSteamSettings(ss SteamSettings) error {
	gm := "/usr/bin/gamescope-manager"
	if _, err := os.Stat(gm); err != nil {
		if p, err2 := exec.LookPath("gamescope-manager"); err2 == nil {
			gm = p
		} else {
			return fmt.Errorf("gamescope-manager nie znaleziony — nie mozna zapisac")
		}
	}

	type kv struct{ k, v string }
	settings := []kv{
		{"big_picture", boolStr(ss.BigPicture)},
		{"vr", boolStr(ss.VR)},
		{"no_cef_sandbox", boolStr(ss.NoCEFSandbox)},
		{"tcp_mode", boolStr(ss.TCPMode)},
		{"no_verify", boolStr(ss.NoVerify)},
		{"full_desktop_res", boolStr(ss.FullDesktop)},
		{"allow_hidcrypto", boolStr(ss.AllowHidCrpt)},
		{"disable_hwa", boolStr(ss.DisableHWA)},
		{"beta_channel", betaChannelKeys[ss.BetaChannel]},
		{"language", steamLanguageKeys[ss.Language]},
		{"steam_extra_flags", ss.ExtraFlags},
	}

	for _, s := range settings {
		cmd := exec.Command("sudo", gm, "config", "set", s.k, s.v)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("blad ustawiania %s: %w", s.k, err)
		}
	}
	return nil
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func (m Model) execAction(action menuAction) (tea.Model, tea.Cmd) {
	switch action {
		case actionQuit:
			return m, tea.Quit

		case actionSwitchGame:
			m.state = stateLoading
			m.loadingMsg = "Przelaczam na tryb gry..."
			m.doneMsg = "Tryb gry aktywny!"
			return m, tea.Batch(
				m.spinner.Tick,
		       func() tea.Msg {
			       return switchDoneMsg{err: SwitchToGame()}
		       },
			)

		case actionSwitchDesktop:
			m.state = stateLoading
			m.loadingMsg = "Przelaczam na KDE Plasma..."
			m.doneMsg = "Tryb pulpitu aktywny!"
			return m, tea.Batch(
				m.spinner.Tick,
		       func() tea.Msg {
			       return switchDoneMsg{err: SwitchToDesktop()}
		       },
			)

		case actionInfo:
			m.state = stateLoading
			m.loadingMsg = "Zbieram informacje..."
			return m, tea.Batch(
				m.spinner.Tick,
		       func() tea.Msg {
			       return infoReadyMsg{info: GatherInfo()}
		       },
			)

		case actionSteamSettings:
			// Załaduj aktualne ustawienia z pliku konfiguracyjnego
			m.steamSettings = loadSteamSettingsFromFile()
			m.steamDirty = false
			m.steamCursor = 0
			m.state = stateSteamSettings
			return m, nil

		case actionGamescope:
			return m, tea.Sequence(tea.Quit, func() tea.Msg { return nil })
	}
	return m, nil
}

// loadSteamSettingsFromFile odczytuje ustawienia Steam z pliku .hk przez gamescope-manager.
func loadSteamSettingsFromFile() SteamSettings {
	ss := defaultSteamSettings()

	gm := "/usr/bin/gamescope-manager"
	if _, err := os.Stat(gm); err != nil {
		if p, err2 := exec.LookPath("gamescope-manager"); err2 == nil {
			gm = p
		} else {
			return ss
		}
	}

	// Wczytaj przez `sudo gamescope-manager config` — parsujemy output
	out, err := exec.Command("sudo", gm, "config").Output()
	if err != nil {
		return ss
	}

	// Prosta parsownia: "klucz                = wartość"
	vals := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		vals[k] = v
	}

	parseBoolVal := func(k string, def bool) bool {
		v, ok := vals[k]
		if !ok {
			return def
		}
		return v == "true"
	}

	ss.BigPicture = parseBoolVal("big_picture", ss.BigPicture)
	ss.VR = parseBoolVal("vr", ss.VR)
	ss.NoCEFSandbox = parseBoolVal("no_cef_sandbox", ss.NoCEFSandbox)
	ss.TCPMode = parseBoolVal("tcp_mode", ss.TCPMode)
	ss.NoVerify = parseBoolVal("no_verify", ss.NoVerify)
	ss.FullDesktop = parseBoolVal("full_desktop_res", ss.FullDesktop)
	ss.AllowHidCrpt = parseBoolVal("allow_hidcrypto", ss.AllowHidCrpt)
	ss.DisableHWA = parseBoolVal("disable_hwa", ss.DisableHWA)

	if bc, ok := vals["beta_channel"]; ok {
		for i, k := range betaChannelKeys {
			if k == bc {
				ss.BetaChannel = i
				break
			}
		}
	}
	if lang, ok := vals["language"]; ok {
		for i, k := range steamLanguageKeys {
			if k == lang {
				ss.Language = i
				break
			}
		}
	}
	if ef, ok := vals["steam_extra_flags"]; ok && ef != "(brak)" {
		ss.ExtraFlags = ef
	}

	return ss
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
		case stateSteamSettings:
			return m.viewSteamSettings()
	}
	return ""
}

// ── Widok Menu ────────────────────────────────────────────────────────────────

func (m Model) viewMenu() string {
	var sb strings.Builder

	sb.WriteString(renderBanner())
	sb.WriteString("\n")

	modeBar := "  Aktywny tryb: " + ModeBadge(m.currentMode)
	sb.WriteString(modeBar)
	sb.WriteString("\n")
	sb.WriteString("  " + Divider(m.width-4))
	sb.WriteString("\n\n")

	for i, item := range menuItems {
		isSelected := i == m.cursor
		var label, sub string
		if isSelected {
			cursor := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("  >")
			label = lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(item.label)
			sub = lipgloss.NewStyle().Foreground(ColorSecondary).Render("    " + item.sublabel)
			row := fmt.Sprintf("%s  %s", cursor, label)
			sb.WriteString(lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1B4B")).
			Width(m.width - 4).
			Render(row))
			sb.WriteString("\n")
			sb.WriteString(lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1B4B")).
			Width(m.width - 4).
			Render(sub))
		} else {
			label = StyleCmdName.Render(item.label)
			sub = StyleCmdDesc.Render("    " + item.sublabel)
			sb.WriteString(fmt.Sprintf("     %s", label))
			sb.WriteString("\n")
			sb.WriteString(sub)
		}
		sb.WriteString("\n\n")
	}

	sb.WriteString("  " + Divider(m.width-4) + "\n")
	sb.WriteString("  " + StyleCmdDesc.Render("strzalki / k j  nawigacja   enter  wybierz   q  wyjdz"))
	sb.WriteString("\n")
	return sb.String()
}

// ── Widok Info ────────────────────────────────────────────────────────────────

func (m Model) viewInfo() string {
	var sb strings.Builder
	info := m.info

	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")

	title := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).
	Render("  Informacje o srodowisku")
	sb.WriteString(title + "\n")
	sb.WriteString("  " + Divider(m.width-4) + "\n\n")

	rows := []struct{ label, value string }{
		{"Dystrybucja", info.PrettyName},
		{"Debian", info.DebianVer},
		{"Jadro", info.Kernel},
		{"GPU", info.GPU},
		{"Gamescope", info.Gamescope},
		{"Steam", info.Steam},
		{"Tryb", info.CurrentMode},
	}

	for _, r := range rows {
		label := StyleLabel.Render(r.label)
		sep := StyleDivider.Render(" : ")
		var val string
		switch {
			case strings.Contains(r.value, "NIE ZAINSTALOWANY"):
				val = StyleValueBad.Render(r.value)
			case strings.Contains(r.value, "game-mode"):
				val = StyleValueGood.Render(r.value)
			case strings.Contains(r.value, "desktop-mode"):
				val = StyleValue.Render(r.value)
			case strings.Contains(r.value, "zainstalowany"):
				val = StyleValueGood.Render(r.value)
			default:
				val = StyleValue.Render(r.value)
		}
		sb.WriteString("  " + label + sep + val + "\n")
	}

	sb.WriteString("\n  " + Divider(m.width-4) + "\n")
	sb.WriteString("  " + StyleCmdDesc.Render("esc / backspace  powrot do menu"))
	sb.WriteString("\n")
	return sb.String()
}

// ── Widok Steam Settings ──────────────────────────────────────────────────────

func (m Model) viewSteamSettings() string {
	var sb strings.Builder

	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")

	title := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).
	Render("  Ustawienia Steam")
	if m.steamDirty {
		title += lipgloss.NewStyle().Foreground(ColorYellow).Render("  *")
	}
	sb.WriteString(title + "\n")
	sb.WriteString("  " + Divider(m.width-4) + "\n\n")

	ss := m.steamSettings

	// Style
	lblW := lipgloss.NewStyle().Foreground(ColorGray).Width(28)
	selStyle := lipgloss.NewStyle().
	Background(lipgloss.Color("#1E1B4B")).
	Foreground(ColorWhite).Bold(true)
	cursorStr := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(" > ")
	noSel := "   "

	renderRow := func(f settingsField, label, value, hint string) {
		isSelected := m.steamCursor == f
		lbl := lblW.Render(label)
		var rowStr string
		if isSelected {
			val := selStyle.Render(" " + value + " ")
			rowStr = fmt.Sprintf("  %s%s : %s", cursorStr, lbl, val)
		} else {
			val := lipgloss.NewStyle().Foreground(ColorWhite).Render(value)
			rowStr = fmt.Sprintf("  %s%s : %s", noSel, lbl, val)
		}
		sb.WriteString(rowStr + "\n")
		if isSelected && hint != "" {
			hintStr := lipgloss.NewStyle().Foreground(ColorDim).Italic(true).
			Render("       " + hint)
			sb.WriteString(hintStr + "\n")
		}
	}

	renderBoolRow := func(f settingsField, val bool) {
		fd := settingsFields[f]
		v := checkBox(val)
		renderRow(f, fd.label, v, fd.hint)
	}

	renderSelectRow := func(f settingsField, idx int, opts []string) {
		fd := settingsFields[f]
		v := "< " + opts[idx] + " >"
		renderRow(f, fd.label, v, fd.hint)
	}

	renderTextRow := func(f settingsField) {
		fd := settingsFields[f]
		v := ss.ExtraFlags
		if m.textEditing && m.steamCursor == f {
			v = m.textBuf + "_"
		}
		if v == "" {
			v = "(brak)"
		}
		renderRow(f, fd.label, v, fd.hint)
	}

	renderActionRow := func(f settingsField) {
		fd := settingsFields[f]
		isSelected := m.steamCursor == f
		lbl := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Width(12).Render(fd.label)
		var rowStr string
		if isSelected {
			lbl = lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1B4B")).
			Foreground(ColorWhite).Bold(true).
			Padding(0, 1).Render(fd.label)
			rowStr = fmt.Sprintf("  %s[ %s ]", cursorStr, lbl)
		} else {
			rowStr = fmt.Sprintf("  %s  %s", noSel, lbl)
		}
		sb.WriteString(rowStr)
	}

	renderBoolRow(sfBigPicture, ss.BigPicture)
	renderBoolRow(sfVR, ss.VR)
	renderBoolRow(sfNoCEFSandbox, ss.NoCEFSandbox)
	renderBoolRow(sfTCPMode, ss.TCPMode)
	renderBoolRow(sfNoVerify, ss.NoVerify)
	renderBoolRow(sfFullDesktop, ss.FullDesktop)
	renderBoolRow(sfAllowHidCrpt, ss.AllowHidCrpt)
	renderBoolRow(sfDisableHWA, ss.DisableHWA)

	sb.WriteString("\n")
	renderSelectRow(sfBetaChannel, ss.BetaChannel, betaChannels)
	renderSelectRow(sfLanguage, ss.Language, steamLanguages)

	sb.WriteString("\n")
	renderTextRow(sfExtraFlags)

	sb.WriteString("\n")
	sb.WriteString("  " + Divider(m.width-4) + "\n  ")
	renderActionRow(sfSave)
	sb.WriteString("  ")
	renderActionRow(sfCancel)
	sb.WriteString("  ")
	renderActionRow(sfDefaults)
	sb.WriteString("\n\n")

	if m.textEditing {
		sb.WriteString("  " + StyleCmdDesc.Render("Edytujesz tekst — enter zatwierdz   esc anuluj"))
	} else {
		sb.WriteString("  " + StyleCmdDesc.Render("k/j  nawigacja   spacja/enter  toggle/wybor   strzalki  lista   esc  powrot"))
	}
	sb.WriteString("\n")

	return sb.String()
}

// checkBox zwraca tekstowy przełącznik [ X ] lub [   ]
func checkBox(b bool) string {
	if b {
		return "[X] ON "
	}
	return "[ ] OFF"
}

// ── Widok Loading ─────────────────────────────────────────────────────────────

func (m Model) viewLoading() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")
	spin := m.spinner.View()
	msg := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(m.loadingMsg)
	sb.WriteString(fmt.Sprintf("  %s  %s\n", spin, msg))
	return sb.String()
}

// ── Widok Done ────────────────────────────────────────────────────────────────

func (m Model) viewDone() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")
	icon := StyleSuccess.Render("  OK")
	msg := StyleSuccess.Render("  " + m.doneMsg)
	sb.WriteString(icon + msg + "\n\n")
	sb.WriteString("  " + StyleCmdDesc.Render("enter / esc  powrot do menu") + "\n")
	return sb.String()
}

// ── Widok Error ───────────────────────────────────────────────────────────────

func (m Model) viewError() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")
	header := StyleError.Render("  BLAD:")
	sb.WriteString(header + "\n")
	sb.WriteString("  " + StyleValueBad.Render(m.errMsg) + "\n\n")
	sb.WriteString("  " + StyleCmdDesc.Render("enter / esc  powrot do menu") + "\n")
	return sb.String()
}

// ── Banner ────────────────────────────────────────────────────────────────────

func renderBanner() string {
	line1 := StyleBannerTitle.Render("  HackerOS Gaming Edition")
	line2 := StyleBannerSub.Render("  gaming-cli") +
	StyleDivider.Render(" - ") +
	StyleBannerTag.Render("v0.0.1  ·  Debian Testing (Forky)  ·  PC / Laptop")
	return line1 + "\n" + line2
}

// ── RunTUI ────────────────────────────────────────────────────────────────────

func RunTUI() {
	m := InitialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI blad: %v\n", err)
		os.Exit(1)
	}
}
