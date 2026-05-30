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
	stateHealth
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
	actionHealth
	actionGamescope
	actionQuit
)

var menuItems = []menuItem{
	{"Tryb gry", "Gamescope + Steam Big Picture", actionSwitchGame},
	{"Tryb pulpitu", "Powrot do KDE Plasma", actionSwitchDesktop},
	{"Info", "Informacje o srodowisku gaming", actionInfo},
	{"Ustawienia", "Konfiguracja gamescope i Steam", actionSteamSettings},
	{"Health check", "Sprawdz gotowos systemu", actionHealth},
	{"Gamescope", "Uruchom gamescope standalone", actionGamescope},
	{"Wyjdz", "Zamknij gaming TUI", actionQuit},
}

// ── SteamSettings — pełna konfiguracja sesji ──────────────────────────────────

type SteamSettings struct {
	// gamescope / rozdzielczosc
	AutoResolution bool   // true = width/height=0 (auto)
	Width          int
	Height         int
	RefreshRate    int    // 0 = auto
	ScalingMode    int    // indeks w scalingModes
	ScalingFilter  int    // indeks w scalingFilters
	FSRSharpness   int    // 0-20
	VSync          bool
	HDR            bool
	MangoApp       bool
	ForceComposite bool

	// steam
	BigPicture   bool
	VR           bool
	NoCEFSandbox bool
	TCPMode      bool
	NoVerify     bool
	FullDesktop  bool
	AllowHidCrpt bool
	DisableHWA   bool
	BetaChannel  int
	Language     int
	ExtraFlags   string
}

// ── Opcje select ──────────────────────────────────────────────────────────────

var scalingModes = []string{"auto", "integer", "fit", "fill", "stretch"}
var scalingModeDesc = []string{
	"gamescope dobiera tryb automatycznie",
	"piksel-perfect (do retro/pixel-art)",
	"zachowaj proporcje (letterbox)",
	"wypelnij ekran (moze ucinac)",
	"rozciagnij (znieksztalca)",
}

var scalingFilters = []string{"linear", "pixel", "fsr", "nis", "nearest"}
var scalingFilterDesc = []string{
	"lekkie rozmycie, ogolny cel",
	"ostry, do gier pixel-art",
	"AMD FidelityFX Super Resolution",
	"NVIDIA/AMD/Intel Image Scaling",
	"brak filtrowania, najostrzejszy",
}

var betaChannels = []string{"brak", "beta", "steamdeckbeta", "beta-client"}
var betaChannelKeys = []string{"", "beta", "steamdeckbeta", "beta-client"}

var steamLanguages = []string{
	"(systemowy)", "polish", "english", "german", "french",
	"spanish", "russian", "czech", "hungarian", "turkish",
}
var steamLanguageKeys = []string{
	"", "polish", "english", "german", "french",
	"spanish", "russian", "czech", "hungarian", "turkish",
}

// ── Pola formularza ustawień ──────────────────────────────────────────────────

type settingsField int

const (
	// --- Sekcja rozdzielczosci ---
	sfSectionResolution settingsField = iota
	sfAutoResolution
	sfWidth
	sfHeight
	sfRefreshRate
	// --- Sekcja skalowania ---
	sfSectionScaling
	sfScalingMode
	sfScalingFilter
	sfFSRSharpness
	// --- Sekcja gamescope ---
	sfSectionGamescope
	sfVSync
	sfHDR
	sfMangoApp
	sfForceComposite
	// --- Sekcja steam ---
	sfSectionSteam
	sfBigPicture
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
	// --- Akcje ---
	sfSave
	sfCancel
	sfDefaults
	sfFieldCount
)

// Które pola są edytowalne (nie są nagłówkami sekcji)
func isEditable(f settingsField) bool {
	switch f {
		case sfSectionResolution, sfSectionScaling, sfSectionGamescope, sfSectionSteam:
			return false
	}
	return true
}

// ── Model ─────────────────────────────────────────────────────────────────────

type healthEntry struct {
	name  string
	ok    bool
	fatal bool
	msg   string
}

