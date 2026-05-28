package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"gamescope-manager/src"
)

const version = "0.1"

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

// ── Root check (wczesne) ───────────────────────────────────────────────────────

func requireRoot() {
	if os.Geteuid() != 0 {
		printBanner()
		printErr("gamescope-manager wymaga uprawnien roota")
		fmt.Fprintf(os.Stderr, "  Uzyj: sudo gamescope-manager\n\n")
		os.Exit(1)
	}
}

// ── Help ───────────────────────────────────────────────────────────────────────

func printHelp() {
	printBanner()

	cmdW := lipgloss.NewStyle().Foreground(cAccent).Bold(true).Width(34)
	descW := lipgloss.NewStyle().Foreground(cGray)
	secW := lipgloss.NewStyle().Foreground(cPrimary).Bold(true)

	fmt.Println(secW.Render("  UZYCIE"))
	fmt.Printf("  %s %s\n\n", sec("gamescope-manager"), gray("<komenda> [opcje]"))

	fmt.Println(secW.Render("  KOMENDY"))
	cmds := [][]string{
		{"start", "Uruchom sesje gamescope + Steam BPM"},
		{"start --no-steam", "Uruchom gamescope bez Steam"},
		{"stop", "Zatrzymaj aktywna sesje"},
		{"restart", "Zrestartuj sesje"},
		{"status", "Status sesji i konfiguracja"},
		{"config", "Pokaz konfiguracje (.hk)"},
		{"config set <k> <v>", "Ustaw klucz konfiguracji"},
		{"config reset", "Przywroc konfiguracje domyslna"},
		{"version", "Pokaz wersje"},
		{"help", "Pokaz te pomoc"},
	}
	for _, c := range cmds {
		fmt.Printf("  %s%s\n", cmdW.Render(c[0]), descW.Render(c[1]))
	}

	fmt.Printf("\n%s\n", secW.Render("  OPCJE DLA 'start' (gamescope)"))
	gsOpts := [][]string{
		{"--steam", "Uruchom z Steam BPM (domyslnie)"},
		{"--no-steam", "Sam gamescope bez Steam"},
		{"--width <px>", "Szerokosc (domyslnie: 1920)"},
		{"--height <px>", "Wysokosc  (domyslnie: 1080)"},
		{"--refresh <hz>", "Czestotliwosc odswiezania (domyslnie: 60)"},
		{"--hdr", "Wlacz HDR"},
		{"--no-mangoapp", "Wylacz overlay FPS"},
		{"--no-vsync", "Wylacz synchronizacje pionowa"},
		{"--no-fullscreen", "Tryb okienkowy"},
		{"--force-composite", "Wymus kompozytor gamescope"},
		{"--extra \"flagi\"", "Dodatkowe flagi gamescope"},
	}
	for _, o := range gsOpts {
		fmt.Printf("  %s%s\n", cmdW.Render(o[0]), descW.Render(o[1]))
	}

	fmt.Printf("\n%s\n", secW.Render("  OPCJE DLA 'start' (steam)"))
	stOpts := [][]string{
		{"--no-big-picture", "Uruchom Steam w trybie pulpitu (bez -tenfoot)"},
		{"--steam-vr", "Uruchom Steam VR"},
		{"--steam-no-cef-sandbox", "Wylacz sandbox CEF w Steam"},
		{"--steam-tcp", "Uzyj TCP zamiast UNIX socket"},
		{"--steam-no-verify", "Nie weryfikuj plikow gier"},
		{"--steam-beta <kanal>", "Kanal beta: beta | steamdeckbeta | beta-client"},
		{"--steam-lang <jezyk>", "Jezyk Steam: polish, english, ..."},
		{"--steam-extra \"flagi\"", "Dodatkowe flagi steam"},
	}
	for _, o := range stOpts {
		fmt.Printf("  %s%s\n", cmdW.Render(o[0]), descW.Render(o[1]))
	}

	fmt.Printf("\n%s\n", secW.Render("  ZMIENNE SRODOWISKOWE"))
	envVars := [][]string{
		{"HACKEROS_GS_WIDTH", "Nadpisz szerokosc"},
		{"HACKEROS_GS_HEIGHT", "Nadpisz wysokosc"},
		{"HACKEROS_GS_REFRESH", "Nadpisz czestotliwosc"},
		{"HACKEROS_GS_FULLSCREEN", "Nadpisz tryb pelnoekranowy (true/false)"},
		{"HACKEROS_ST_BIG_PICTURE", "Nadpisz Big Picture (true/false)"},
		{"HACKEROS_ST_BETA_CHANNEL", "Nadpisz kanal beta Steam"},
	}
	for _, e := range envVars {
		fmt.Printf("  %s%s\n", cmdW.Render(e[0]), descW.Render(e[1]))
	}

	fmt.Println()
	fmt.Println("  " + divider(60))
	fmt.Println("  " + gray("Konfiguracja: /etc/hackeros/gamescope-manager.hk (format .hk)"))
	fmt.Println("  " + gray("WYMAGA UPRAWNIEN ROOTA: sudo gamescope-manager <komenda>"))
	fmt.Println("  " + gray("Przeznaczony wylacznie dla HackerOS Gaming Edition — PC i laptopy."))
	fmt.Println()
}

