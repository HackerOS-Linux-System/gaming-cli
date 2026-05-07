package src

import "github.com/charmbracelet/lipgloss"

// Paleta kolorów HackerOS Gaming Edition
var (
	// Akcenty
	ColorPrimary   = lipgloss.Color("#7C3AED") // fiolet — główny kolor HackerOS
	ColorSecondary = lipgloss.Color("#A855F7") // jasny fiolet
	ColorAccent    = lipgloss.Color("#22D3EE") // cyan — gaming
	ColorGreen     = lipgloss.Color("#22C55E") // sukces
	ColorYellow    = lipgloss.Color("#F59E0B") // ostrzeżenie
	ColorRed       = lipgloss.Color("#EF4444") // błąd
	ColorGray      = lipgloss.Color("#6B7280") // pomocniczy
	ColorWhite     = lipgloss.Color("#F9FAFB") // tekst
	ColorDim       = lipgloss.Color("#4B5563") // wyciszony

	// Style bazowe
	Bold      = lipgloss.NewStyle().Bold(true)
	Faint     = lipgloss.NewStyle().Faint(true)

	// Nagłówek / baner
	StyleBannerBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 2)

	StyleBannerTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPrimary)

	StyleBannerSub = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	StyleBannerTag = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Faint(true)

	// Status / info
	StyleLabel = lipgloss.NewStyle().
			Foreground(ColorGray).
			Width(16)

	StyleValue = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Bold(true)

	StyleValueGood = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)

	StyleValueBad = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	StyleValueWarn = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)

	// Separator / divider
	StyleDivider = lipgloss.NewStyle().
			Foreground(ColorDim)

	// Komenda w helpie
	StyleCmdName = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StyleCmdDesc = lipgloss.NewStyle().
			Foreground(ColorGray)

	StyleCmdArg = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// Badge trybu
	StyleBadgeGame = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorWhite).
			Bold(true).
			Padding(0, 1)

	StyleBadgeDesktop = lipgloss.NewStyle().
				Background(lipgloss.Color("#1E40AF")).
				Foreground(ColorWhite).
				Bold(true).
				Padding(0, 1)

	// Błąd / sukces
	StyleError = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)

	StyleWarn = lipgloss.NewStyle().
			Foreground(ColorYellow).
			Bold(true)

	// Sekcja w helpie
	StyleSection = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			MarginTop(1)
)

// Divider zwraca poziomą linię dekoracyjną o podanej szerokości.
func Divider(width int) string {
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return StyleDivider.Render(line)
}

// Badge trybu — kolorowy znacznik GAME / DESKTOP
func ModeBadge(mode string) string {
	if mode == "game-mode" {
		return StyleBadgeGame.Render(" 🎮 GAME MODE ")
	}
	return StyleBadgeDesktop.Render(" 🖥  DESKTOP MODE ")
}