type Model struct {
	cursor        int
	state         tuiState
	currentMode   string
	info          EnvInfo
	spinner       spinner.Model
	loadingMsg    string
	doneMsg       string
	errMsg        string
	width         int
	height        int
	steamCursor   settingsField
	steamSettings SteamSettings
	steamDirty    bool
	textEditing   bool
	textBuf       string
	textField     settingsField
	healthResults []healthEntry
	healthOK      bool
}

// ── Wiadomości ────────────────────────────────────────────────────────────────

type infoReadyMsg struct {
	info           EnvInfo
	_steamSettings *SteamSettings
}
type switchDoneMsg struct{ err error }
type steamSavedMsg struct{ err error }
type healthDoneMsg struct {
	results []healthEntry
	canStart bool
}

// ── Init ──────────────────────────────────────────────────────────────────────

func InitialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ColorPrimary)
	mode, _ := CurrentMode()
	return Model{
		cursor:        0,
		state:         stateMenu,
		currentMode:   mode,
		spinner:       s,
		width:         80,
		height:        24,
		steamSettings: defaultSteamSettings(),
	}
}

func defaultSteamSettings() SteamSettings {
	return SteamSettings{
		AutoResolution: true,
		Width:          1920,
		Height:         1080,
		RefreshRate:    0,
		ScalingMode:    0, // auto
		ScalingFilter:  0, // linear
		FSRSharpness:   5,
		VSync:          true,
		HDR:            false,
		MangoApp:       true,
		ForceComposite: false,
		BigPicture:     true,
		VR:             false,
		NoCEFSandbox:   true,
		TCPMode:        false,
		NoVerify:       false,
		FullDesktop:    false,
		AllowHidCrpt:   false,
		DisableHWA:     false,
		BetaChannel:    0,
		Language:       0,
		ExtraFlags:     "",
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
			if msg._steamSettings != nil {
				m.steamSettings = *msg._steamSettings
				m.steamDirty = false
				m.steamCursor = sfAutoResolution
				m.state = stateSteamSettings
			} else {
				m.info = msg.info
				m.state = stateInfo
			}
			return m, nil

		case steamSavedMsg:
			if msg.err != nil {
				m.state = stateError
				m.errMsg = "Blad zapisu ustawien: " + msg.err.Error()
			} else {
				m.state = stateDone
				m.doneMsg = "Ustawienia zapisane."
			}
			return m, nil

		case healthDoneMsg:
			m.healthResults = msg.results
			m.healthOK = msg.canStart
			m.state = stateHealth
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

				case stateInfo, stateHealth:
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

// ── Obsługa klawiatury w ustawieniach ─────────────────────────────────────────

func (m Model) handleSteamKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	switch msg.String() {
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j", "tab":
			m.moveCursor(+1)
		case "left", "h":
			m.adjustField(-1)
			m.steamDirty = true
		case "right", "l":
			m.adjustField(+1)
			m.steamDirty = true
		case " ", "enter":
			return m.activateField()
		case "esc", "q":
			m.state = stateMenu
	}
	return m, nil
}

// moveCursor przesuwa kursor, pomijając nagłówki sekcji.
func (m *Model) moveCursor(delta int) {
	f := int(m.steamCursor)
	for {
		f += delta
		if f < 0 {
			f = int(sfFieldCount) - 1
		}
		if f >= int(sfFieldCount) {
			f = 0
		}
		if isEditable(settingsField(f)) {
			break
		}
	}
	m.steamCursor = settingsField(f)
}

func (m *Model) adjustField(delta int) {
	ss := &m.steamSettings
	switch m.steamCursor {
		case sfScalingMode:
			ss.ScalingMode = wrap(ss.ScalingMode+delta, len(scalingModes))
		case sfScalingFilter:
			ss.ScalingFilter = wrap(ss.ScalingFilter+delta, len(scalingFilters))
		case sfBetaChannel:
			ss.BetaChannel = wrap(ss.BetaChannel+delta, len(betaChannels))
		case sfLanguage:
			ss.Language = wrap(ss.Language+delta, len(steamLanguages))
		case sfFSRSharpness:
			ss.FSRSharpness = clamp(ss.FSRSharpness+delta, 0, 20)
		case sfWidth:
			ss.Width = clamp(ss.Width+delta*10, 640, 7680)
		case sfHeight:
			ss.Height = clamp(ss.Height+delta*10, 480, 4320)
		case sfRefreshRate:
			rates := []int{0, 30, 40, 48, 50, 60, 75, 90, 120, 144, 165, 240}
			ss.RefreshRate = stepThrough(rates, ss.RefreshRate, delta)
	}
}

func (m Model) activateField() (tea.Model, tea.Cmd) {
	ss := &m.steamSettings
	switch m.steamCursor {
		// Boole
		case sfAutoResolution:
			ss.AutoResolution = !ss.AutoResolution
			m.steamDirty = true
		case sfVSync:
			ss.VSync = !ss.VSync
			m.steamDirty = true
		case sfHDR:
			ss.HDR = !ss.HDR
			m.steamDirty = true
		case sfMangoApp:
			ss.MangoApp = !ss.MangoApp
			m.steamDirty = true
		case sfForceComposite:
			ss.ForceComposite = !ss.ForceComposite
			m.steamDirty = true
		case sfBigPicture:
			ss.BigPicture = !ss.BigPicture
			m.steamDirty = true
		case sfVR:
			ss.VR = !ss.VR
			m.steamDirty = true
		case sfNoCEFSandbox:
			ss.NoCEFSandbox = !ss.NoCEFSandbox
			m.steamDirty = true
		case sfTCPMode:
			ss.TCPMode = !ss.TCPMode
			m.steamDirty = true
		case sfNoVerify:
			ss.NoVerify = !ss.NoVerify
			m.steamDirty = true
		case sfFullDesktop:
			ss.FullDesktop = !ss.FullDesktop
			m.steamDirty = true
		case sfAllowHidCrpt:
			ss.AllowHidCrpt = !ss.AllowHidCrpt
			m.steamDirty = true
		case sfDisableHWA:
			ss.DisableHWA = !ss.DisableHWA
			m.steamDirty = true
			// Select — enter = next
		case sfScalingMode:
			ss.ScalingMode = wrap(ss.ScalingMode+1, len(scalingModes))
			m.steamDirty = true
		case sfScalingFilter:
			ss.ScalingFilter = wrap(ss.ScalingFilter+1, len(scalingFilters))
			m.steamDirty = true
		case sfBetaChannel:
			ss.BetaChannel = wrap(ss.BetaChannel+1, len(betaChannels))
			m.steamDirty = true
		case sfLanguage:
			ss.Language = wrap(ss.Language+1, len(steamLanguages))
			m.steamDirty = true
			// Tekst
		case sfExtraFlags:
			m.textEditing = true
			m.textField = sfExtraFlags
			m.textBuf = ss.ExtraFlags
			// Akcje
		case sfSave:
			ss2 := m.steamSettings
			return m, func() tea.Msg {
				return steamSavedMsg{err: saveSteamSettings(ss2)}
			}
		case sfCancel:
			m.state = stateMenu
		case sfDefaults:
			m.steamSettings = defaultSteamSettings()
			m.steamDirty = true
	}
	return m, nil
}

// ── Zapis ustawień ────────────────────────────────────────────────────────────

func saveSteamSettings(ss SteamSettings) error {
	gm := findGamescopeManager()
	if gm == "" {
		return fmt.Errorf("gamescope-manager nie znaleziony")
	}

	type kv struct{ k, v string }
	var settings []kv

	// Rozdzielczość
	if ss.AutoResolution {
		settings = append(settings, kv{"width", "0"}, kv{"height", "0"})
	} else {
		settings = append(settings,
				  kv{"width", fmt.Sprintf("%d", ss.Width)},
				  kv{"height", fmt.Sprintf("%d", ss.Height)},
		)
	}
	settings = append(settings, kv{"refresh", fmt.Sprintf("%d", ss.RefreshRate)})

	// Gamescope
	settings = append(settings,
			  kv{"vsync", boolStr(ss.VSync)},
			  kv{"hdr", boolStr(ss.HDR)},
			  kv{"mangoapp", boolStr(ss.MangoApp)},
			  kv{"force_composite", boolStr(ss.ForceComposite)},
			  kv{"scaling_mode", scalingModes[ss.ScalingMode]},
		   kv{"scaling_filter", scalingFilters[ss.ScalingFilter]},
		   kv{"fsr_sharpness", fmt.Sprintf("%d", ss.FSRSharpness)},
	)

	// Steam
	settings = append(settings,
			  kv{"big_picture", boolStr(ss.BigPicture)},
			  kv{"vr", boolStr(ss.VR)},
			  kv{"no_cef_sandbox", boolStr(ss.NoCEFSandbox)},
			  kv{"tcp_mode", boolStr(ss.TCPMode)},
			  kv{"no_verify", boolStr(ss.NoVerify)},
			  kv{"full_desktop_res", boolStr(ss.FullDesktop)},
			  kv{"allow_hidcrypto", boolStr(ss.AllowHidCrpt)},
			  kv{"disable_hwa", boolStr(ss.DisableHWA)},
			  kv{"beta_channel", betaChannelKeys[ss.BetaChannel]},
		   kv{"language", steamLanguageKeys[ss.Language]},
		   kv{"steam_extra_flags", ss.ExtraFlags},
	)

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

func findGamescopeManager() string {
	if _, err := os.Stat("/usr/bin/gamescope-manager"); err == nil {
		return "/usr/bin/gamescope-manager"
	}
	if p, err := exec.LookPath("gamescope-manager"); err == nil {
		return p
	}
	return ""
}

func loadSteamSettingsFromFile() SteamSettings {
	ss := defaultSteamSettings()
	gm := findGamescopeManager()
	if gm == "" {
		return ss
	}
	out, err := exec.Command("sudo", gm, "config").Output()
	if err != nil {
		return ss
	}
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
	pb := func(k string, def bool) bool {
		if v, ok := vals[k]; ok {
			return v == "true"
		}
		return def
	}
	pi := func(k string, def int) int {
		if v, ok := vals[k]; ok {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
				return n
			}
		}
		return def
	}
	ps := func(k string) string { return vals[k] }

	w := pi("width", 0)
	h := pi("height", 0)
	ss.AutoResolution = (w == 0 && h == 0)
	if !ss.AutoResolution {
		ss.Width = w
		ss.Height = h
	}
	ss.RefreshRate = pi("refresh", 0)

	ss.VSync = pb("vsync", ss.VSync)
	ss.HDR = pb("hdr", ss.HDR)
	ss.MangoApp = pb("mangoapp", ss.MangoApp)
	ss.ForceComposite = pb("force_composite", ss.ForceComposite)

	sm := ps("scaling_mode")
	for i, v := range scalingModes {
		if v == sm {
			ss.ScalingMode = i
			break
		}
	}
	sf := ps("scaling_filter")
	for i, v := range scalingFilters {
		if v == sf {
			ss.ScalingFilter = i
			break
		}
	}
	ss.FSRSharpness = pi("fsr_sharpness", ss.FSRSharpness)

	ss.BigPicture = pb("big_picture", ss.BigPicture)
	ss.VR = pb("vr", ss.VR)
	ss.NoCEFSandbox = pb("no_cef_sandbox", ss.NoCEFSandbox)
	ss.TCPMode = pb("tcp_mode", ss.TCPMode)
	ss.NoVerify = pb("no_verify", ss.NoVerify)
	ss.FullDesktop = pb("full_desktop_res", ss.FullDesktop)
	ss.AllowHidCrpt = pb("allow_hidcrypto", ss.AllowHidCrpt)
	ss.DisableHWA = pb("disable_hwa", ss.DisableHWA)

	bc := ps("beta_channel")
	for i, k := range betaChannelKeys {
		if k == bc {
			ss.BetaChannel = i
			break
		}
	}
	lang := ps("language")
	for i, k := range steamLanguageKeys {
		if k == lang {
			ss.Language = i
			break
		}
	}
	if ef := ps("steam_extra_flags"); ef != "(brak)" {
		ss.ExtraFlags = ef
	}
	return ss
}

// ── execAction ────────────────────────────────────────────────────────────────

func (m Model) execAction(action menuAction) (tea.Model, tea.Cmd) {
	switch action {
		case actionQuit:
			return m, tea.Quit

		case actionSwitchGame:
			m.state = stateLoading
			m.loadingMsg = "Przelaczam na tryb gry..."
			m.doneMsg = "Tryb gry aktywny!"
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				return switchDoneMsg{err: SwitchToGame()}
			})

		case actionSwitchDesktop:
			m.state = stateLoading
			m.loadingMsg = "Przelaczam na KDE Plasma..."
			m.doneMsg = "Tryb pulpitu aktywny!"
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				return switchDoneMsg{err: SwitchToDesktop()}
			})

		case actionInfo:
			m.state = stateLoading
			m.loadingMsg = "Zbieram informacje..."
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				return infoReadyMsg{info: GatherInfo()}
			})

		case actionSteamSettings:
			m.state = stateLoading
			m.loadingMsg = "Wczytuje konfiguracje..."
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				ss := loadSteamSettingsFromFile()
				return infoReadyMsg{info: GatherInfo(), _steamSettings: &ss}
			})

		case actionHealth:
			m.state = stateLoading
			m.loadingMsg = "Sprawdzam gotowos systemu..."
			return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
				return healthDoneMsg{results: runHealthChecks(), canStart: true}
			})

		case actionGamescope:
			return m, tea.Sequence(tea.Quit, func() tea.Msg { return nil })
	}
	return m, nil
}


