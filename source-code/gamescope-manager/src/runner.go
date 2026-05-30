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

const (
	// Ile razy z rzędu sesja może crashować zanim wrócimy do pulpitu
	maxCrashCount = 5
	// Jeśli sesja żyje krócej niż to, liczymy jako crash
	shortSessionSecs = 60
)

// checkRoot sprawdza uprawnienia roota.
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
// Jeśli cfg.Width/Height == 0, auto-wykrywa rozdzielczość monitora.
func Start(cfg Config) error {
	if err := checkRoot(); err != nil {
		return err
	}

	// --- Health checks ---
	results, canStart := RunHealthChecks(cfg)
	for _, r := range results {
		if !r.OK {
			if r.Fatal {
				fmt.Fprintf(os.Stderr, "[!!] %s: %s\n", r.Name, r.Message)
			} else {
				fmt.Fprintf(os.Stderr, "[??] %s: %s\n", r.Name, r.Message)
			}
		}
	}
	if !canStart {
		return fmt.Errorf("preflight checks nie powiodly sie — nie uruchamiam sesji")
	}

	gsPath, err := findBinary("gamescope")
	if err != nil {
		return fmt.Errorf("gamescope nie jest zainstalowany\nZainstaluj: sudo apt install gamescope")
	}
	if IsRunning() {
		return fmt.Errorf("sesja gamescope juz dziala (PID: %s)\nUzyj 'restart' aby zrestartowac", ReadPID())
	}

	// --- Auto-detekcja rozdzielczości ---
	w, h, refresh, displayInfo := ResolveResolution(cfg)
	fmt.Printf("[gamescope-manager] Rozdzielczosc: %s\n", displayInfo.FormatResolution())
	LogEntry(fmt.Sprintf("DISPLAY: %s", displayInfo.FormatResolution()))

	// Zaktualizuj cfg o wykryte wartości (tylko na czas tego uruchomienia)
	runCfg := cfg
	runCfg.Width = w
	runCfg.Height = h
	runCfg.RefreshRate = refresh

	// --- Buduj argumenty ---
	args := buildArgs(runCfg, displayInfo)

	var steamPath string
	if cfg.AutoSteam {
		steamPath, err = findSteam()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Ostrzezenie: Steam nie znaleziony: %v\n", err)
			runCfg.AutoSteam = false
		}
	}

	if runCfg.AutoSteam && steamPath != "" {
		steamArgs := buildSteamArgs(runCfg)
		args = append(args, "--")
		args = append(args, steamPath)
		args = append(args, steamArgs...)
	}

	env := buildEnv(runCfg)

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

	start := time.Now()
	waitErr := cmd.Wait()
	elapsed := time.Since(start)
	RemovePID()

	if waitErr != nil {
		LogEntry(fmt.Sprintf("Sesja zakonczona z bledem po %v: %v", elapsed.Round(time.Second), waitErr))
		fmt.Printf("[gamescope-manager] Sesja zakonczona po %v: %v\n", elapsed.Round(time.Second), waitErr)
	} else {
		LogEntry(fmt.Sprintf("Sesja zakonczona normalnie po %v", elapsed.Round(time.Second)))
		fmt.Println("[gamescope-manager] Sesja gamescope zakonczona.")
	}

	// Crash detection — zapisz do crash trackera
	if elapsed < shortSessionSecs*time.Second {
		trackCrash()
	} else {
		resetCrashTracker()
	}

	return waitErr
}

// StartWithRecovery uruchamia sesję z automatycznym restartem po crashu.
// Używane przez hackeros-game-mode.service (Restart=on-failure).
// Po maxCrashCount crashów z rzędu — wraca do pulpitu.
func StartWithRecovery(cfg Config) error {
	for {
		count := readCrashCount()
		if count >= maxCrashCount {
			resetCrashTracker()
			LogEntry(fmt.Sprintf("CRASH LOOP: %d crashow — powrot do pulpitu", count))
			fmt.Fprintf(os.Stderr,
				    "[gamescope-manager] Za duzo crashow (%d) — powrot do pulpitu\n", count)
			// Przełącz na tty2 (KDE)
			_ = exec.Command("chvt", "2").Run()
			return fmt.Errorf("sesja crashowala %d razy z rzedu", count)
		}

		err := Start(cfg)
		if err == nil {
			// Normalne zakończenie (użytkownik wyszedł)
			return nil
		}

		count = readCrashCount()
		fmt.Printf("[gamescope-manager] Crash #%d. Restart za 3 sekundy...\n", count)
		LogEntry(fmt.Sprintf("RESTART po crashu #%d", count))
		time.Sleep(3 * time.Second)
	}
}

// Stop zatrzymuje aktywną sesję gamescope z timeoutem 30s.
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

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		_ = proc.Signal(syscall.SIGKILL)
		RemovePID()
		return nil
	}

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

