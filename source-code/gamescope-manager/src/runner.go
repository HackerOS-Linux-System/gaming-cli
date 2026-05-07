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

// Start uruchamia sesję gamescope + opcjonalnie Steam BPM.
func Start(cfg Config) error {
	gsPath, err := findBinary("gamescope")
	if err != nil {
		return fmt.Errorf("gamescope nie jest zainstalowany\nZainstaluj: sudo apt install gamescope")
	}
	if IsRunning() {
		return fmt.Errorf("sesja gamescope już działa (PID: %s)\nUżyj 'restart' aby zrestartować", ReadPID())
	}

	args := buildArgs(cfg)

	var steamPath string
	if cfg.AutoSteam {
		steamPath, err = findSteam()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gamescope-manager] Ostrzeżenie: Steam nie znaleziony: %v\n", err)
			cfg.AutoSteam = false
		}
	}

	if cfg.AutoSteam && steamPath != "" {
		args = append(args, "--")
		args = append(args, steamPath, "-tenfoot")
	}

	env := buildEnv(cfg)
	LogEntry(fmt.Sprintf("START: %s %s", gsPath, strings.Join(args, " ")))
	fmt.Printf("[gamescope-manager] Uruchamiam: gamescope %s\n", strings.Join(args, " "))

	cmd := exec.Command(gsPath, args...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("nie można uruchomić gamescope: %w", err)
	}

	pid := cmd.Process.Pid
	fmt.Printf("[gamescope-manager] Sesja uruchomiona (PID: %d)\n", pid)
	LogEntry(fmt.Sprintf("PID=%d", pid))

	if err := WritePID(pid); err != nil {
		fmt.Fprintf(os.Stderr, "[gamescope-manager] Ostrzeżenie: nie można zapisać PID: %v\n", err)
	}

	go watchSignals(cmd)

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
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("nie można znaleźć procesu PID %d: %w", pid, err)
	}
	fmt.Printf("[gamescope-manager] Zatrzymuję sesję (PID: %d)...\n", pid)
	LogEntry(fmt.Sprintf("STOP: PID=%d", pid))

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		_ = proc.Signal(syscall.SIGKILL)
	}
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		if !IsRunning() {
			break
		}
		if i == 3 {
			_ = proc.Signal(syscall.SIGKILL)
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
	if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("DISPLAY") != "" {
		args = append(args, "--nested")
	}
	args = append(args, cfg.ExtraFlags...)
	return args
}

func buildEnv(cfg Config) []string {
	env := os.Environ()
	env = appendMissing(env, "XDG_SESSION_TYPE", "wayland")
	env = appendMissing(env, "XDG_CURRENT_DESKTOP", "gamescope")
	env = appendMissing(env, "XDG_RUNTIME_DIR", fmt.Sprintf("/run/user/%d", os.Getuid()))
	env = appendMissing(env, "STEAM_USE_DYNAMIC_VRS", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_COLOR_MANAGED", "1")
	env = appendMissing(env, "STEAM_GAMESCOPE_TEARING_SUPPORTED", "1")
	env = appendMissing(env, "GAMESCOPE_WAYLAND_DISPLAY", "gamescope-0")
	env = appendMissing(env, "MESA_VK_WSI_PRESENT_MODE", "mailbox")
	env = appendMissing(env, "STEAM_COMPAT_CLIENT_INSTALL_PATH", steamDir())
	if cfg.HDR {
		env = appendMissing(env, "DXVK_HDR", "1")
	}
	if cfg.MangoApp {
		env = appendMissing(env, "MANGOHUD", "1")
	}
	return env
}

func watchSignals(cmd *exec.Cmd) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-ch
	fmt.Printf("\n[gamescope-manager] Sygnał %v — zamykam sesję...\n", sig)
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

func steamDir() string {
	home, _ := os.UserHomeDir()
	for _, p := range []string{filepath.Join(home, ".local/share/Steam"), "/usr/share/steam", "/opt/steam"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(home, ".local/share/Steam")
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