// runHealthChecks uruchamia sudo gamescope-manager health i parsuje wynik.
func runHealthChecks() []healthEntry {
	gm := findGamescopeManager()
	if gm == "" {
		return []healthEntry{{name: "gamescope-manager", ok: false, fatal: true, msg: "nie znaleziony"}}
	}
	out, err := exec.Command("sudo", gm, "health").Output()
	if err != nil {
		return []healthEntry{{name: "health check", ok: false, fatal: false, msg: err.Error()}}
	}
	var entries []healthEntry
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var ok, fatal bool
		var name, msg string
		switch {
			case strings.HasPrefix(line, "[OK]"):
				ok = true
				rest := strings.TrimPrefix(line, "[OK]")
				name = strings.TrimSpace(rest)
			case strings.HasPrefix(line, "[!!]"):
				fatal = true
				rest := strings.TrimPrefix(line, "[!!]")
				parts := strings.SplitN(rest, ":", 2)
				name = strings.TrimSpace(parts[0])
				if len(parts) > 1 {
					msg = strings.TrimSpace(parts[1])
				}
			case strings.HasPrefix(line, "[??]"):
				rest := strings.TrimPrefix(line, "[??]")
				parts := strings.SplitN(rest, ":", 2)
				name = strings.TrimSpace(parts[0])
				if len(parts) > 1 {
					msg = strings.TrimSpace(parts[1])
				}
			default:
				continue
		}
		entries = append(entries, healthEntry{name: name, ok: ok, fatal: fatal, msg: msg})
	}
	return entries
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
		case stateHealth:
			return m.viewHealth()
	}
	return ""
}

