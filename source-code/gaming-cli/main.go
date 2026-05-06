package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"gaming-cli/src"
)

const version = "0.0.1"

func printBanner() {
	title := lipgloss.NewStyle().Bold(true).Foreground(src.ColorPrimary).
	Render("  HackerOS Gaming Edition")
	sub := lipgloss.NewStyle().Foreground(src.ColorSecondary).
	Render("  gaming-cli") +
	lipgloss.NewStyle().Foreground(src.ColorDim).Render(" ─ ") +
	lipgloss.NewStyle().Foreground(src.ColorAccent).Faint(true).
	Render(fmt.Sprintf("v%s  ·  Debian Testing (Forky)  ·  PC / Laptop", version))
	fmt.Println(title)
	fmt.Println(sub)
	fmt.Println()
}

func printHelp() {
	printBanner()

	sectionStyle := lipgloss.NewStyle().Foreground(src.ColorPrimary).Bold(true)
	cmdStyle     := lipgloss.NewStyle().Foreground(src.ColorAccent).Bold(true).Width(24)
	descStyle    := lipgloss.NewStyle().Foreground(src.ColorGray)
	argStyle     := lipgloss.NewStyle().Foreground(src.ColorSecondary)
	divider      := src.Divider(60)

	fmt.Println(sectionStyle.Render("  UŻYCIE"))
	fmt.Println("  " + argStyle.Render("gaming-cli") + " " + descStyle.Render("[komenda] [argumenty]"))
	fmt.Println()

	fmt.Println(sectionStyle.Render("  TRYB INTERAKTYWNY"))
	fmt.Println("  " + cmdStyle.Render("gaming-cli") + descStyle.Render("Otwórz TUI (bez argumentów)"))
	fmt.Println("  " + cmdStyle.Render("gaming-cli tui") + descStyle.Render("Otwórz TUI jawnie"))
	fmt.Println()

	fmt.Println(sectionStyle.Render("  KOMENDY"))
	cmds := [][]string{
		{"switch game-mode",    "Przełącz na tryb gry (Gamescope + Steam BPM)"},
		{"switch desktop-mode", "Przełącz z powrotem na KDE Plasma"},
		{"status",              "Pokaż aktywny tryb"},
		{"info",                "Informacje o środowisku gaming"},
		{"gamescope [args]",    "Uruchom gamescope z podanymi argumentami"},
		{"version",             "Pokaż wersję"},
		{"help",                "Pokaż tę pomoc"},
	}
	for _, c := range cmds {
		fmt.Println("  " + cmdStyle.Render(c[0]) + descStyle.Render(c[1]))
	}
	fmt.Println()

	fmt.Println("  " + divider)
	fmt.Println("  " + descStyle.Render("Przeznaczony wyłącznie dla HackerOS Gaming Edition (Debian Testing/Forky)."))
	fmt.Println("  " + descStyle.Render("Obsługuje tylko PC i laptopy — handheldowe urządzenia nie są wspierane."))
	fmt.Println()
}

func printStatus() {
	mode, err := src.CurrentMode()
	if err != nil {
		printErr(err)
		os.Exit(1)
	}
	printBanner()
	label := lipgloss.NewStyle().Foreground(src.ColorGray).Render("  Aktywny tryb : ")
	badge := src.ModeBadge(mode)
	fmt.Println(label + badge)
	fmt.Println()
}

func printInfoCmd() {
	printBanner()
	info := src.GatherInfo()

	title := lipgloss.NewStyle().Foreground(src.ColorAccent).Bold(true).
	Render("  📊 Środowisko gaming")
	fmt.Println(title)
	fmt.Println("  " + src.Divider(56))

	rows := []struct{ label, value string }{
		{"Dystrybucja", info.PrettyName},
		{"Debian",      info.DebianVer},
		{"Jądro",       info.Kernel},
		{"GPU",         info.GPU},
		{"Gamescope",   info.Gamescope},
		{"Steam",       info.Steam},
		{"Tryb",        info.CurrentMode},
	}

	lStyle := lipgloss.NewStyle().Foreground(src.ColorGray).Width(14)
	sep    := lipgloss.NewStyle().Foreground(src.ColorDim).Render(" : ")

	for _, r := range rows {
		var val string
		switch {
			case strings.Contains(r.value, "NIE ZAINSTALOWANY"):
				val = src.StyleValueBad.Render(r.value)
			case strings.Contains(r.value, "game-mode"):
				val = src.StyleValueGood.Render("🎮 " + r.value)
			case strings.Contains(r.value, "desktop-mode"):
				val = src.StyleValue.Render("🖥  " + r.value)
			case strings.Contains(r.value, "zainstalowany"):
				val = src.StyleValueGood.Render(r.value)
			default:
				val = src.StyleValue.Render(r.value)
		}
		fmt.Println("  " + lStyle.Render(r.label) + sep + val)
	}
	fmt.Println()
}

