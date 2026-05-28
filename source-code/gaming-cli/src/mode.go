package src

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	plasmaTTY       = "2" // tty2 = SDDM/KDE Plasma w HackerOS
	gameTTY         = "3" // tty3 = tryb gry (NIE tty2, żeby nie zabijać SDDM!)
plasmaService   = "sddm.service"
gameModeService = "hackeros-game-mode.service"
)

// SwitchToGame przełącza na tryb gry.
// SDDM NIE jest wyłączane — gamescope działa na tty3, KDE zostaje na tty2.
func SwitchToGame() error {
	cur, err := CurrentMode()
	if err != nil {
		return err
	}
	if cur == "game-mode" {
		return fmt.Errorf("juz jestes w trybie gry")
	}

	// Przełącz na tty3 (NIE wyłączaj SDDM!)
	if err := switchTTY(gameTTY); err != nil {
		fmt.Fprintf(os.Stderr, "ostrzezenie TTY: %v\n", err)
	}

	// Spróbuj przez systemd, fallback na bezpośrednie uruchomienie
	if err := runSystemctl("start", gameModeService); err != nil {
		if err2 := launchGamescopeManagerDirect(); err2 != nil {
			// Wróć na tty2 jeśli start się nie powiódł
			_ = switchTTY(plasmaTTY)
			return fmt.Errorf("nie mozna uruchomic trybu gry: %w", err2)
		}
	}

	return SetMode("game-mode")
}

// SwitchToDesktop przełącza z powrotem na KDE Plasma / SDDM.
func SwitchToDesktop() error {
	cur, err := CurrentMode()
	if err != nil {
		return err
	}
	if cur == "desktop-mode" {
		return fmt.Errorf("juz jestes w trybie pulpitu")
	}

	// Zatrzymaj tryb gry
	if isServiceActive(gameModeService) {
		_ = runSystemctl("stop", gameModeService)
		time.Sleep(2 * time.Second)
	} else {
		// Fallback: zabij gamescope-manager i gamescope
		_ = exec.Command("pkill", "-TERM", "gamescope-manager").Run()
		_ = exec.Command("pkill", "-TERM", "gamescope").Run()
		time.Sleep(1 * time.Second)
	}

	// Przełącz z tty3 na tty2 gdzie jest SDDM/KDE
	if err := switchTTY(plasmaTTY); err != nil {
		fmt.Fprintf(os.Stderr, "ostrzezenie TTY: %v\n", err)
	}

	// SDDM powinno już działać na tty2 — sprawdź i ewentualnie uruchom
	if !isServiceActive(plasmaService) {
		if err := runSystemctl("start", plasmaService); err != nil {
			return fmt.Errorf("nie mozna uruchomic SDDM: %w", err)
		}
	}

	return SetMode("desktop-mode")
}

func launchGamescopeManagerDirect() error {
	p := "/usr/bin/gamescope-manager"
	if _, err := os.Stat(p); err != nil {
		lp, err2 := exec.LookPath("gamescope-manager")
		if err2 != nil {
			return fmt.Errorf("gamescope-manager nie znaleziony")
		}
		p = lp
	}
	env := setEnv(os.Environ(), "XDG_SESSION_TYPE", "wayland")
	env = setEnv(env, "XDG_CURRENT_DESKTOP", "gamescope")
	cmd := exec.Command(p, "start", "--steam")
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = os.MkdirAll("/var/lib/hackeros-gaming", 0755)
	_ = os.WriteFile("/var/lib/hackeros-gaming/gamescope-manager.pid",
			 []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0644)
	return nil
}

func switchTTY(tty string) error      { return exec.Command("chvt", tty).Run() }
func isServiceActive(svc string) bool { return exec.Command("systemctl", "is-active", "--quiet", svc).Run() == nil }

func runSystemctl(action, svc string) error {
	cmd := exec.Command("systemctl", action, svc)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func setEnv(env []string, k, v string) []string {
	p := k + "="
	for i, e := range env {
		if strings.HasPrefix(e, p) {
			env[i] = p + v
			return env
		}
	}
	return append(env, p+v)
}