// ── Widok Menu ────────────────────────────────────────────────────────────────

func (m Model) viewMenu() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n")
	sb.WriteString("  Aktywny tryb: " + ModeBadge(m.currentMode) + "\n")
	sb.WriteString("  " + Divider(m.width-4) + "\n\n")

	for i, item := range menuItems {
		isSelected := i == m.cursor
		if isSelected {
			cursor := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("  >")
			label := lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(item.label)
			sub := lipgloss.NewStyle().Foreground(ColorSecondary).Render("    " + item.sublabel)
			row := fmt.Sprintf("%s  %s", cursor, label)
			sb.WriteString(lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1B4B")).Width(m.width - 4).Render(row))
			sb.WriteString("\n")
			sb.WriteString(lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1B4B")).Width(m.width - 4).Render(sub))
		} else {
			label := StyleCmdName.Render(item.label)
			sub := StyleCmdDesc.Render("    " + item.sublabel)
			sb.WriteString(fmt.Sprintf("     %s", label))
			sb.WriteString("\n" + sub)
		}
		sb.WriteString("\n\n")
	}

	sb.WriteString("  " + Divider(m.width-4) + "\n")
	sb.WriteString("  " + StyleCmdDesc.Render("k/j  nawigacja   enter  wybierz   q  wyjdz") + "\n")
	return sb.String()
}

// ── Widok Info ────────────────────────────────────────────────────────────────

func (m Model) viewInfo() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).
	Render("  Informacje o srodowisku") + "\n")
	sb.WriteString("  " + Divider(m.width-4) + "\n\n")

	rows := []struct{ label, value string }{
		{"Dystrybucja", m.info.PrettyName},
		{"Debian", m.info.DebianVer},
		{"Jadro", m.info.Kernel},
		{"GPU", m.info.GPU},
		{"Gamescope", m.info.Gamescope},
		{"Steam", m.info.Steam},
		{"Tryb", m.info.CurrentMode},
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
	sb.WriteString("  " + StyleCmdDesc.Render("esc  powrot") + "\n")
	return sb.String()
}