func printOK(msg string) {
	icon := lipgloss.NewStyle().Foreground(src.ColorGreen).Bold(true).Render("✔")
	text := lipgloss.NewStyle().Foreground(src.ColorWhite).Bold(true).Render(msg)
	fmt.Printf("  %s  %s\n", icon, text)
}

func printErr(err error) {
	icon := lipgloss.NewStyle().Foreground(src.ColorRed).Bold(true).Render("✖")
	text := lipgloss.NewStyle().Foreground(src.ColorRed).Render(err.Error())
	fmt.Fprintf(os.Stderr, "  %s  %s\n", icon, text)
}

func printWarn(msg string) {
	icon := lipgloss.NewStyle().Foreground(src.ColorYellow).Bold(true).Render("⚠")
	text := lipgloss.NewStyle().Foreground(src.ColorYellow).Render(msg)
	fmt.Printf("  %s  %s\n", icon, text)
}

func main() {
	// Bez argumentów → TUI
	if len(os.Args) < 2 {
		if err := src.CheckDistro(); err != nil {
			printErr(err)
			os.Exit(1)
		}
		src.RunTUI()
		return
	}

	cmd := strings.ToLower(os.Args[1])

	// help i version nie wymagają weryfikacji dystrybucji
	if cmd != "help" && cmd != "--help" && cmd != "-h" &&
		cmd != "version" && cmd != "--version" {
			if err := src.CheckDistro(); err != nil {
				printErr(err)
				os.Exit(1)
			}
		}

		switch cmd {
			case "tui":
				src.RunTUI()

			case "switch":
				if len(os.Args) < 3 {
					printBanner()
					printErr(fmt.Errorf("podaj tryb: game-mode lub desktop-mode"))
					os.Exit(1)
				}
				target := strings.ToLower(os.Args[2])
				printBanner()
				switch target {
					case "game-mode":
						printWarn("Przełączam na tryb gry…")
						if err := src.SwitchToGame(); err != nil {
							printErr(err)
							os.Exit(1)
						}
						printOK("Tryb gry aktywny! Miłej gry! 🎮")

					case "desktop-mode":
						printWarn("Przełączam na tryb pulpitu…")
						if err := src.SwitchToDesktop(); err != nil {
							printErr(err)
							os.Exit(1)
						}
						printOK("KDE Plasma uruchomiona! 🖥")

					default:
						printErr(fmt.Errorf("nieznany tryb: %q — użyj game-mode lub desktop-mode", target))
						os.Exit(1)
				}

					case "status":
						printStatus()

					case "info":
						printInfoCmd()

					case "gamescope":
						if err := src.CheckDistro(); err != nil {
							printErr(err)
							os.Exit(1)
						}
						runGamescope(os.Args[2:])

					case "version", "--version":
						printBanner()
						fmt.Printf("  gaming-cli %s\n", version)
						fmt.Printf("  HackerOS Gaming Edition · Debian Testing (Forky)\n\n")

					case "help", "--help", "-h":
						printHelp()

					default:
						printBanner()
						printErr(fmt.Errorf("nieznana komenda: %q", cmd))
						fmt.Println("  Użyj " +
						lipgloss.NewStyle().Foreground(src.ColorAccent).Render("gaming-cli help") +
						" aby uzyskać pomoc.")
						os.Exit(1)
		}
}

func runGamescope(args []string) {
	gsPath := "/usr/bin/gamescope"
	if _, err := os.Stat(gsPath); err != nil {
		p, err2 := exec.LookPath("gamescope")
		if err2 != nil {
			printErr(fmt.Errorf("gamescope nie jest zainstalowany\nZainstaluj: sudo apt install gamescope"))
			os.Exit(1)
		}
		gsPath = p
	}
	printBanner()
	printWarn("Uruchamiam gamescope…")
	c := exec.Command(gsPath, args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			os.Exit(e.ExitCode())
		}
		printErr(err)
		os.Exit(1)
	}
}
