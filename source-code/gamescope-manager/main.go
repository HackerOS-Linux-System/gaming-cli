package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	version    = "1.0.0"
	configFile = "/etc/hackeros/gamescope-manager.conf"
	pidFile    = "/var/lib/hackeros-gaming/gamescope-manager.pid"
	logFile    = "/var/log/hackeros-gaming/gamescope-manager.log"
)

// Config przechowuje konfigurację gamescope-managera.
type Config struct {
	// Rozdzielczość gamescope
	Width  int
	Height int
	// Refresh rate
	RefreshRate int
	// Czy uruchomić Steam automatycznie
	AutoSteam bool
	// Dodatkowe flagi gamescope
	ExtraFlags []string
	// Ścieżka do Steama
	SteamPath string
	// Tryb HDR (jeśli obsługiwany)
	HDR bool
	// Mangoapp (overlay FPS)
	MangoApp bool
	// VSync
	VSync bool
	// Fullscreen
	Fullscreen bool
}

func defaultConfig() Config {
	return Config{
		Width:       1920,
		Height:      1080,
		RefreshRate: 60,
		AutoSteam:   true,
		HDR:         false,
		MangoApp:    true,
		VSync:       true,
		Fullscreen:  true,
		ExtraFlags:  []string{},
	}
}

func printHelp() {
	fmt.Printf(`
  ____                                     __  __
 / ___| __ _ _ __ ___   ___  ___  ___ ___|  \/  | __ _ _ __
| |  _ / _' | '_ ' _ \ / _ \/ __|/ __/ _ \ |\/| |/ _' | '__|
| |_| | (_| | | | | | |  __/\__ \ (_|  __/ |  | | (_| | |
 \____|\__,_|_| |_| |_|\___||___/\___\___|_|  |_|\__, |_|
  HackerOS Gamescope Manager v%s            |___/

UŻYCIE:
  gamescope-manager <komenda> [opcje]

KOMENDY:
  start               Uruchom sesję gamescope + Steam BPM
  start --steam       Uruchom gamescope ze Steam (domyślnie)
  start --no-steam    Uruchom sam gamescope bez Steam
  stop                Zatrzymaj aktywną sesję gamescope
  restart             Zrestartuj sesję gamescope
  status              Pokaż status sesji
  config              Pokaż aktualną konfigurację
  config set <k> <v>  Ustaw opcję konfiguracji
  config reset        Przywróć domyślną konfigurację
  help                Pokaż tę pomoc
  version             Pokaż wersję

OPCJE DLA 'start':
  --width  <px>       Szerokość okna gamescope (domyślnie: 1920)
  --height <px>       Wysokość okna gamescope (domyślnie: 1080)
  --refresh <hz>      Częstotliwość odświeżania (domyślnie: 60)
  --hdr               Włącz HDR (jeśli GPU obsługuje)
  --no-mangoapp       Wyłącz overlay MangoApp (FPS)
  --no-vsync          Wyłącz synchronizację pionową
  --no-fullscreen     Uruchom w oknie zamiast pełnym ekranie
  --extra <flagi>     Dodatkowe flagi dla gamescope

PRZYKŁADY:
  gamescope-manager start
  gamescope-manager start --width 2560 --height 1440 --refresh 144
  gamescope-manager start --no-steam --no-mangoapp
  gamescope-manager stop
  gamescope-manager status
  gamescope-manager config set refresh 144

UWAGA:
  gamescope-manager jest przeznaczony WYŁĄCZNIE dla HackerOS Gaming Edition
  na PC i laptopach z Debianem Testing (Forky).
  Handheldowe urządzenia nie są obsługiwane.
`, version)
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	// Zapewnij katalogi
	ensureDirs()

	cmd := strings.ToLower(os.Args[1])
	args := os.Args[2:]

	switch cmd {
	case "start":
		cfg := loadConfig()
		parseStartFlags(&cfg, args)
		if err := startSession(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd uruchamiania sesji: %v\n", err)
			os.Exit(1)
		}

	case "stop":
		if err := stopSession(); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd zatrzymywania sesji: %v\n", err)
			os.Exit(1)
		}

	case "restart":
		_ = stopSession()
		time.Sleep(2 * time.Second)
		cfg := loadConfig()
		if err := startSession(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd restartu sesji: %v\n", err)
			os.Exit(1)
		}

	case "status":
		showStatus()

	case "config":
		handleConfig(args)

	case "version":
		fmt.Printf("gamescope-manager %s\nHackerOS Gaming Edition — Debian Testing (Forky)\n", version)

	case "help", "--help", "-h":
		printHelp()

	default:
		fmt.Fprintf(os.Stderr, "[gamescope-manager] Nieznana komenda: %q\nUżyj 'gamescope-manager help'.\n", cmd)
		os.Exit(1)
	}
}