// ── Widok Health ──────────────────────────────────────────────────────────────

func (m Model) viewHealth() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).
	Render("  Health check — gotowos systemu") + "\n")
	sb.WriteString("  " + Divider(m.width-4) + "\n\n")

	lblW := lipgloss.NewStyle().Foreground(ColorGray).Width(30)
	for _, r := range m.healthResults {
		var icon string
		if r.ok {
			icon = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true).Render("[OK]")
		} else if r.fatal {
			icon = lipgloss.NewStyle().Foreground(ColorRed).Bold(true).Render("[!!]")
		} else {
			icon = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true).Render("[??]")
		}
		name := lblW.Render(r.name)
		var msg string
		if r.ok {
			msg = lipgloss.NewStyle().Foreground(ColorGreen).Render("OK")
		} else {
			msg = lipgloss.NewStyle().Foreground(ColorYellow).Render(r.msg)
		}
		sb.WriteString(fmt.Sprintf("  %s  %s  %s\n", icon, name, msg))
	}

	sb.WriteString("\n  " + Divider(m.width-4) + "\n  ")
	if m.healthOK {
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorGreen).Bold(true).
		Render("System gotowy do uruchomienia trybu gry."))
	} else {
		sb.WriteString(lipgloss.NewStyle().Foreground(ColorRed).Bold(true).
		Render("Bledy krytyczne — napraw problemy przed uruchomieniem."))
	}
	sb.WriteString("\n\n  " + StyleCmdDesc.Render("esc  powrot") + "\n")
	return sb.String()
}

