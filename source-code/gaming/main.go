package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const wrapperVersion = "0.0.1"

// shortcuts mapuje krótkie aliasy na argumenty gaming-cli.
var shortcuts = map[string][]string{
	"game":    {"switch", "game-mode"},
	"desktop": {"switch", "desktop-mode"},
	"status":  {"status"},
	"info":    {"info"},
	"version": {"version"},
}

func printHelp() {
	fmt.Printf(`gaming %s — wrapper dla gaming-cli (HackerOS Gaming Edition)

UŻYCIE:
  gaming <skrót> [argumenty]

SKRÓTY:
  game         Przełącz na tryb gry     (= gaming-cli switch game-mode)
  desktop      Przełącz na KDE Plasma   (= gaming-cli switch desktop-mode)
  status       Pokaż aktywny tryb       (= gaming-cli status)
  info         Informacje o środowisku  (= gaming-cli info)
  version      Pokaż wersję
  help         Ta pomoc

  Inne argumenty są przekazywane bezpośrednio do gaming-cli.

PRZYKŁADY:
  gaming game
  gaming desktop
  gaming status
  gaming gamescope --width 2560 --height 1440

`, wrapperVersion)
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	arg := strings.ToLower(os.Args[1])

	switch arg {
	case "help", "--help", "-h":
		printHelp()
		return
	case "version", "--version", "-v":
		fmt.Printf("gaming (wrapper) %s → gaming-cli %s\n", wrapperVersion, wrapperVersion)
		return
	}

	// Znajdź gaming-cli
	cliPath, err := findGamingCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[gaming] Błąd: %v\n", err)
		os.Exit(1)
	}

	// Zbuduj argumenty dla gaming-cli
	var finalArgs []string
	if mapped, ok := shortcuts[arg]; ok {
		finalArgs = append(finalArgs, mapped...)
		finalArgs = append(finalArgs, os.Args[2:]...)
	} else {
		// Przekaż bezpośrednio
		finalArgs = os.Args[1:]
	}

	cmd := exec.Command(cliPath, finalArgs...)
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

// findGamingCLI szuka pliku wykonywalnego gaming-cli.
func findGamingCLI() (string, error) {
	// Preferuj instalację systemową
	if _, err := os.Stat("/usr/bin/gaming-cli"); err == nil {
		return "/usr/bin/gaming-cli", nil
	}
	// Fallback: PATH
	p, err := exec.LookPath("gaming-cli")
	if err != nil {
		return "", fmt.Errorf(
			"nie znaleziono 'gaming-cli' w /usr/bin ani w PATH\n" +
				"Upewnij się że HackerOS Gaming Edition jest poprawnie zainstalowane",
		)
	}
	return p, nil
}
