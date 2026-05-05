package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/HackerOS-Linux-System/gaming-cli/internal/distro"
	"github.com/HackerOS-Linux-System/gaming-cli/internal/mode"
	"github.com/HackerOS-Linux-System/gaming-cli/internal/session"
)

const (
	version = "1.0.0"
	banner  = `
  _   _            _              ___  ____  
 | | | | __ _  ___| | _____ _ __|/ _ \/ ___| 
 | |_| |/ _' |/ __| |/ / _ \ '__| | | \___ \ 
 |  _  | (_| | (__|   <  __/ |  | |_| |___) |
 |_| |_|\__,_|\___|_|\_\___|_|   \___/|____/ 
 Gaming CLI — HackerOS Gaming Edition v` + version + `
`
)

func printHelp() {
	fmt.Print(banner)
	fmt.Println(`
UŻYCIE:
  gaming-cli <komenda> [argumenty]

KOMENDY:
  switch game-mode      Przełącz na tryb gry (Gamescope + Steam BPM)
  switch desktop-mode   Przełącz z powrotem na KDE Plasma
  status                Pokaż aktywny tryb (game/desktop)
  info                  Informacje o środowisku gaming
  gamescope [args...]   Uruchom gamescope z niestandardowymi argumentami
  help                  Pokaż tę pomoc
  version               Pokaż wersję

PRZYKŁADY:
  gaming-cli switch game-mode
  gaming-cli switch desktop-mode
  gaming-cli status
  gaming-cli gamescope --width 1920 --height 1080

UWAGA:
  Narzędzie przeznaczone wyłącznie dla HackerOS Gaming Edition (Debian Testing).
  Nie obsługuje handheldów — tylko PC i laptopy.
`)
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	// Sprawdź dystrybucję — tylko HackerOS Gaming Edition na Debianie Testing
	if err := distro.Check(); err != nil {
		fmt.Fprintf(os.Stderr, "[gaming-cli] BŁĄD: %v\n", err)
		os.Exit(1)
	}

	cmd := strings.ToLower(os.Args[1])

	switch cmd {
	case "switch":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "[gaming-cli] Błąd: podaj tryb: game-mode lub desktop-mode")
			os.Exit(1)
		}
		target := strings.ToLower(os.Args[2])
		switch target {
		case "game-mode":
			if err := mode.SwitchToGame(); err != nil {
				fmt.Fprintf(os.Stderr, "[gaming-cli] Błąd przy przełączaniu do trybu gry: %v\n", err)
				os.Exit(1)
			}
		case "desktop-mode":
			if err := mode.SwitchToDesktop(); err != nil {
				fmt.Fprintf(os.Stderr, "[gaming-cli] Błąd przy przełączaniu do trybu pulpitu: %v\n", err)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "[gaming-cli] Nieznany tryb: %q. Użyj: game-mode lub desktop-mode\n", target)
			os.Exit(1)
		}

	case "status":
		s, err := session.CurrentMode()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gaming-cli] Błąd odczytu stanu: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[gaming-cli] Aktywny tryb: %s\n", s)

	case "info":
		printInfo()

	case "gamescope":
		extraArgs := os.Args[2:]
		gsPath, err := exec.LookPath("gamescope")
		if err != nil {
			fmt.Fprintln(os.Stderr, "[gaming-cli] Błąd: gamescope nie jest zainstalowany.")
			os.Exit(1)
		}
		gsCmd := exec.Command(gsPath, extraArgs...)
		gsCmd.Stdin = os.Stdin
		gsCmd.Stdout = os.Stdout
		gsCmd.Stderr = os.Stderr
		if err := gsCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "[gaming-cli] gamescope zakończył się błędem: %v\n", err)
			os.Exit(1)
		}

	case "version":
		fmt.Printf("gaming-cli %s\nHackerOS Gaming Edition — Debian Testing (Forky)\n", version)

	case "help", "--help", "-h":
		printHelp()

	default:
		fmt.Fprintf(os.Stderr, "[gaming-cli] Nieznana komenda: %q\nUżyj 'gaming-cli help' aby uzyskać pomoc.\n", cmd)
		os.Exit(1)
	}
}

func printInfo() {
	fmt.Print(banner)

	// Kernel
	kernel := runCmd("uname", "-r")
	// Distro
	distroName := readOSRelease("PRETTY_NAME")
	// Gamescope
	gsVersion := ""
	if p, err := exec.LookPath("gamescope"); err == nil {
		out, _ := exec.Command(p, "--version").Output()
		gsVersion = strings.TrimSpace(string(out))
		if gsVersion == "" {
			gsVersion = "zainstalowany (" + p + ")"
		}
	} else {
		gsVersion = "NIE ZAINSTALOWANY"
	}
	// Steam
	steamStatus := "NIE ZAINSTALOWANY"
	if _, err := exec.LookPath("steam"); err == nil {
		steamStatus = "zainstalowany"
	} else if _, err := exec.LookPath("steam-runtime"); err == nil {
		steamStatus = "zainstalowany (runtime)"
	}
	// Tryb
	currentMode, _ := session.CurrentMode()

	fmt.Printf("  Dystrybucja : %s\n", distroName)
	fmt.Printf("  Jądro       : %s\n", kernel)
	fmt.Printf("  Gamescope   : %s\n", gsVersion)
	fmt.Printf("  Steam       : %s\n", steamStatus)
	fmt.Printf("  Tryb        : %s\n", currentMode)

	// GPU
	gpuInfo := detectGPU()
	fmt.Printf("  GPU         : %s\n", gpuInfo)
	fmt.Println()
}

func runCmd(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "nieznane"
	}
	return strings.TrimSpace(string(out))
}

func readOSRelease(key string) string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "nieznane"
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, key+"=") {
			val := strings.TrimPrefix(line, key+"=")
			val = strings.Trim(val, `"`)
			return val
		}
	}
	return "nieznane"
}

func detectGPU() string {
	// Sprawdź przez lspci
	out, err := exec.Command("lspci").Output()
	if err != nil {
		return "nieznane (lspci niedostępne)"
	}
	var gpus []string
	for _, line := range strings.Split(string(out), "\n") {
		lo := strings.ToLower(line)
		if strings.Contains(lo, "vga") || strings.Contains(lo, "3d") || strings.Contains(lo, "display") {
			// Wyciągnij nazwę po ostatnim ':'
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				gpus = append(gpus, strings.TrimSpace(parts[1]))
			}
		}
	}
	if len(gpus) == 0 {
		return "nieznane"
	}
	return strings.Join(gpus, " | ")
}

// Pomocnicza funkcja do sprawdzania czy plik istnieje
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Pomocnicza do tworzenia katalogu stanu
func ensureStateDir() error {
	dir := filepath.Join("/var/lib", "hackeros-gaming")
	return os.MkdirAll(dir, 0755)
}
