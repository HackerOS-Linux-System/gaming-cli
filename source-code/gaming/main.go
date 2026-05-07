package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const version = "0.0.1"

var (
	colorPrimary   = lipgloss.Color("#7C3AED")
	colorSecondary = lipgloss.Color("#A855F7")
	colorAccent    = lipgloss.Color("#22D3EE")
	colorGreen     = lipgloss.Color("#22C55E")
	colorRed       = lipgloss.Color("#EF4444")
	colorGray      = lipgloss.Color("#6B7280")
	colorYellow    = lipgloss.Color("#F59E0B")
	colorWhite     = lipgloss.Color("#F9FAFB")
	colorDim       = lipgloss.Color("#4B5563")
)

// Aliasy: krótka komenda → argumenty gaming-cli
var shortcuts = map[string][]string{
	"game":    {"switch", "game-mode"},
	"desktop": {"switch", "desktop-mode"},
	"status":  {"status"},
	"info":    {"info"},
	"tui":     {"tui"},
	"version": {"version"},
}

func banner() string {
	t := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render("HackerOS Gaming Edition")
	s := lipgloss.NewStyle().Foreground(colorSecondary).Render("gaming") +
		lipgloss.NewStyle().Foreground(colorDim).Render(" ─ ") +
		lipgloss.NewStyle().Foreground(colorAccent).Faint(true).Render("v"+version+" · wrapper dla gaming-cli")
	return t + "\n" + s + "\n"
}

func printHelp() {
	fmt.Println(banner())

	sec  := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	cmd  := lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Width(16)
	desc := lipgloss.NewStyle().Foreground(colorGray)
	arg  := lipgloss.NewStyle().Foreground(colorSecondary)

	fmt.Println(sec.Render("UŻYCIE"))
	fmt.Printf("  %s %s\n\n", arg.Render("gaming"), desc.Render("<skrót> [argumenty]"))

	fmt.Println(sec.Render("SKRÓTY"))
	items := [][]string{
		{"game",    "Przełącz na tryb gry          (= gaming-cli switch game-mode)"},
		{"desktop", "Przełącz na KDE Plasma         (= gaming-cli switch desktop-mode)"},
		{"tui",     "Otwórz interaktywne TUI        (= gaming-cli tui)"},
		{"status",  "Pokaż aktywny tryb             (= gaming-cli status)"},
		{"info",    "Informacje o środowisku        (= gaming-cli info)"},
		{"version", "Pokaż wersję"},
		{"help",    "Ta pomoc"},
	}
	for _, it := range items {
		fmt.Printf("  %s%s\n", cmd.Render(it[0]), desc.Render(it[1]))
	}

	fmt.Println()
	div := lipgloss.NewStyle().Foreground(colorDim).Render(strings.Repeat("─", 56))
	fmt.Println("  " + div)
	fmt.Println("  " + desc.Render("Pozostałe argumenty są przekazywane bezpośrednio do gaming-cli."))
	fmt.Println()
}

func ok(msg string) {
	icon := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("✔")
	text := lipgloss.NewStyle().Foreground(colorWhite).Bold(true).Render(msg)
	fmt.Printf("  %s  %s\n", icon, text)
}

func fail(msg string) {
	icon := lipgloss.NewStyle().Foreground(colorRed).Bold(true).Render("✖")
	text := lipgloss.NewStyle().Foreground(colorRed).Render(msg)
	fmt.Fprintf(os.Stderr, "  %s  %s\n", icon, text)
}

func warn(msg string) {
	icon := lipgloss.NewStyle().Foreground(colorYellow).Bold(true).Render("⚠")
	text := lipgloss.NewStyle().Foreground(colorYellow).Render(msg)
	fmt.Printf("  %s  %s\n", icon, text)
}

func main() {
	// Bez argumentów → TUI (tak jak gaming-cli)
	if len(os.Args) < 2 {
		cliPath, err := findCLI()
		if err != nil {
			fail(err.Error())
			os.Exit(1)
		}
		runCLI(cliPath)
		return
	}

	arg := strings.ToLower(os.Args[1])

	switch arg {
	case "help", "--help", "-h":
		printHelp()
		return
	case "version", "--version", "-v":
		fmt.Println(banner())
		return
	}

	cliPath, err := findCLI()
	if err != nil {
		fail(err.Error())
		os.Exit(1)
	}

	var finalArgs []string
	if mapped, ok := shortcuts[arg]; ok {
		finalArgs = append(mapped, os.Args[2:]...)
	} else {
		finalArgs = os.Args[1:]
	}

	// Pokaż mini-banner dla akcji przełączenia
	if len(finalArgs) >= 2 && finalArgs[0] == "switch" {
		fmt.Println(banner())
		switch finalArgs[1] {
		case "game-mode":
			warn("Przełączam na tryb gry…")
		case "desktop-mode":
			warn("Przełączam na tryb pulpitu…")
		}
	}

	cmd := exec.Command(cliPath, finalArgs...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if e, ok2 := err.(*exec.ExitError); ok2 {
			os.Exit(e.ExitCode())
		}
		fail(err.Error())
		os.Exit(1)
	}
}

func runCLI(path string) {
	cmd := exec.Command(path)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			os.Exit(e.ExitCode())
		}
		fail(err.Error())
		os.Exit(1)
	}
}

func findCLI() (string, error) {
	if _, err := os.Stat("/usr/bin/gaming-cli"); err == nil {
		return "/usr/bin/gaming-cli", nil
	}
	p, err := exec.LookPath("gaming-cli")
	if err != nil {
		return "", fmt.Errorf(
			"nie znaleziono 'gaming-cli' w /usr/bin ani w PATH\n" +
				"Upewnij się że HackerOS Gaming Edition jest poprawnie zainstalowane",
		)
	}
	return p, nil
}