// ShowStatus wyświetla status sesji i konfigurację.
func ShowStatus() {
	fmt.Println("[gamescope-manager] Status sesji:")
	fmt.Println()
	if IsRunning() {
		fmt.Printf("  Status        : AKTYWNA\n")
		fmt.Printf("  PID           : %s\n", ReadPID())
		fmt.Printf("  Gamescope     : %s\n", procStatus("gamescope"))
		fmt.Printf("  Steam         : %s\n", procStatus("steam"))
		fmt.Printf("  MangoApp      : %s\n", procStatus("mangoapp"))
		fmt.Printf("  Crashe        : %d / %d\n", readCrashCount(), maxCrashCount)
	} else {
		fmt.Println("  Status        : NIEAKTYWNA")
	}
	fmt.Println()

	cfg := Load()
	w, h, refresh, dispInfo := ResolveResolution(cfg)
	fmt.Printf("  Konfiguracja  : %s\n", ConfigFile)
	if w == 0 {
		fmt.Printf("  Rozdzielczosc : auto (DRM) @%dHz — %s\n", refresh, dispInfo.Source)
	} else {
		fmt.Printf("  Rozdzielczosc : %dx%d@%dHz [%s]\n", w, h, refresh, dispInfo.Source)
	}
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

// ── Budowanie argumentów ───────────────────────────────────────────────────────

// buildArgs buduje argumenty dla gamescope.
// Kluczowe: jeśli w/h == 0, NIE przekazujemy -w/-h/-W/-H (gamescope sam wykryje z DRM).
func buildArgs(cfg Config, display DisplayInfo) []string {
	var args []string

	// Tryb embedded (-e) + --steam — jak w gamescope-session-steam (SteamOS/ChimeraOS)
	// W embedded mode gamescope przejmuje KMS/DRM bezpośrednio, bez X/Wayland hosta
	isEmbedded := os.Getenv("WAYLAND_DISPLAY") == "" && os.Getenv("DISPLAY") == ""
	if isEmbedded {
		args = append(args, "-e") // embedded mode — przejmuje DRM bezpośrednio
	} else {
		args = append(args, "--nested") // nested mode — wewnątrz istniejącego Wayland/X11
	}

	// Rozdzielczość — tylko jeśli auto-detekcja dała wynik lub użytkownik ustawił
	if cfg.Width > 0 && cfg.Height > 0 {
		args = append(args, "-w", strconv.Itoa(cfg.Width))
		args = append(args, "-h", strconv.Itoa(cfg.Height))
		args = append(args, "-W", strconv.Itoa(cfg.Width))
		args = append(args, "-H", strconv.Itoa(cfg.Height))
	}
	// Jeśli Width==0 — pomijamy -w/-h/-W/-H i gamescope wykryje z DRM (jak ChimeraOS)

	// Refresh rate
	if cfg.RefreshRate > 0 {
		args = append(args, "-r", strconv.Itoa(cfg.RefreshRate))
	}

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

	// Preferowany output connector (np. "HDMI-A-1") — jak w gamescope-session
	if display.Connector != "" {
		args = append(args, "--prefer-output", display.Connector)
	}

	// Gniazdo Wayland gamescope (dla komunikacji Steam ↔ gamescope)
	socket := gamescopeSocket()
	stats := gamescopeStats()
	args = append(args, "-R", socket)
	args = append(args, "-T", stats)

	// --steam flag dla gamescope — informuje gamescope że to sesja Steam
	// (gamescope ma specjalne ścieżki kodu dla Steam)
	if cfg.AutoSteam {
		args = append(args, "--steam")
	}

	// Xwayland count — Steam potrzebuje co najmniej 2
	args = append(args, "--xwayland-count", "2")

	// Cursor
	args = append(args, "--hide-cursor-delay", "3000")
	args = append(args, "--fade-out-duration", "200")

	// Dodatkowe flagi użytkownika
	args = append(args, cfg.ExtraFlags...)

	return args
}

// buildSteamArgs buduje argumenty przekazywane do steam.
func buildSteamArgs(cfg Config) []string {
	var args []string

	// Kluczowe flagi jak w gamescope-session-steam (SteamOS/ChimeraOS)
	args = append(args, "-steamos3")   // tryb SteamOS 3 UI
	args = append(args, "-pipewire-dmabuf") // PipeWire DMA-BUF dla zero-copy video

	if cfg.SteamBigPicture {
		args = append(args, "-gamepadui") // nowy Steam Deck/BPM UI
	}

	if cfg.SteamVR {
		args = append(args, "-vr")
	}

	if cfg.SteamNoCEFSandbox {
		args = append(args, "-no-cef-sandbox")
	}

	if cfg.SteamTCP {
		args = append(args, "-nosharedmemory")
	}

	if cfg.SteamNoVerify {
		args = append(args, "-noverifyfiles")
	}

	if cfg.SteamFullDesktop {
		args = append(args, "-fulldesktopres")
	}

	if cfg.SteamAllowHidCrypt {
		args = append(args, "-allow-hidcrypto")
	}

	if cfg.SteamDisableHWA {
		args = append(args, "-disable-gpu")
	}

	switch cfg.SteamBetaChannel {
		case "beta":
			args = append(args, "-beta", "beta")
		case "steamdeckbeta":
			args = append(args, "-beta", "steamdeckbeta")
		case "beta-client":
			args = append(args, "-beta", "beta-client")
	}

	if cfg.SteamLanguage != "" {
		args = append(args, "-lang", cfg.SteamLanguage)
	}

	if cfg.SteamExtraFlags != "" {
		args = append(args, strings.Fields(cfg.SteamExtraFlags)...)
	}

	return args
}

// buildEnv buduje środowisko dla gamescope, wzorowane na gamescope-session-steam.
func buildEnv(cfg Config) []string {
	env := os.Environ()

	uid := os.Getuid()
	runtimeDir := fmt.Sprintf("/run/user/%d", uid)

	// Podstawowe zmienne sesji
	env = appendMissing(env, "XDG_SESSION_TYPE", "wayland")
	env = appendMissing(env, "XDG_CURRENT_DESKTOP", "gamescope")
	env = appendMissing(env, "XDG_SESSION_CLASS", "user")
	env = appendMissing(env, "XDG_SESSION_DESKTOP", "gamescope")
	env = appendMissing(env, "XDG_RUNTIME_DIR", runtimeDir)

	// Gniazda gamescope
	socket := gamescopeSocket()
	env = appendMissing(env, "GAMESCOPE_WAYLAND_DISPLAY", socket)
	env = appendMissing(env, "WAYLAND_DISPLAY", socket)

	// Steam + gamescope integracja (jak w SteamOS / Bazzite)
	env = appendMissing(env, "STEAM_USE_DYNAMIC_VRS", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_COLOR_MANAGED", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_TEARING_SUPPORTED", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_COMPOSITE_DEBUG", "0")
	env = appendMissing(env, "STEAM_GAMESCOPE_FANCY_SCALING_SUPPORT", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_DYNAMIC_FPSLIMIT", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_NIS_SUPPORTED", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_FRAMEOUT_INHIBIT_SUPPORT", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_PIPELINE_CACHE", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_VRMODE_SUPPORT", boolToStr(cfg.SteamVR))
	env = appendMissing(env, "STEAM_GAMESCOPE_HDR_SUPPORTED", boolToStr(cfg.HDR))
	env = appendMissing(env, "STEAM_GAMESCOPE_VRR_SUPPORTED", boolToStr(cfg.VSync))
	env = appendMissing(env, "DBUS_SESSION_BUS_ADDRESS",
			    fmt.Sprintf("unix:path=%s/bus", runtimeDir))

	// Mesa / Vulkan
	env = appendMissing(env, "MESA_VK_WSI_PRESENT_MODE", "mailbox")
	env = appendMissing(env, "STEAM_COMPAT_CLIENT_INSTALL_PATH", steamDir())
	env = appendMissing(env, "STEAM_COMPAT_DATA_PATH", steamCompatPath())

	// Proton / Wine
	env = appendMissing(env, "PROTON_LOG", "0")

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

// ── Crash tracking ─────────────────────────────────────────────────────────────

const crashTrackerFile = "/var/lib/hackeros-gaming/crash-tracker"

func trackCrash() {
	count := readCrashCount() + 1
	_ = os.WriteFile(crashTrackerFile, []byte(strconv.Itoa(count)), 0644)
	LogEntry(fmt.Sprintf("CRASH #%d (krotka sesja)", count))
}

func resetCrashTracker() {
	_ = os.Remove(crashTrackerFile)
}

func readCrashCount() int {
	data, err := os.ReadFile(crashTrackerFile)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return n
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func gamescopeSocket() string {
	return fmt.Sprintf("/run/user/%d/gamescope-0", os.Getuid())
}

func gamescopeStats() string {
	return fmt.Sprintf("/tmp/gamescope-stats-%d", os.Getpid())
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
	for _, p := range []string{
		"/usr/bin/" + name,
		"/usr/local/bin/" + name,
		"/usr/games/" + name,
	} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return exec.LookPath(name)
}

func findSteam() (string, error) {
	for _, p := range []string{
		"/usr/bin/steam",
		"/usr/games/steam",
		"/usr/local/bin/steam",
	} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return exec.LookPath("steam")
}

func isSteamRunning() bool {
	return exec.Command("pgrep", "-x", "steam").Run() == nil
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
