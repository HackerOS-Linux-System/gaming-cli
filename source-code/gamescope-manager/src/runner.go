package src

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

// checkRoot sprawdza czy proces działa z uprawnieniami roota.
// gamescope-manager musi być root żeby zapisywać PID i logi.
func checkRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf(
			"gamescope-manager wymaga uprawnien roota\n" +
			"Uzyj: sudo gamescope-manager",
		)
	}
	return nil
}

// Start uruchamia sesję gamescope + opcjonalnie Steam BPM.
func Start(cfg Config) error {
	if err := checkRoot(); err != nil {
		return err
	}

	gsPath, err := findBinary("gamescope")
	if err != nil {
		return fmt.Errorf("gamescope nie jest zainstalowany\nZainstaluj: sudo apt install gamescope")
	}
	if IsRunning() {
		return fmt.Errorf("sesja gamescope juz dziala (PID: %s)\nUzyj 'restart' aby zrestartowac", ReadPID())
	}

	args := buildArgs(cfg)

	var steamPath string
	if cfg.AutoSteam {
		steamPath, err = findSteam()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Ostrzezenie: Steam nie znaleziony: %v\n", err)
			cfg.AutoSteam = false
		}
	}

	if cfg.AutoSteam && steamPath != "" {
		steamArgs := buildSteamArgs(cfg)
		args = append(args, "--")
		args = append(args, steamPath)
		args = append(args, steamArgs...)
	}

	env := buildEnv(cfg)

	LogEntry(fmt.Sprintf("START: %s %s", gsPath, strings.Join(args, " ")))
	fmt.Printf("[gamescope-manager] Uruchamiam: gamescope %s\n", strings.Join(args, " "))

	cmd := exec.Command(gsPath, args...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("nie mozna uruchomiec gamescope: %w", err)
	}

	pid := cmd.Process.Pid
	fmt.Printf("[gamescope-manager] Sesja uruchomiona (PID: %d)\n", pid)
	LogEntry(fmt.Sprintf("PID=%d", pid))

	if err := WritePID(pid); err != nil {
		fmt.Fprintf(os.Stderr, "[gamescope-manager] Ostrzezenie: nie mozna zapisac PID: %v\n", err)
	}

	go watchSignals(cmd)

	waitErr := cmd.Wait()
	RemovePID()

	if waitErr != nil {
		LogEntry(fmt.Sprintf("Sesja zakonczona z bledem: %v", waitErr))
		fmt.Printf("[gamescope-manager] Sesja zakonczona: %v\n", waitErr)
	} else {
		LogEntry("Sesja zakonczona normalnie")
		fmt.Println("[gamescope-manager] Sesja gamescope zakonczona.")
	}
	return nil
}

// Stop zatrzymuje aktywną sesję gamescope z rozsądnym timeoutem.
func Stop() error {
	if err := checkRoot(); err != nil {
		return err
	}

	if !IsRunning() {
		fmt.Println("[gamescope-manager] Brak aktywnej sesji.")
		return nil
	}
	pid := GetPIDInt()
	if pid == -1 {
		return fmt.Errorf("nie mozna odczytac PID sesji")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("nie mozna znalezc procesu PID %d: %w", pid, err)
	}
	fmt.Printf("[gamescope-manager] Zatrzymuje sesje (PID: %d)...\n", pid)
	LogEntry(fmt.Sprintf("STOP: PID=%d", pid))

	// SIGTERM — poproś o zamknięcie
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Proces może już nie żyć — spróbuj SIGKILL od razu
		_ = proc.Signal(syscall.SIGKILL)
		RemovePID()
		return nil
	}

	// Timeout 30 sekund: pierwsze 20 sek. czekamy, po 20 sek. SIGKILL
	const (
		checkInterval = time.Second
		softTimeout   = 20 * time.Second
		hardTimeout   = 30 * time.Second
	)
	start := time.Now()
	for {
		time.Sleep(checkInterval)
		if !IsRunning() {
			break
		}
		elapsed := time.Since(start)
		if elapsed >= softTimeout && elapsed < hardTimeout {
			fmt.Printf("[gamescope-manager] Proces nie odpowiada (%ds), wysylam SIGKILL...\n",
				   int(elapsed.Seconds()))
			_ = proc.Signal(syscall.SIGKILL)
		} else if elapsed >= hardTimeout {
			fmt.Println("[gamescope-manager] Hard timeout — wymuszam zatrzymanie.")
			_ = proc.Signal(syscall.SIGKILL)
			break
		}
	}

	RemovePID()
	LogEntry("Sesja zatrzymana")
	fmt.Println("[gamescope-manager] Sesja zatrzymana.")
	return nil
}

