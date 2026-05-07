package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"gamescope-manager/src"
)

const version = "0.0.1"

// ── Kolory ─────────────────────────────────────────────────────────────────────

var (
	cPrimary   = lipgloss.Color("#7C3AED")
	cSecondary = lipgloss.Color("#A855F7")
	cAccent    = lipgloss.Color("#22D3EE")
	cGreen     = lipgloss.Color("#22C55E")
	cRed       = lipgloss.Color("#EF4444")
	cYellow    = lipgloss.Color("#F59E0B")
	cGray      = lipgloss.Color("#6B7280")
	cWhite     = lipgloss.Color("#F9FAFB")
	cDim       = lipgloss.Color("#4B5563")
)

func prim(s string) string  { return lipgloss.NewStyle().Foreground(cPrimary).Bold(true).Render(s) }
func acc(s string) string   { return lipgloss.NewStyle().Foreground(cAccent).Bold(true).Render(s) }
func gray(s string) string  { return lipgloss.NewStyle().Foreground(cGray).Render(s) }
func dim(s string) string   { return lipgloss.NewStyle().Foreground(cDim).Render(s) }
func good(s string) string  { return lipgloss.NewStyle().Foreground(cGreen).Bold(true).Render(s) }
func bad(s string) string   { return lipgloss.NewStyle().Foreground(cRed).Bold(true).Render(s) }
func warn2(s string) string { return lipgloss.NewStyle().Foreground(cYellow).Bold(true).Render(s) }
func bold(s string) string  { return lipgloss.NewStyle().Foreground(cWhite).Bold(true).Render(s) }
func sec(s string) string   { return lipgloss.NewStyle().Foreground(cSecondary).Render(s) }

func divider(n int) string { return dim(strings.Repeat("─", n)) }

func printBanner() {
	title := prim("  HackerOS Gaming Edition") + "\n" +
		sec("  gamescope-manager") +
		dim(" ─ ") +
		lipgloss.NewStyle().Foreground(cAccent).Faint(true).
			Render(fmt.Sprintf("v%s  ·  Debian Testing (Forky)  ·  PC / Laptop", version))
	fmt.Println(title)
	fmt.Println()
}

func printOK(msg string)   { fmt.Printf("  %s  %s\n", good("✔"), bold(msg)) }
func printErr(msg string)  { fmt.Fprintf(os.Stderr, "  %s  %s\n", bad("✖"), bad(msg)) }
func printWarn(msg string) { fmt.Printf("  %s  %s\n", warn2("⚠"), warn2(msg)) }

// ── Help ───────────────────────────────────────────────────────────────────────

func printHelp() {
	printBanner()

	cmdW := lipgloss.NewStyle().Foreground(cAccent).Bold(true).Width(32)
	descW := lipgloss.NewStyle().Foreground(cGray)
	secW := lipgloss.NewStyle().Foreground(cPrimary).Bold(true)

	fmt.Println(secW.Render("  UŻYCIE"))
	fmt.Printf("  %s %s\n\n", sec("gamescope-manager"), gray("<komenda> [opcje]"))

	fmt.Println(secW.Render("  KOMENDY"))
	cmds := [][]string{
		{"start",               "Uruchom sesję gamescope + Steam BPM"},
		{"start --no-steam",    "Uruchom gamescope bez Steam"},
		{"stop",                "Zatrzymaj aktywną sesję"},
		{"restart",             "Zrestartuj sesję"},
		{"status",              "Status sesji i konfiguracja"},
		{"config",              "Pokaż konfigurację (.hk)"},
		{"config set <k> <v>",  "Ustaw klucz konfiguracji"},
		{"config reset",        "Przywróć konfigurację domyślną"},
		{"version",             "Pokaż wersję"},
		{"help",                "Pokaż tę pomoc"},
	}
	for _, c := range cmds {
		fmt.Printf("  %s%s\n", cmdW.Render(c[0]), descW.Render(c[1]))
	}

	fmt.Printf("\n%s\n", secW.Render("  OPCJE DLA 'start'"))
	opts := [][]string{
		{"--steam",          "Uruchom z Steam BPM (domyślnie)"},
		{"--no-steam",       "Sam gamescope bez Steam"},
		{"--width <px>",     "Szerokość (domyślnie: 1920)"},
		{"--height <px>",    "Wysokość  (domyślnie: 1080)"},
		{"--refresh <hz>",   "Częstotliwość odświeżania (domyślnie: 60)"},
		{"--hdr",            "Włącz HDR (wymaga obsługi GPU)"},
		{"--no-mangoapp",    "Wyłącz overlay FPS"},
		{"--no-vsync",       "Wyłącz synchronizację pionową"},
		{"--no-fullscreen",  "Tryb okienkowy"},
		{"--extra \"flagi\"","Dodatkowe flagi gamescope"},
	}
	for _, o := range opts {
		fmt.Printf("  %s%s\n", cmdW.Render(o[0]), descW.Render(o[1]))
	}

	fmt.Println()
	fmt.Println("  " + divider(58))
	fmt.Println("  " + gray("Konfiguracja: /etc/hackeros/gamescope-manager.hk (format .hk)"))
	fmt.Println("  " + gray("Dokumentacja .hk: https://hackeros-linux-system.github.io/HackerOS-Website/tools-docs/hk.html"))
	fmt.Println("  " + gray("Przeznaczony wyłącznie dla HackerOS Gaming Edition — PC i laptopy."))
	fmt.Println()
}