// startSession uruchamia sesję gamescope z opcjonalnym Steam Big Picture Mode.
func startSession(cfg Config) error {
	// Sprawdź czy gamescope jest zainstalowany
	gscopePath, err := findExecutable("gamescope")
	if err != nil {
		return fmt.Errorf("gamescope nie jest zainstalowany lub nie jest w PATH.\nZainstaluj: sudo apt install gamescope")
	}

	// Sprawdź czy sesja już działa
	if isRunning() {
		return fmt.Errorf("sesja gamescope już działa (PID: %s). Użyj 'restart' aby zrestartować", readPID())
	}

	// Zbuduj argumenty gamescope
	gsArgs := buildGamescopeArgs(cfg)

	// Znajdź Steam
	var steamPath string
	if cfg.AutoSteam {
		steamPath, err = findSteam()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Ostrzeżenie: Steam nie znaleziony: %v\n", err)
			fmt.Fprintln(os.Stderr, "[gamescope-manager] Uruchamiam gamescope bez Steam.")
			cfg.AutoSteam = false
		}
	}

	// Ustaw zmienne środowiskowe
	env := buildEnvironment(cfg)

	logEntry(fmt.Sprintf("Uruchamiam sesję: gamescope %s", strings.Join(gsArgs, " ")))

	if cfg.AutoSteam && steamPath != "" {
		// Gamescope działa jako kompozytor, Steam BPM jako aplikacja wewnątrz
		// gamescope [opcje] -- steam -bigpicture -gamepadui
		gsArgs = append(gsArgs, "--")
		gsArgs = append(gsArgs, steamPath)
		gsArgs = append(gsArgs, "-tenfoot") // Big Picture / GamepadUI
	}

	fmt.Printf("[gamescope-manager] Uruchamiam: %s %s\n", gscopePath, strings.Join(gsArgs, " "))

	cmd := exec.Command(gscopePath, gsArgs...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Uruchom w tle (gamescope przejmuje cały ekran)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("nie można uruchomić gamescope: %w", err)
	}

	pid := cmd.Process.Pid
	fmt.Printf("[gamescope-manager] Sesja uruchomiona (PID: %d)\n", pid)
	logEntry(fmt.Sprintf("Sesja uruchomiona PID=%d", pid))

	// Zapisz PID
	if err := writePID(pid); err != nil {
		fmt.Fprintf(os.Stderr, "[gamescope-manager] Ostrzeżenie: nie można zapisać PID: %v\n", err)
	}

	// Obsługa sygnałów — czysty shutdown
	go handleSignals(cmd)

	// Czekaj na zakończenie gamescope
	if err := cmd.Wait(); err != nil {
		logEntry(fmt.Sprintf("Sesja zakończona z błędem: %v", err))
		// Nie traktuj błędu wyjścia jako krytyczny — Steam może się zamknąć normalnie
		fmt.Printf("[gamescope-manager] Sesja zakończona: %v\n", err)
	} else {
		logEntry("Sesja zakończona normalnie")
		fmt.Println("[gamescope-manager] Sesja gamescope zakończona.")
	}

	// Usuń PID po zakończeniu
	_ = os.Remove(pidFile)
	return nil
}