// ShowStatus wyswietla status sesji i konfiguracje.
func ShowStatus() {
	fmt.Println("[gamescope-manager] Status sesji:")
	fmt.Println()
	if IsRunning() {
		fmt.Printf("  Status      : AKTYWNA\n")
		fmt.Printf("  PID         : %s\n", ReadPID())
		fmt.Printf("  Gamescope   : %s\n", procStatus("gamescope"))
		fmt.Printf("  Steam       : %s\n", procStatus("steam"))
		fmt.Printf("  MangoApp    : %s\n", procStatus("mangoapp"))
	} else {
		fmt.Println("  Status      : NIEAKTYWNA")
	}
	fmt.Println()
	cfg := Load()
	fmt.Printf("  Konfiguracja (.hk): %s\n", ConfigFile)
	fmt.Printf("  Rozdzielczosc : %dx%d @ %d Hz\n", cfg.Width, cfg.Height, cfg.RefreshRate)
	fmt.Printf("  Fullscreen    : %v\n", cfg.Fullscreen)
	fmt.Printf("  VSync         : %v\n", cfg.VSync)
	fmt.Printf("  HDR           : %v\n", cfg.HDR)
	fmt.Printf("  MangoApp      : %v\n", cfg.MangoApp)
	fmt.Printf("  Auto Steam    : %v\n", cfg.AutoSteam)
	fmt.Printf("  Big Picture   : %v\n", cfg.SteamBigPicture)
	if len(cfg.ExtraFlags) > 0 {
		fmt.Printf("  Extra flags   : %s\n", strings.Join(cfg.ExtraFlags, " "))
	}
	fmt.Println()
}

// buildArgs buduje argumenty dla gamescope.
func buildArgs(cfg Config) []string {
	var args []string
	args = append(args, "-w", strconv.Itoa(cfg.Width))
	args = append(args, "-h", strconv.Itoa(cfg.Height))
	args = append(args, "-W", strconv.Itoa(cfg.Width))
	args = append(args, "-H", strconv.Itoa(cfg.Height))
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
		args = append(args, "--mangoapp")
	}
	if cfg.ForceComposite {
		args = append(args, "--force-composition")
	}
	// Tryb zagnieżdżony tylko jeśli uruchamiamy w środowisku graficznym
	if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("DISPLAY") != "" {
		args = append(args, "--nested")
	}
	args = append(args, cfg.ExtraFlags...)
	return args
}

// buildSteamArgs buduje argumenty przekazywane do steam,
// zgodnie z tym co robi gamescope-session-steam (SteamOS).
func buildSteamArgs(cfg Config) []string {
	var args []string

	// Tryb Big Picture (Steam Deck UI / BPM)
	if cfg.SteamBigPicture {
		args = append(args, "-tenfoot")
	}

	// Tryb VR
	if cfg.SteamVR {
		args = append(args, "-vr")
	}

	// Wyłącz sandbox CEF — przyspiesza uruchamianie, zalecane dla gamescope
	if cfg.SteamNoCEFSandbox {
		args = append(args, "-no-cef-sandbox")
	}

	// TCP fallback (przydatne gdy masz problemy z gniazdem UNIX)
	if cfg.SteamTCP {
		args = append(args, "-nosharedmemory")
	}

	// Nie weryfikuj plików gry przy starcie
	if cfg.SteamNoVerify {
		args = append(args, "-noverifyfiles")
	}

	// Pełna rozdzielczość ekranu w trybie pulpitu
	if cfg.SteamFullDesktop {
		args = append(args, "-fulldesktopres")
	}

	// HID crypto (urządzenia wejścia)
	if cfg.SteamAllowHidCrypt {
		args = append(args, "-allow-hidcrypto")
	}

	// Wyłącz hardware acceleration w UI Steam
	if cfg.SteamDisableHWA {
		args = append(args, "-disable-gpu")
	}

	// Kanał beta
	switch cfg.SteamBetaChannel {
		case "beta":
			args = append(args, "-beta", "beta")
		case "steamdeckbeta":
			args = append(args, "-beta", "steamdeckbeta")
		case "beta-client":
			args = append(args, "-beta", "beta-client")
	}

	// Język
	if cfg.SteamLanguage != "" {
		args = append(args, "-lang", cfg.SteamLanguage)
	}

	// Dodatkowe flagi użytkownika
	if cfg.SteamExtraFlags != "" {
		args = append(args, strings.Fields(cfg.SteamExtraFlags)...)
	}

	return args
}

