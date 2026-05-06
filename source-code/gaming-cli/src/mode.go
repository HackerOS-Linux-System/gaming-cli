package src

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	plasmaTTY       = "1"
	gameTTY         = "2"
	plasmaService   = "sddm.service"
	gameModeService = "hackeros-game-mode.service"
)

func SwitchToGame() error {
	cur, err := CurrentMode()
	if err != nil {
		return err
	}
	if cur == "game-mode" {
		return fmt.Errorf("już jesteś w trybie gry")
	}
	if isServiceActive(plasmaService) {
		if err := runSystemctl("stop", plasmaService); err != nil {
			return fmt.Errorf("nie można zatrzymać SDDM: %w", err)
		}
		time.Sleep(2 * time.Second)
	}
	if err := switchTTY(gameTTY); err != nil {
		fmt.Fprintf(os.Stderr, "ostrzeżenie TTY: %v\n", err)
	}
	if err := runSystemctl("start", gameModeService); err != nil {
		if err2 := launchGamescopeManagerDirect(); err2 != nil {
			_ = runSystemctl("start", plasmaService)
			return fmt.Errorf("nie można uruchomić trybu gry: %w", err2)
		}
	}
	return SetMode("game-mode")
}

func SwitchToDesktop() error {
	cur, err := CurrentMode()
	if err != nil {
		return err
	}
	if cur == "desktop-mode" {
		return fmt.Errorf("już jesteś w trybie pulpitu")
	}
	if isServiceActive(gameModeService) {
		_ = runSystemctl("stop", gameModeService)
		time.Sleep(2 * time.Second)
	} else {
		_ = exec.Command("pkill", "-TERM", "gamescope-manager").Run()
		_ = exec.Command("pkill", "-TERM", "gamescope").Run()
		time.Sleep(1 * time.Second)
	}
	if err := switchTTY(plasmaTTY); err != nil {
		fmt.Fprintf(os.Stderr, "ostrzeżenie TTY: %v\n", err)
	}
	if err := runSystemctl("start", plasmaService); err != nil {
		return fmt.Errorf("nie można uruchomić SDDM: %w", err)
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

func switchTTY(tty string) error        { return exec.Command("chvt", tty).Run() }
func isServiceActive(svc string) bool   { return exec.Command("systemctl", "is-active", "--quiet", svc).Run() == nil }
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
