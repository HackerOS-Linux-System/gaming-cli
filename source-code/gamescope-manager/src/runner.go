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

// Start uruchamia sesję gamescope zgodnie z podaną konfiguracją.
func Start(cfg Config) error {
	// Sprawdź czy gamescope jest zainstalowany
	gsPath, err := findBinary("gamescope")
	if err != nil {
		return fmt.Errorf(
			"gamescope nie jest zainstalowany lub niedostępny w PATH\n"+
				"Zainstaluj: sudo apt install gamescope",
		)
	}

	// Sprawdź czy sesja już działa
	if IsRunning() {
		return fmt.Errorf(
			"sesja gamescope już działa (PID: %s)\n"+
				"Użyj 'gamescope-manager restart' aby zrestartować",
			ReadPID(),
		)
	}

	// Zbuduj argumenty
	args := buildArgs(cfg)

	// Znajdź Steam jeśli potrzebny
	var steamPath string
	if cfg.AutoSteam {
		steamPath, err = findSteam()
		if err != nil {
			fmt.Fprintf(os.Stderr,
				"[gamescope-manager] Ostrzeżenie: Steam nie znaleziony (%v)\n"+
					"[gamescope-manager] Uruchamiam gamescope bez Steam.\n",
				err,
			)
			cfg.AutoSteam = false
		}
	}

	// Dołącz Steam do argumentów
	if cfg.AutoSteam && steamPath != "" {
		args = append(args, "--")
		args = append(args, steamPath, "-tenfoot")
	}

	// Środowisko
	env := buildEnv(cfg)

	LogEntry(fmt.Sprintf("START: %s %s", gsPath, strings.Join(args, " ")))
	fmt.Printf("[gamescope-manager] Uruchamiam: gamescope %s\n", strings.Join(args, " "))

	cmd := exec.Command(gsPath, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("nie można uruchomić gamescope: %w", err)
	}

	pid := cmd.Process.Pid
	fmt.Printf("[gamescope-manager] Sesja uruchomiona (PID: %d)\n", pid)
	LogEntry(fmt.Sprintf("Sesja uruchomiona PID=%d", pid))

	if err := WritePID(pid); err != nil {
		fmt.Fprintf(os.Stderr, "[gamescope-manager] Ostrzeżenie: nie można zapisać PID: %v\n", err)
	}

	// Obsługa sygnałów — czysty shutdown
	go watchSignals(cmd)

	// Czekaj na zakończenie
	waitErr := cmd.Wait()
	RemovePID()

	if waitErr != nil {
		LogEntry(fmt.Sprintf("Sesja zakończona z błędem: %v", waitErr))
		fmt.Printf("[gamescope-manager] Sesja zakończona: %v\n", waitErr)
	} else {
		LogEntry("Sesja zakończona normalnie")
		fmt.Println("[gamescope-manager] Sesja gamescope zakończona.")
	}

	return nil
}

// Stop zatrzymuje aktywną sesję gamescope.
func Stop() error {
	if !IsRunning() {
		fmt.Println("[gamescope-manager] Brak aktywnej sesji.")
		return nil
	}

	pid := GetPIDInt()
	if pid == -1 {
		return fmt.Errorf("nie można odczytać PID sesji")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("nie można znaleźć procesu PID %d: %w", pid, err)
	}

	fmt.Printf("[gamescope-manager] Zatrzymuję sesję (PID: %d)...\n", pid)
	LogEntry(fmt.Sprintf("STOP: PID=%d", pid))

	// Spróbuj SIGTERM
	if err := process.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "[gamescope-manager] SIGTERM nie powiódł się, używam SIGKILL...\n")
		_ = process.Signal(syscall.SIGKILL)
	}

	// Poczekaj na zakończenie (max 5s)
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		if !IsRunning() {
			break
		}
		if i == 3 {
			// Jeśli nadal żyje po 3s — SIGKILL
			_ = process.Signal(syscall.SIGKILL)
		}
	}

	RemovePID()
	LogEntry("Sesja zatrzymana")
	fmt.Println("[gamescope-manager] Sesja zatrzymana.")
	return nil
}