// stopSession zatrzymuje aktywną sesję gamescope.
func stopSession() error {
	if !isRunning() {
		fmt.Println("[gamescope-manager] Brak aktywnej sesji gamescope.")
		return nil
	}

	pidStr := readPID()
	pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
	if err != nil {
		return fmt.Errorf("nieprawidłowy PID w pliku stanu: %q", pidStr)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("nie można znaleźć procesu PID %d: %w", pid, err)
	}

	fmt.Printf("[gamescope-manager] Zatrzymuję sesję (PID: %d)...\n", pid)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// SIGTERM nie zadziałało — próbuj SIGKILL
		fmt.Fprintf(os.Stderr, "[gamescope-manager] SIGTERM nie powiódł się, używam SIGKILL...\n")
		_ = process.Signal(syscall.SIGKILL)
	}

	// Poczekaj na zakończenie
	time.Sleep(2 * time.Second)
	_ = os.Remove(pidFile)

	logEntry(fmt.Sprintf("Sesja zatrzymana (PID: %d)", pid))
	fmt.Println("[gamescope-manager] Sesja zatrzymana.")
	return nil
}

// showStatus wyświetla status aktywnej sesji.
func showStatus() {
	fmt.Println("[gamescope-manager] Status sesji:")

	if isRunning() {
		pid := readPID()
		fmt.Printf("  Status    : AKTYWNA\n")
		fmt.Printf("  PID       : %s\n", strings.TrimSpace(pid))

		// Sprawdź procesy
		if isProcessRunning("gamescope") {
			fmt.Println("  Gamescope : uruchomiony")
		} else {
			fmt.Println("  Gamescope : NIE URUCHOMIONY (osierocony PID?)")
		}
		if isProcessRunning("steam") {
			fmt.Println("  Steam     : uruchomiony")
		} else {
			fmt.Println("  Steam     : nie uruchomiony")
		}
		if isProcessRunning("mangoapp") {
			fmt.Println("  MangoApp  : uruchomiony")
		}
	} else {
		fmt.Println("  Status    : NIEAKTYWNA")
	}

	// Pokaż konfigurację
	cfg := loadConfig()
	fmt.Printf("\n  Konfiguracja:\n")
	fmt.Printf("    Rozdzielczość : %dx%d @ %dHz\n", cfg.Width, cfg.Height, cfg.RefreshRate)
	fmt.Printf("    Fullscreen    : %v\n", cfg.Fullscreen)
	fmt.Printf("    VSync         : %v\n", cfg.VSync)
	fmt.Printf("    HDR           : %v\n", cfg.HDR)
	fmt.Printf("    MangoApp      : %v\n", cfg.MangoApp)
	fmt.Printf("    Auto Steam    : %v\n", cfg.AutoSteam)
}

// buildGamescopeArgs buduje argumenty dla procesu gamescope.
func buildGamescopeArgs(cfg Config) []string {
	var args []string

	// Rozdzielczość wyświetlana (wirtualna)
	args = append(args, "-w", strconv.Itoa(cfg.Width))
	args = append(args, "-h", strconv.Itoa(cfg.Height))
	// Rozdzielczość wyjściowa (natywna monitora — domyślnie ta sama)
	args = append(args, "-W", strconv.Itoa(cfg.Width))
	args = append(args, "-H", strconv.Itoa(cfg.Height))
	// Refresh rate
	args = append(args, "-r", strconv.Itoa(cfg.RefreshRate))

	if cfg.Fullscreen {
		args = append(args, "-f")
	}

	if cfg.VSync {
		args = append(args, "--adaptive-sync")
	}

	if cfg.HDR {
		args = append(args, "--hdr-enabled")
	}

	if cfg.MangoApp {
		// Overlay z FPS przez mangoapp
		args = append(args, "--mangoapp")
	}

	// Nested mode — gamescope uruchomiony na Wayland/X11
	// Na TTY (bez serwera wyświetlania) — tryb standalone
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		args = append(args, "--nested")
	} else if os.Getenv("DISPLAY") != "" {
		args = append(args, "--nested")
	}
	// Jeśli żaden — tryb standalone na KMS/TTY (domyślny dla game mode)

	// Dodatkowe flagi
	args = append(args, cfg.ExtraFlags...)

	return args
}