// buildEnv buduje środowisko dla gamescope, wzorowane na gamescope-session-steam (SteamOS).
func buildEnv(cfg Config) []string {
	env := os.Environ()

	// Podstawowe
	env = appendMissing(env, "XDG_SESSION_TYPE", "wayland")
	env = appendMissing(env, "XDG_CURRENT_DESKTOP", "gamescope")
	env = appendMissing(env, "XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%d", os.Getuid()))
	env = appendMissing(env, "XDG_SESSION_CLASS", "user")
	env = appendMissing(env, "XDG_SESSION_DESKTOP", "gamescope")

	// Steam + gamescope integracja (jak w gamescope-session-steam / SteamOS)
	env = appendMissing(env, "STEAM_USE_DYNAMIC_VRS", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_COLOR_MANAGED", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_TEARING_SUPPORTED", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_COMPOSITE_DEBUG", "0")
	env = appendMissing(env, "STEAM_GAMESCOPE_FANCY_SCALING_SUPPORT", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_HDR_SUPPORTED", boolToStr(cfg.HDR))
	env = appendMissing(env, "STEAM_GAMESCOPE_DYNAMIC_FPSLIMIT", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_NIS_SUPPORTED", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_FRAMEOUT_INHIBIT_SUPPORT", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_VRR_SUPPORTED", boolToStr(cfg.VSync))
	env = appendMissing(env, "GAMESCOPE_WAYLAND_DISPLAY", "gamescope-0")
	env = appendMissing(env, "MESA_VK_WSI_PRESENT_MODE", "mailbox")
	env = appendMissing(env, "STEAM_COMPAT_CLIENT_INSTALL_PATH", steamDir())
	env = appendMissing(env, "STEAM_COMPAT_DATA_PATH", steamCompatPath())

	// Dbus (Steam potrzebuje działającego dbus w sesji)
	env = appendMissing(env, "DBUS_SESSION_BUS_ADDRESS", fmt.Sprintf("unix:path=/run/user/%d/bus", os.Getuid()))

	// Wayland socket dla zagnieżdżonego gamescope
	env = appendMissing(env, "WAYLAND_DISPLAY", "gamescope-0")

	if cfg.HDR {
		env = appendMissing(env, "DXVK_HDR", "1")
	}
	if cfg.MangoApp {
		env = appendMissing(env, "MANGOHUD", "1")
	}
	return env
}

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func watchSignals(cmd *exec.Cmd) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-ch
	fmt.Printf("\n[gamescope-manager] Sygnal %v — zamykam sesje...\n", sig)
	if cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
}

func findBinary(name string) (string, error) {
	for _, p := range []string{"/usr/bin/" + name, "/usr/local/bin/" + name, "/usr/games/" + name} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return exec.LookPath(name)
}

func findSteam() (string, error) {
	for _, p := range []string{"/usr/bin/steam", "/usr/games/steam", "/usr/local/bin/steam"} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return exec.LookPath("steam")
}

// isSteamRunning sprawdza czy Steam już działa jako osobny proces.
// Jeśli tak — nie uruchamiamy nowej instancji, tylko przekazujemy flagi przez pipe/socket.
func isSteamRunning() bool {
	return exec.Command("pgrep", "-x", "steam").Run() == nil ||
	exec.Command("pgrep", "-x", "steam_osx").Run() == nil
}

func steamDir() string {
	home, _ := os.UserHomeDir()
	for _, p := range []string{
		filepath.Join(home, ".local/share/Steam"),
		"/usr/share/steam",
		"/opt/steam",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(home, ".local/share/Steam")
}

func steamCompatPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local/share/Steam/steamapps/compatdata")
}

func appendMissing(env []string, key, value string) []string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return env
		}
	}
	return append(env, prefix+value)
}

func procStatus(name string) string {
	if exec.Command("pgrep", "-x", name).Run() == nil {
		return "uruchomiony"
	}
	return "nie uruchomiony"
}