// ShowStatus wyświetla status aktywnej sesji i konfigurację.
func ShowStatus() {
	fmt.Println("[gamescope-manager] Status sesji:")
	fmt.Println()

	if IsRunning() {
		fmt.Printf("  Status      : AKTYWNA\n")
		fmt.Printf("  PID         : %s\n", ReadPID())
		fmt.Printf("  Gamescope   : %s\n", processStatus("gamescope"))
		fmt.Printf("  Steam       : %s\n", processStatus("steam"))
		fmt.Printf("  MangoApp    : %s\n", processStatus("mangoapp"))
	} else {
		fmt.Println("  Status      : NIEAKTYWNA")
	}

	fmt.Println()
	cfg := Load()
	fmt.Printf("  Rozdzielczość : %dx%d @ %d Hz\n", cfg.Width, cfg.Height, cfg.RefreshRate)
	fmt.Printf("  Fullscreen    : %v\n", cfg.Fullscreen)
	fmt.Printf("  VSync         : %v\n", cfg.VSync)
	fmt.Printf("  HDR           : %v\n", cfg.HDR)
	fmt.Printf("  MangoApp      : %v\n", cfg.MangoApp)
	fmt.Printf("  Auto Steam    : %v\n", cfg.AutoSteam)
	if len(cfg.ExtraFlags) > 0 {
		fmt.Printf("  Extra flags   : %s\n", strings.Join(cfg.ExtraFlags, " "))
	}
	fmt.Println()
}

// buildArgs buduje listę argumentów dla procesu gamescope.
func buildArgs(cfg Config) []string {
	var args []string

	// Rozdzielczość wewnętrzna (gry renderują w tej rozdzielczości)
	args = append(args, "-w", strconv.Itoa(cfg.Width))
	args = append(args, "-h", strconv.Itoa(cfg.Height))
	// Rozdzielczość wyjściowa (natywna monitora)
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
		args = append(args, "--mangoapp")
	}

	// Tryb zagnieżdżony jeśli mamy już serwer wyświetlania
	if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("DISPLAY") != "" {
		args = append(args, "--nested")
	}
	// Bez serwera wyświetlania — gamescope działa na KMS/TTY (tryb standalone)

	args = append(args, cfg.ExtraFlags...)
	return args
}

// buildEnv buduje środowisko dla sesji gamescope + Steam.
func buildEnv(cfg Config) []string {
	env := os.Environ()

	env = appendIfMissing(env, "XDG_SESSION_TYPE", "wayland")
	env = appendIfMissing(env, "XDG_CURRENT_DESKTOP", "gamescope")
	env = appendIfMissing(env, "XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%d", os.Getuid()))

	// Steam — optymalizacje dla gamescope
	env = appendIfMissing(env, "STEAM_USE_DYNAMIC_VRS", "1")
	env = appendIfMissing(env, "STEAM_GAMESCOPE_COLOR_MANAGED", "1")
	env = appendIfMissing(env, "STEAM_GAMESCOPE_TEARING_SUPPORTED", "1")
	env = appendIfMissing(env, "GAMESCOPE_WAYLAND_DISPLAY", "gamescope-0")

	// Mesa / Vulkan
	env = appendIfMissing(env, "MESA_VK_WSI_PRESENT_MODE", "mailbox")

	// Proton
	env = appendIfMissing(env, "STEAM_COMPAT_CLIENT_INSTALL_PATH", steamDir())

	if cfg.HDR {
		env = appendIfMissing(env, "DXVK_HDR", "1")
	}
	if cfg.MangoApp {
		env = appendIfMissing(env, "MANGOHUD", "1")
	}

	return env
}

// watchSignals obsługuje SIGINT/SIGTERM przekazując je do procesu gamescope.
func watchSignals(cmd *exec.Cmd) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-ch
	fmt.Printf("\n[gamescope-manager] Otrzymano %v, zamykam sesję...\n", sig)
	if cmd.Process != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
	}
}

// findBinary szuka pliku wykonywalnego w /usr/bin, /usr/games, potem PATH.
func findBinary(name string) (string, error) {
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

// findSteam szuka pliku wykonywalnego Steam.
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

// steamDir zwraca katalog instalacji Steam.
func steamDir() string {
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

// appendIfMissing dodaje zmienną środowiskową tylko jeśli nie jest jeszcze ustawiona.
func appendIfMissing(env []string, key, value string) []string {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return env
		}
	}
	return append(env, prefix+value)
}

// processStatus zwraca status procesu o podanej nazwie.
func processStatus(name string) string {
	if exec.Command("pgrep", "-x", name).Run() == nil {
		return "uruchomiony"
	}
	return "nie uruchomiony"
}