// buildEnvironment buduje zmienne środowiskowe dla sesji gamescope.
func buildEnvironment(cfg Config) []string {
	env := os.Environ()

	// Wayland
	env = appendEnv(env, "XDG_SESSION_TYPE", "wayland")
	env = appendEnv(env, "XDG_CURRENT_DESKTOP", "gamescope")
	env = appendEnv(env, "XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%d", os.Getuid()))

	// Steam — wymuś tryb Big Picture / GamepadUI
	env = appendEnv(env, "STEAM_USE_DYNAMIC_VRS", "1")
	env = appendEnv(env, "STEAM_GAMESCOPE_COLOR_MANAGED", "1")
	env = appendEnv(env, "STEAM_GAMESCOPE_TEARING_SUPPORTED", "1")
	env = appendEnv(env, "GAMESCOPE_WAYLAND_DISPLAY", "gamescope-0")

	// Mesa / Vulkan
	env = appendEnv(env, "MESA_VK_WSI_PRESENT_MODE", "mailbox")

	// Proton / Wine — wymuś Proton dla gier Windows
	env = appendEnv(env, "STEAM_COMPAT_CLIENT_INSTALL_PATH", steamInstallPath())

	// HDR
	if cfg.HDR {
		env = appendEnv(env, "DXVK_HDR", "1")
	}

	// MangoHud
	if cfg.MangoApp {
		env = appendEnv(env, "MANGOHUD", "1")
	}

	return env
}

// handleConfig obsługuje subkomendy konfiguracyjne.
func handleConfig(args []string) {
	if len(args) == 0 {
		cfg := loadConfig()
		printConfig(cfg)
		return
	}

	switch strings.ToLower(args[0]) {
	case "show", "":
		cfg := loadConfig()
		printConfig(cfg)

	case "set":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "[gamescope-manager] Użycie: config set <klucz> <wartość>")
			os.Exit(1)
		}
		cfg := loadConfig()
		if err := setConfigKey(&cfg, args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd: %v\n", err)
			os.Exit(1)
		}
		if err := saveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd zapisu konfiguracji: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("[gamescope-manager] Ustawiono: %s = %s\n", args[1], args[2])

	case "reset":
		cfg := defaultConfig()
		if err := saveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Błąd: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("[gamescope-manager] Konfiguracja przywrócona do domyślnej.")

	default:
		fmt.Fprintf(os.Stderr, "[gamescope-manager] Nieznana subkomenda config: %q\n", args[0])
		os.Exit(1)
	}
}

func printConfig(cfg Config) {
	fmt.Println("[gamescope-manager] Aktualna konfiguracja:")
	fmt.Printf("  width       = %d\n", cfg.Width)
	fmt.Printf("  height      = %d\n", cfg.Height)
	fmt.Printf("  refresh     = %d\n", cfg.RefreshRate)
	fmt.Printf("  fullscreen  = %v\n", cfg.Fullscreen)
	fmt.Printf("  vsync       = %v\n", cfg.VSync)
	fmt.Printf("  hdr         = %v\n", cfg.HDR)
	fmt.Printf("  mangoapp    = %v\n", cfg.MangoApp)
	fmt.Printf("  auto_steam  = %v\n", cfg.AutoSteam)
	if len(cfg.ExtraFlags) > 0 {
		fmt.Printf("  extra_flags = %s\n", strings.Join(cfg.ExtraFlags, " "))
	}
	fmt.Printf("\n  (Plik konfiguracyjny: %s)\n", configFile)
}

// parseStartFlags parsuje flagi dla komendy 'start'.
func parseStartFlags(cfg *Config, args []string) {
	i := 0
	for i < len(args) {
		switch args[i] {
		case "--steam":
			cfg.AutoSteam = true
		case "--no-steam":
			cfg.AutoSteam = false
		case "--width":
			if i+1 < len(args) {
				if v, err := strconv.Atoi(args[i+1]); err == nil {
					cfg.Width = v
				}
				i++
			}
		case "--height":
			if i+1 < len(args) {
				if v, err := strconv.Atoi(args[i+1]); err == nil {
					cfg.Height = v
				}
				i++
			}
		case "--refresh":
			if i+1 < len(args) {
				if v, err := strconv.Atoi(args[i+1]); err == nil {
					cfg.RefreshRate = v
				}
				i++
			}
		case "--hdr":
			cfg.HDR = true
		case "--no-mangoapp":
			cfg.MangoApp = false
		case "--no-vsync":
			cfg.VSync = false
		case "--no-fullscreen":
			cfg.Fullscreen = false
		case "--extra":
			if i+1 < len(args) {
				cfg.ExtraFlags = append(cfg.ExtraFlags, strings.Fields(args[i+1])...)
				i++
			}
		}
		i++
	}
}