// ── Widok Ustawień ────────────────────────────────────────────────────────────

func (m Model) viewSteamSettings() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")

	title := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render("  Ustawienia")
	if m.steamDirty {
		title += lipgloss.NewStyle().Foreground(ColorYellow).Render("  [niezapisane]")
	}
	sb.WriteString(title + "\n")
	sb.WriteString("  " + Divider(m.width-4) + "\n\n")

	ss := m.steamSettings
	lblW := lipgloss.NewStyle().Foreground(ColorGray).Width(30)
	selStyle := lipgloss.NewStyle().Background(lipgloss.Color("#1E1B4B")).Foreground(ColorWhite).Bold(true)
	cur := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render(" > ")
	noCur := "   "
	sectionStyle := lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorDim)

	renderSectionHeader := func(title string) {
		sb.WriteString("\n  " + sectionStyle.Render(title) + "\n")
		sb.WriteString("  " + dimStyle.Render(strings.Repeat("─", 40)) + "\n")
	}

	renderRow := func(f settingsField, label, value, hint string) {
		isSel := m.steamCursor == f
		lbl := lblW.Render(label)
		var row string
		if isSel {
			val := selStyle.Render(" " + value + " ")
			row = fmt.Sprintf("  %s%s : %s", cur, lbl, val)
		} else {
			val := lipgloss.NewStyle().Foreground(ColorWhite).Render(value)
			row = fmt.Sprintf("  %s%s : %s", noCur, lbl, val)
		}
		sb.WriteString(row + "\n")
		if isSel && hint != "" {
			sb.WriteString(lipgloss.NewStyle().Foreground(ColorDim).
			Render("       "+hint) + "\n")
		}
	}

	rb := func(f settingsField, label string, val bool, hint string) {
		renderRow(f, label, checkBox(val), hint)
	}
	rs := func(f settingsField, label string, idx int, opts []string, descs []string, hint string) {
		v := "< " + opts[idx] + " >"
		h := hint
		if len(descs) > idx && descs[idx] != "" {
			h = descs[idx]
		}
		renderRow(f, label, v, h)
	}
	rn := func(f settingsField, label string, val int, unit, hint string) {
		v := fmt.Sprintf("%d %s", val, unit)
		if val == 0 {
			v = "0 (auto)"
		}
		renderRow(f, label, v, hint)
	}

	// === Rozdzielczosc ===
	renderSectionHeader("ROZDZIELCZOSC")
	rb(sfAutoResolution, "Auto-detekcja (DRM)", ss.AutoResolution,
	   "0/0 = gamescope-manager wykryje natywna rozdzielczosc z DRM")
	if !ss.AutoResolution {
		rn(sfWidth, "Szerokosc (px)", ss.Width, "px", "< / > zmiana po 10px")
		rn(sfHeight, "Wysokosc (px)", ss.Height, "px", "< / > zmiana po 10px")
	}
	rn(sfRefreshRate, "Czestotliwosc (Hz)", ss.RefreshRate, "Hz", "0=auto, dostepne: 30 40 48 60 75 90 120 144 165 240")

	// === Skalowanie ===
	renderSectionHeader("SKALOWANIE")
	rs(sfScalingMode, "Tryb skalowania", ss.ScalingMode, scalingModes, scalingModeDesc, "")
	rs(sfScalingFilter, "Filtr", ss.ScalingFilter, scalingFilters, scalingFilterDesc, "")
	if scalingFilters[ss.ScalingFilter] == "fsr" || scalingFilters[ss.ScalingFilter] == "nis" {
		renderRow(sfFSRSharpness, "Ostrosc (0=max 20=min)", fmt.Sprintf("%d", ss.FSRSharpness),
			  "< / > zmiana wartosci")
	}

	// === Gamescope ===
	renderSectionHeader("GAMESCOPE")
	rb(sfVSync, "Adaptive sync / VRR", ss.VSync, "FreeSync / G-Sync (wymaga monitora)")
	rb(sfHDR, "HDR", ss.HDR, "wymaga kernela 6.4+, sterownika HDR i monitora HDR")
	rb(sfMangoApp, "MangoApp overlay", ss.MangoApp, "FPS, GPU, CPU, temp — overlay na ekranie")
	rb(sfForceComposite, "Force composite", ss.ForceComposite, "wymuszone kompozytowanie (wlacz jezeli czarny ekran)")

	// === Steam ===
	renderSectionHeader("STEAM")
	rb(sfBigPicture, "Big Picture / Deck UI", ss.BigPicture, "-gamepadui -steamos3 (pelny interfejs TV)")
	rb(sfVR, "Tryb VR", ss.VR, "-vr — uruchamia SteamVR")
	rb(sfNoCEFSandbox, "Bez sandbox CEF", ss.NoCEFSandbox, "szybszy start, mniej RAM (-no-cef-sandbox)")
	rb(sfTCPMode, "TCP zamiast socket", ss.TCPMode, "gdy masz problemy z polaczeniem (-nosharedmemory)")
	rb(sfNoVerify, "Pomij weryfikacje", ss.NoVerify, "szybszy start Steam (-noverifyfiles)")
	rb(sfFullDesktop, "Pelna rozdzielczosc", ss.FullDesktop, "(-fulldesktopres)")
	rb(sfAllowHidCrpt, "HID Crypto", ss.AllowHidCrpt, "szyfrowane kontrolery (-allow-hidcrypto)")
	rb(sfDisableHWA, "Bez GPU w UI", ss.DisableHWA, "wylacz GPU w interfejsie Steam (-disable-gpu)")
	rs(sfBetaChannel, "Kanal beta", ss.BetaChannel, betaChannels, nil, "")
	rs(sfLanguage, "Jezyk Steam", ss.Language, steamLanguages, nil, "")

	// Dodatkowe flagi
	efVal := ss.ExtraFlags
	if m.textEditing && m.steamCursor == sfExtraFlags {
		efVal = m.textBuf + "_"
	}
	if efVal == "" {
		efVal = "(brak)"
	}
	renderRow(sfExtraFlags, "Dodatkowe flagi Steam", efVal, "enter = edytuj, wpisz flagi Steam np. -silent -console")

	// Akcje
	sb.WriteString("\n  " + Divider(m.width-4) + "\n  ")
	actions := []struct {
		f     settingsField
		label string
	}{
		{sfSave, "Zapisz"},
		{sfCancel, "Anuluj"},
		{sfDefaults, "Domyslne"},
	}
	for _, a := range actions {
		isSel := m.steamCursor == a.f
		var rendered string
		if isSel {
			rendered = selStyle.Padding(0, 1).Render(a.label)
			sb.WriteString("  " + cur + rendered)
		} else {
			rendered = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true).Render(a.label)
			sb.WriteString("  " + noCur + "  " + rendered)
		}
		sb.WriteString("  ")
	}
	sb.WriteString("\n\n")

	if m.textEditing {
		sb.WriteString("  " + StyleCmdDesc.Render("Edytujesz tekst — enter zatwierdz   esc anuluj") + "\n")
	} else {
		sb.WriteString("  " + StyleCmdDesc.Render("k/j  nawigacja   spacja/enter  toggle   < >  lista/liczba   esc  powrot") + "\n")
	}
	return sb.String()
}

