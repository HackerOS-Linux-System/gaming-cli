package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gamescope-manager/src"
)

const version = "0.0.1"

const banner = `
  ____                                     __  __
 / ___| __ _ _ __ ___   ___  ___  ___ ___|  \/  | __ _ _ __
| |  _ / _' | '_ ' _ \ / _ \/ __|/ __/ _ \ |\/| |/ _' | '__|
| |_| | (_| | | | | | |  __/\__ \ (_|  __/ |  | | (_| | |
 \____|\__,_|_| |_| |_|\___||___/\___\___|_|  |_|\__, |_|
  HackerOS Gamescope Manager v` + version + `                 |___/`

func printHelp() {
	fmt.Println(banner)
	fmt.Printf(`
UŻYCIE:
  gamescope-manager <komenda> [opcje]

KOMENDY:
  start                  Uruchom sesję gamescope + Steam Big Picture
  start --no-steam       Uruchom gamescope bez Steam
  stop                   Zatrzymaj aktywną sesję
  restart                Zrestartuj sesję
  status                 Pokaż status sesji i konfigurację
  config                 Pokaż aktualną konfigurację
  config set <k> <v>     Ustaw opcję konfiguracji
  config reset           Przywróć domyślną konfigurację
  version                Pokaż wersję
  help                   Pokaż tę pomoc

OPCJE DLA 'start':
  --steam                Uruchom z Steam BPM (domyślnie)
  --no-steam             Uruchom sam gamescope bez Steam
  --width  <px>          Szerokość (domyślnie: 1920)
  --height <px>          Wysokość  (domyślnie: 1080)
  --refresh <hz>         Częstotliwość odświeżania (domyślnie: 60)
  --hdr                  Włącz HDR (jeśli GPU obsługuje)
  --no-mangoapp          Wyłącz overlay FPS (MangoApp)
  --no-vsync             Wyłącz synchronizację pionową
  --no-fullscreen        Tryb okienkowy zamiast pełnego ekranu
  --extra "<flagi>"      Dodatkowe flagi dla gamescope

KLUCZE KONFIGURACJI (config set):
  width, height, refresh, fullscreen, vsync, hdr, mangoapp, auto_steam, extra_flags

PRZYKŁADY:
  gamescope-manager start
  gamescope-manager start --width 2560 --height 1440 --refresh 144
  gamescope-manager start --no-steam
  gamescope-manager stop
  gamescope-manager status
  gamescope-manager config set refresh 144
  gamescope-manager config set hdr true

UWAGA:
  Przeznaczony WYŁĄCZNIE dla HackerOS Gaming Edition (Debian Testing/Forky).
  Obsługuje tylko PC i laptopy — handheldowe urządzenia nie są wspierane.
`)
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	cmd := strings.ToLower(os.Args[1])

	// Weryfikacja dystrybucji — pomijana tylko dla help i version
	if cmd != "help" && cmd != "--help" && cmd != "-h" && cmd != "version" && cmd != "--version" {
		if err := src.CheckDistro(); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] BŁĄD: %v\n", err)
			os.Exit(1)
		}
	}

	// Upewnij się że katalogi istnieją
	ensureDirs()

	args := os.Args[2:]

	switch cmd {
	case "start":
		cfg := src.Load()
		parseStartFlags(&cfg, args)
		if err := src.Start(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd: %v\n", err)
			os.Exit(1)
		}

	case "stop":
		if err := src.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd: %v\n", err)
			os.Exit(1)
		}

	case "restart":
		fmt.Println("[gamescope-manager] Restartuję sesję...")
		if err := src.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Ostrzeżenie przy zatrzymaniu: %v\n", err)
		}
		time.Sleep(2 * time.Second)
		cfg := src.Load()
		parseStartFlags(&cfg, args)
		if err := src.Start(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd przy uruchamianiu: %v\n", err)
			os.Exit(1)
		}

	case "status":
		src.ShowStatus()

	case "config":
		handleConfig(args)

	case "version", "--version":
		fmt.Printf("gamescope-manager %s\nHackerOS Gaming Edition — Debian Testing (Forky)\n", version)

	case "help", "--help", "-h":
		printHelp()

	default:
		fmt.Fprintf(os.Stderr, "[gamescope-manager] Nieznana komenda: %q\nUżyj 'gamescope-manager help'.\n", cmd)
		os.Exit(1)
	}
}

// handleConfig obsługuje subkomendy zarządzania konfiguracją.
func handleConfig(args []string) {
	if len(args) == 0 {
		cfg := src.Load()
		src.Print(cfg)
		return
	}

	switch strings.ToLower(args[0]) {
	case "show", "":
		cfg := src.Load()
		src.Print(cfg)

	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "[gamescope-manager] Użycie: config set <klucz> <wartość>")
			os.Exit(1)
		}
		cfg := src.Load()
		if err := src.SetKey(&cfg, args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd: %v\n", err)
			os.Exit(1)
		}
		if err := src.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd zapisu: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[gamescope-manager] Ustawiono: %s = %s\n", args[1], args[2])

	case "reset":
		cfg := src.Default()
		if err := src.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[gamescope-manager] Konfiguracja przywrócona do wartości domyślnych.")

	default:
		fmt.Fprintf(os.Stderr, "[gamescope-manager] Nieznana subkomenda config: %q\n", args[0])
		fmt.Fprintln(os.Stderr, "Dostępne: show, set <k> <v>, reset")
		os.Exit(1)
	}
}

// parseStartFlags parsuje flagi wiersza poleceń dla komendy 'start'.
func parseStartFlags(cfg *src.Config, args []string) {
	i := 0
	for i < len(args) {
		switch strings.ToLower(args[i]) {
		case "--steam":
			cfg.AutoSteam = true
		case "--no-steam":
			cfg.AutoSteam = false
		case "--hdr":
			cfg.HDR = true
		case "--no-mangoapp":
			cfg.MangoApp = false
		case "--no-vsync":
			cfg.VSync = false
		case "--no-fullscreen":
			cfg.Fullscreen = false
		case "--width":
			if i+1 < len(args) {
				i++
				if v := parseInt(args[i], "width"); v > 0 {
					cfg.Width = v
				}
			}
		case "--height":
			if i+1 < len(args) {
				i++
				if v := parseInt(args[i], "height"); v > 0 {
					cfg.Height = v
				}
			}
		case "--refresh":
			if i+1 < len(args) {
				i++
				if v := parseInt(args[i], "refresh"); v > 0 {
					cfg.RefreshRate = v
				}
			}
		case "--extra":
			if i+1 < len(args) {
				i++
				cfg.ExtraFlags = append(cfg.ExtraFlags, strings.Fields(args[i])...)
			}
		}
		i++
	}
}

// parseInt parsuje int z argumentu wiersza poleceń, wyświetlając ostrzeżenie przy błędzie.
func parseInt(s, name string) int {
	v := 0
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil {
		fmt.Fprintf(os.Stderr, "[gamescope-manager] Ostrzeżenie: nieprawidłowa wartość dla --%s: %q\n", name, s)
		return 0
	}
	return v
}

// ensureDirs tworzy wymagane katalogi jeśli nie istnieją.
func ensureDirs() {
	dirs := []string{
		"/var/lib/hackeros-gaming",
		"/var/log/hackeros-gaming",
		"/etc/hackeros",
	}
	for _, d := range dirs {
		_ = os.MkdirAll(d, 0755)
	}
}
