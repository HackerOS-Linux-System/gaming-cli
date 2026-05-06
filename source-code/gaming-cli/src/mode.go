package src

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	// TTY dla KDE Plasma (SDDM domyślnie startuje na tty1)
	plasmaTTY = "1"
	// TTY dla trybu gry
	gameTTY = "2"
	// Serwis systemd dla display managera
	plasmaService = "sddm.service"
	// Serwis trybu gry — uruchamia gamescope-manager w tle
	gameModeService = "hackeros-game-mode.service"
)

// SwitchToGame przełącza system z KDE Plasma na tryb gry.
// Strategia:
//  1. Zatrzymuje SDDM (co ubija Plasmę).
//  2. Przełącza konsolę wirtualną na TTY2.
//  3. Uruchamia serwis hackeros-game-mode (gamescope + Steam BPM).
//     Jeśli serwis nie istnieje — uruchamia gamescope-manager bezpośrednio.
func SwitchToGame() error {
	current, err := CurrentMode()
	if err != nil {
		return err
	}
	if current == "game-mode" {
		fmt.Println("[gaming-cli] Już jesteś w trybie gry.")
		return nil
	}

	fmt.Println("[gaming-cli] Przełączam na tryb gry...")

	// Zatrzymaj SDDM jeśli działa
	if isServiceActive(plasmaService) {
		fmt.Println("[gaming-cli] Zatrzymuję KDE Plasma (SDDM)...")
		if err := runSystemctl("stop", plasmaService); err != nil {
			return fmt.Errorf("nie można zatrzymać %s: %w", plasmaService, err)
		}
		// Daj czas display serverowi na pełne zamknięcie
		time.Sleep(2 * time.Second)
	}

	// Przełącz na TTY2
	fmt.Printf("[gaming-cli] Przełączam na TTY%s...\n", gameTTY)
	if err := switchTTY(gameTTY); err != nil {
		fmt.Fprintf(os.Stderr, "[gaming-cli] Ostrzeżenie: przełączenie TTY: %v\n", err)
	}

	// Uruchom serwis trybu gry
	fmt.Println("[gaming-cli] Uruchamiam tryb gry (gamescope + Steam)...")
	if err := runSystemctl("start", gameModeService); err != nil {
		// Fallback: gamescope-manager bezpośrednio
		fmt.Fprintf(os.Stderr,
			"[gaming-cli] Serwis %s niedostępny, próbuję gamescope-manager...\n",
			gameModeService,
		)
		if err2 := launchGamescopeManagerDirect(); err2 != nil {
			// Przywróć Plasmę jeśli coś poszło nie tak
			_ = runSystemctl("start", plasmaService)
			return fmt.Errorf("nie można uruchomić trybu gry: %w", err2)
		}
	}

	if err := SetMode("game-mode"); err != nil {
		fmt.Fprintf(os.Stderr, "[gaming-cli] Ostrzeżenie: nie można zapisać stanu: %v\n", err)
	}

	fmt.Println("[gaming-cli] Tryb gry aktywny. Miłej gry! 🎮")
	return nil
}

// SwitchToDesktop przełącza z trybu gry z powrotem na KDE Plasma.
func SwitchToDesktop() error {
	current, err := CurrentMode()
	if err != nil {
		return err
	}
	if current == "desktop-mode" {
		fmt.Println("[gaming-cli] Już jesteś w trybie pulpitu.")
		return nil
	}

	fmt.Println("[gaming-cli] Przełączam na tryb pulpitu (KDE Plasma)...")

	// Zatrzymaj serwis trybu gry
	if isServiceActive(gameModeService) {
		fmt.Println("[gaming-cli] Zatrzymuję tryb gry...")
		if err := runSystemctl("stop", gameModeService); err != nil {
			fmt.Fprintf(os.Stderr, "[gaming-cli] Ostrzeżenie: %v\n", err)
		}
		time.Sleep(2 * time.Second)
	} else {
		// Fallback: ubij procesy gamescope-manager i gamescope
		killByName("gamescope-manager")
		killByName("gamescope")
		time.Sleep(1 * time.Second)
	}

	// Przełącz na TTY1
	fmt.Printf("[gaming-cli] Przełączam na TTY%s...\n", plasmaTTY)
	if err := switchTTY(plasmaTTY); err != nil {
		fmt.Fprintf(os.Stderr, "[gaming-cli] Ostrzeżenie: przełączenie TTY: %v\n", err)
	}

	// Uruchom SDDM
	fmt.Println("[gaming-cli] Uruchamiam KDE Plasma (SDDM)...")
	if err := runSystemctl("start", plasmaService); err != nil {
		return fmt.Errorf("nie można uruchomić %s: %w", plasmaService, err)
	}

	if err := SetMode("desktop-mode"); err != nil {
		fmt.Fprintf(os.Stderr, "[gaming-cli] Ostrzeżenie: nie można zapisać stanu: %v\n", err)
	}

	fmt.Println("[gaming-cli] Tryb pulpitu aktywny.")
	return nil
}

// launchGamescopeManagerDirect uruchamia gamescope-manager jako proces w tle.
// Używany gdy serwis systemd nie jest dostępny.
func launchGamescopeManagerDirect() error {
	gsmPath, err := findBinary("gamescope-manager")
	if err != nil {
		return fmt.Errorf("gamescope-manager nie znaleziony: %w", err)
	}

	env := os.Environ()
	env = setEnv(env, "XDG_SESSION_TYPE", "wayland")
	env = setEnv(env, "XDG_CURRENT_DESKTOP", "gamescope")

	cmd := exec.Command(gsmPath, "start", "--steam")
	cmd.Env = env
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("nie można uruchomić gamescope-manager: %w", err)
	}

	fmt.Printf("[gaming-cli] gamescope-manager uruchomiony (PID: %d)\n", cmd.Process.Pid)

	// Zapisz PID
	pidPath := "/var/lib/hackeros-gaming/gamescope-manager.pid"
	_ = os.MkdirAll("/var/lib/hackeros-gaming", 0755)
	_ = os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0644)

	return nil
}

// switchTTY przełącza aktywną konsolę wirtualną przez chvt.
func switchTTY(tty string) error {
	return exec.Command("chvt", tty).Run()
}

// runSystemctl uruchamia polecenie systemctl z podaną akcją i serwisem.
func runSystemctl(action, service string) error {
	cmd := exec.Command("systemctl", action, service)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// isServiceActive sprawdza czy serwis systemd jest aktywny.
func isServiceActive(service string) bool {
	return exec.Command("systemctl", "is-active", "--quiet", service).Run() == nil
}

// killByName wysyła SIGTERM do procesu o podanej nazwie.
func killByName(name string) {
	_ = exec.Command("pkill", "-TERM", name).Run()
}

// findBinary szuka pliku wykonywalnego najpierw w /usr/bin, potem w PATH.
func findBinary(name string) (string, error) {
	fixed := "/usr/bin/" + name
	if _, err := os.Stat(fixed); err == nil {
		return fixed, nil
	}
	return exec.LookPath(name)
}

// setEnv ustawia zmienną środowiskową w slice, nadpisując istniejącą wartość.
func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
