package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const wrapperVersion = "0.0.1"

var shortcutMap = map[string][]string{
	"game":    {"switch", "game-mode"},
	"desktop": {"switch", "desktop-mode"},
	"status":  {"status"},
	"info":    {"info"},
	"help":    {"help"},
	"version": {"version"},
}

func printWrapperHelp() {
	fmt.Printf(`gaming — skrócony wrapper dla gaming-cli (HackerOS Gaming Edition v%s)

UŻYCIE:
  gaming <skrót> [args...]

SKRÓTY:
  game          Przełącz na tryb gry       (= gaming-cli switch game-mode)
  desktop       Przełącz na KDE Plasma     (= gaming-cli switch desktop-mode)
  status        Pokaż aktywny tryb         (= gaming-cli status)
  info          Informacje o środowisku    (= gaming-cli info)
  help          Ta pomoc
  version       Pokaż wersję

  Inne komendy są przekazywane bezpośrednio do gaming-cli.

PRZYKŁADY:
  gaming game
  gaming desktop
  gaming status
`, wrapperVersion)
}

func main() {
	if len(os.Args) < 2 {
		printWrapperHelp()
		os.Exit(0)
	}

	arg := strings.ToLower(os.Args[1])

	// Obsługa wersji i pomocy lokalnie
	switch arg {
	case "help", "--help", "-h":
		printWrapperHelp()
		return
	case "version", "--version", "-v":
		fmt.Printf("gaming (wrapper) %s → gaming-cli %s\n", wrapperVersion, wrapperVersion)
		return
	}

	// Znajdź gaming-cli w PATH lub /usr/bin
	gamingCLI, err := findGamingCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gaming] Błąd: %v\n", err)
		os.Exit(1)
	}

	var finalArgs []string

	if mapped, ok := shortcutMap[arg]; ok {
		// Skrót — zamień na pełne argumenty
		finalArgs = append(finalArgs, mapped...)
		finalArgs = append(finalArgs, os.Args[2:]...) // dołącz dodatkowe args
	} else {
		// Przekaż bezpośrednio do gaming-cli
		finalArgs = os.Args[1:]
	}

	cmd := exec.Command(gamingCLI, finalArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "[gaming] Błąd: %v\n", err)
		os.Exit(1)
	}
}

func findGamingCLI() (string, error) {
	// Najpierw sprawdź /usr/bin/gaming-cli (instalacja systemowa)
	if _, err := os.Stat("/usr/bin/gaming-cli"); err == nil {
		return "/usr/bin/gaming-cli", nil
	}
	// Fallback: szukaj w PATH
	p, err := exec.LookPath("gaming-cli")
	if err != nil {
		return "", fmt.Errorf("nie znaleziono 'gaming-cli' w /usr/bin ani w PATH.\nUpewnij się że HackerOS Gaming Edition jest poprawnie zainstalowane")
	}
	return p, nil
}