// ── Main ───────────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	cmd := strings.ToLower(os.Args[1])

	if cmd != "help" && cmd != "--help" && cmd != "-h" &&
		cmd != "version" && cmd != "--version" {
		if err := src.CheckDistro(); err != nil {
			printBanner()
			printErr(err.Error())
			os.Exit(1)
		}
	}

	ensureDirs()
	args := os.Args[2:]

	switch cmd {
	case "start":
		cfg := src.Load()
		parseStartFlags(&cfg, args)
		printBanner()
		printWarn("Uruchamiam sesję gamescope…")
		if err := src.Start(cfg); err != nil {
			printErr(err.Error())
			os.Exit(1)
		}

	case "stop":
		printBanner()
		printWarn("Zatrzymuję sesję gamescope…")
		if err := src.Stop(); err != nil {
			printErr(err.Error())
			os.Exit(1)
		}
		printOK("Sesja zatrzymana.")

	case "restart":
		printBanner()
		printWarn("Restartuję sesję gamescope…")
		_ = src.Stop()
		time.Sleep(2 * time.Second)
		cfg := src.Load()
		parseStartFlags(&cfg, args)
		if err := src.Start(cfg); err != nil {
			printErr(err.Error())
			os.Exit(1)
		}

	case "status":
		printBanner()
		src.ShowStatus()

	case "config":
		printBanner()
		handleConfig(args)

	case "version", "--version":
		printBanner()
		fmt.Printf("  gamescope-manager %s\n", version)
		fmt.Printf("  HackerOS Gaming Edition · Debian Testing (Forky)\n")
		fmt.Printf("  Konfiguracja: %s\n\n", src.ConfigFile)

	case "help", "--help", "-h":
		printHelp()

	default:
		printBanner()
		printErr(fmt.Sprintf("nieznana komenda: %q", cmd))
		fmt.Println("  Użyj " + acc("gamescope-manager help") + " aby uzyskać pomoc.")
		os.Exit(1)
	}
}

// ── Config handler ─────────────────────────────────────────────────────────────

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
			printErr("Użycie: config set <klucz> <wartość>")
			os.Exit(1)
		}
		cfg := src.Load()
		if err := src.SetKey(&cfg, args[1], args[2]); err != nil {
			printErr(err.Error())
			os.Exit(1)
		}
		if err := src.Save(cfg); err != nil {
			printErr("Błąd zapisu konfiguracji: " + err.Error())
			os.Exit(1)
		}
		printOK(fmt.Sprintf("Ustawiono: %s = %s", args[1], args[2]))
	case "reset":
		cfg := src.Default()
		if err := src.Save(cfg); err != nil {
			printErr(err.Error())
			os.Exit(1)
		}
		printOK("Konfiguracja przywrócona do domyślnej.")
	default:
		printErr(fmt.Sprintf("nieznana subkomenda config: %q", args[0]))
		os.Exit(1)
	}
}

// ── Start flags ────────────────────────────────────────────────────────────────

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
				fmt.Sscanf(args[i], "%d", &cfg.Width)
			}
		case "--height":
			if i+1 < len(args) {
				i++
				fmt.Sscanf(args[i], "%d", &cfg.Height)
			}
		case "--refresh":
			if i+1 < len(args) {
				i++
				fmt.Sscanf(args[i], "%d", &cfg.RefreshRate)
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

func ensureDirs() {
	for _, d := range []string{
		"/var/lib/hackeros-gaming",
		"/var/log/hackeros-gaming",
		"/etc/hackeros",
	} {
		_ = os.MkdirAll(d, 0755)
	}
}