// ── Main ───────────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	cmd := strings.ToLower(os.Args[1])

	// Komendy nie wymagające roota ani weryfikacji dystrybucji
	switch cmd {
		case "help", "--help", "-h":
			printHelp()
			return
		case "version", "--version":
			printBanner()
			fmt.Printf("  gamescope-manager %s\n", version)
			fmt.Printf("  HackerOS Gaming Edition · Debian Testing (Forky)\n")
			fmt.Printf("  Konfiguracja: %s\n\n", src.ConfigFile)
			return
	}

	// Wszystkie pozostałe komendy wymagają roota
	requireRoot()

	if err := src.CheckDistro(); err != nil {
		printBanner()
		printErr(err.Error())
		os.Exit(1)
	}

	ensureDirs()
	args := os.Args[2:]

	switch cmd {
		case "start":
			cfg := src.Load()
			parseStartFlags(&cfg, args)
			printBanner()
			printWarn("Uruchamiam sesje gamescope...")
			if err := src.Start(cfg); err != nil {
				printErr(err.Error())
				os.Exit(1)
			}

		case "stop":
			printBanner()
			printWarn("Zatrzymuje sesje gamescope...")
			if err := src.Stop(); err != nil {
				printErr(err.Error())
				os.Exit(1)
			}
			printOK("Sesja zatrzymana.")

		case "restart":
			printBanner()
			printWarn("Restartuję sesje gamescope...")
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

		default:
			printBanner()
			printErr(fmt.Sprintf("nieznana komenda: %q", cmd))
			fmt.Println("  Uzyj " + acc("gamescope-manager help") + " aby uzyskac pomoc.")
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
				printErr("Uzycie: config set <klucz> <wartosc>")
				os.Exit(1)
			}
			cfg := src.Load()
			if err := src.SetKey(&cfg, args[1], args[2]); err != nil {
				printErr(err.Error())
				os.Exit(1)
			}
			if err := src.Save(cfg); err != nil {
				printErr("Blad zapisu konfiguracji: " + err.Error())
				os.Exit(1)
			}
			printOK(fmt.Sprintf("Ustawiono: %s = %s", args[1], args[2]))
		case "reset":
			cfg := src.Default()
			if err := src.Save(cfg); err != nil {
				printErr(err.Error())
				os.Exit(1)
			}
			printOK("Konfiguracja przywrocona do domyslnej.")
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
			// gamescope
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
			case "--force-composite":
				cfg.ForceComposite = true
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
				// steam
			case "--no-big-picture":
				cfg.SteamBigPicture = false
			case "--steam-vr":
				cfg.SteamVR = true
			case "--steam-no-cef-sandbox":
				cfg.SteamNoCEFSandbox = true
			case "--steam-tcp":
				cfg.SteamTCP = true
			case "--steam-no-verify":
				cfg.SteamNoVerify = true
			case "--steam-beta":
				if i+1 < len(args) {
					i++
					cfg.SteamBetaChannel = args[i]
				}
			case "--steam-lang":
				if i+1 < len(args) {
					i++
					cfg.SteamLanguage = args[i]
				}
			case "--steam-extra":
				if i+1 < len(args) {
					i++
					cfg.SteamExtraFlags = args[i]
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