// checkBox zwraca [X] ON lub [ ] OFF
func checkBox(b bool) string {
	if b {
		return "[X] ON "
	}
	return "[ ] OFF"
}

// ── Widok Loading / Done / Error ──────────────────────────────────────────────

func (m Model) viewLoading() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("  %s  %s\n",
				   m.spinner.View(),
				   lipgloss.NewStyle().Foreground(ColorWhite).Bold(true).Render(m.loadingMsg)))
	return sb.String()
}

func (m Model) viewDone() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")
	sb.WriteString(StyleSuccess.Render("  OK  ") + StyleSuccess.Render(m.doneMsg) + "\n\n")
	sb.WriteString("  " + StyleCmdDesc.Render("enter / esc  powrot") + "\n")
	return sb.String()
}

func (m Model) viewError() string {
	var sb strings.Builder
	sb.WriteString(renderBanner())
	sb.WriteString("\n\n")
	sb.WriteString(StyleError.Render("  BLAD:") + "\n")
	sb.WriteString("  " + StyleValueBad.Render(m.errMsg) + "\n\n")
	sb.WriteString("  " + StyleCmdDesc.Render("enter / esc  powrot") + "\n")
	return sb.String()
}

// ── Banner ────────────────────────────────────────────────────────────────────

func renderBanner() string {
	line1 := StyleBannerTitle.Render("  HackerOS Gaming Edition")
	line2 := StyleBannerSub.Render("  gaming-cli") +
	StyleDivider.Render(" - ") +
	StyleBannerTag.Render("v0.2.0  ·  Debian Testing (Forky)  ·  PC / Laptop")
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

// ── Helpers ───────────────────────────────────────────────────────────────────

func wrap(v, n int) int {
	return ((v % n) + n) % n
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func stepThrough(vals []int, current, delta int) int {
	for i, v := range vals {
		if v == current {
			return vals[clamp(i+delta, 0, len(vals)-1)]
		}
	}
	if delta > 0 {
		return vals[0]
	}
	return vals[len(vals)-1]
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