// loadConfig wczytuje konfigurację z pliku lub zwraca domyślną.
func loadConfig() Config {
	cfg := defaultConfig()
	data, err := os.ReadFile(configFile)
	if err != nil {
		return cfg
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		_ = setConfigKey(&cfg, key, val)
	}
	return cfg
}

// saveConfig zapisuje konfigurację do pliku.
func saveConfig(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return err
	}
	lines := []string{
		"# HackerOS Gamescope Manager — plik konfiguracyjny",
		"# Edytuj ręcznie lub użyj: gamescope-manager config set <klucz> <wartość>",
		"",
		fmt.Sprintf("width       = %d", cfg.Width),
		fmt.Sprintf("height      = %d", cfg.Height),
		fmt.Sprintf("refresh     = %d", cfg.RefreshRate),
		fmt.Sprintf("fullscreen  = %v", cfg.Fullscreen),
		fmt.Sprintf("vsync       = %v", cfg.VSync),
		fmt.Sprintf("hdr         = %v", cfg.HDR),
		fmt.Sprintf("mangoapp    = %v", cfg.MangoApp),
		fmt.Sprintf("auto_steam  = %v", cfg.AutoSteam),
		fmt.Sprintf("extra_flags = %s", strings.Join(cfg.ExtraFlags, " ")),
	}
	return os.WriteFile(configFile, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// setConfigKey ustawia konkretny klucz konfiguracyjny.
func setConfigKey(cfg *Config, key, val string) error {
	switch strings.ToLower(key) {
	case "width":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("width musi być liczbą całkowitą")
		}
		cfg.Width = v
	case "height":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("height musi być liczbą całkowitą")
		}
		cfg.Height = v
	case "refresh":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("refresh musi być liczbą całkowitą")
		}
		cfg.RefreshRate = v
	case "fullscreen":
		cfg.Fullscreen = parseBool(val)
	case "vsync":
		cfg.VSync = parseBool(val)
	case "hdr":
		cfg.HDR = parseBool(val)
	case "mangoapp":
		cfg.MangoApp = parseBool(val)
	case "auto_steam":
		cfg.AutoSteam = parseBool(val)
	case "extra_flags":
		cfg.ExtraFlags = strings.Fields(val)
	default:
		return fmt.Errorf("nieznany klucz konfiguracji: %q\nDostępne: width, height, refresh, fullscreen, vsync, hdr, mangoapp, auto_steam, extra_flags", key)
	}
	return nil
}

// Helpers

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

func isRunning() bool {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	// Sprawdź czy proces istnieje
	if err := syscall.Kill(pid, 0); err != nil {
		_ = os.Remove(pidFile)
		return false
	}
	return true
}

func readPID() string {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writePID(pid int) error {
	dir := filepath.Dir(pidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(pidFile, []byte(strconv.Itoa(pid)+"\n"), 0644)
}

func isProcessRunning(name string) bool {
	cmd := exec.Command("pgrep", "-x", name)
	return cmd.Run() == nil
}

func findExecutable(name string) (string, error) {
	// Typowe lokalizacje gamescope
	candidates := []string{
		"/usr/bin/" + name,
		"/usr/local/bin/" + name,
		"/usr/games/" + name,
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return exec.LookPath(name)
}

func findSteam() (string, error) {
	candidates := []string{
		"/usr/bin/steam",
		"/usr/games/steam",
		"/usr/local/bin/steam",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return exec.LookPath("steam")
}

func steamInstallPath() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local/share/Steam"),
		"/usr/share/steam",
		"/opt/steam",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return filepath.Join(home, ".local/share/Steam")
}

func appendEnv(env []string, key, value string) []string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return env // już ustawione — nie nadpisuj
		}
	}
	return append(env, prefix+value)
}

func ensureDirs() {
	dirs := []string{
		"/var/lib/hackeros-gaming",
		"/var/log/hackeros-gaming",
		"/etc/hackeros",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			// Ignoruj — może brakować uprawnień, ale to nie jest błąd krytyczny
		}
	}
}

func logEntry(msg string) {
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] %s\n", ts, msg)
}

func handleSignals(cmd *exec.Cmd) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-sigCh
	fmt.Printf("\n[gamescope-manager] Otrzymano sygnał %v, zamykam sesję...\n", sig)
	if cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
}
