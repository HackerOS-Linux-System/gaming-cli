package mode

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/HackerOS-Linux-System/gaming-cli/internal/session"
)

const (
	// Numer TTY na którym działa Plasma
	plasmaTTY = "1"
	// Numer TTY dla trybu gry
	gameTTY = "2"
	// Serwis systemd dla Plazmy (SDDM)
	plasmaService = "sddm.service"
	// Serwis trybu gry
	gameModeService = "hackeros-game-mode.service"
)

// SwitchToGame przełącza system z KDE Plasma na tryb gry (Gamescope + Steam BPM).
// Strategia: zatrzymuje Plasmę (SDDM), przełącza na TTY2 i uruchamia serwis gamescope-session.
func SwitchToGame() error {
	current, err := session.CurrentMode()
	if err != nil {
		return err
	}
	if current == "game-mode" {
		fmt.Println("[gaming-cli] Już jesteś w trybie gry.")
		return nil
	}

	fmt.Println("[gaming-cli] Przełączam na tryb gry...")
	fmt.Println("[gaming-cli] Zamykam KDE Plasma (SDDM)...")

	// Sprawdź czy SDDM działa
	sddmRunning := isServiceActive(plasmaService)

	if sddmRunning {
		// Zatrzymaj SDDM — to ubije Plasmę
		if err := runSystemctl("stop", plasmaService); err != nil {
			return fmt.Errorf("nie można zatrzymać SDDM: %w", err)
		}
		// Krótka pauza żeby display server się zakończył
		time.Sleep(2 * time.Second)
	}

	// Przełącz na TTY2
	fmt.Printf("[gaming-cli] Przełączam na TTY%s...\n", gameTTY)
	if err := switchTTY(gameTTY); err != nil {
		// Nie przerywaj — może już jesteśmy na właściwym TTY
		fmt.Fprintf(os.Stderr, "[gaming-cli] Ostrzeżenie: przełączenie TTY: %v\n", err)
	}

	// Uruchom serwis trybu gry (gamescope-session + Steam)
	fmt.Println("[gaming-cli] Uruchamiam tryb gry (gamescope + Steam)...")
	if err := runSystemctl("start", gameModeService); err != nil {
		// Fallback: uruchom gamescope-manager bezpośrednio
		fmt.Fprintf(os.Stderr, "[gaming-cli] Serwis %s niedostępny, próbuję uruchomić gamescope-manager...\n", gameModeService)
		if err2 := launchGamescopeManager(); err2 != nil {
			// Przywróć SDDM w razie błędu
			_ = runSystemctl("start", plasmaService)
			return fmt.Errorf("nie można uruchomić trybu gry: gamescope-manager: %w", err2)
		}
	}

	if err := session.SetMode("game-mode"); err != nil {
		fmt.Fprintf(os.Stderr, "[gaming-cli] Ostrzeżenie: nie można zapisać stanu: %v\n", err)
	}

	fmt.Println("[gaming-cli] Tryb gry aktywny. Miłej gry! 🎮")
	return nil
}

// SwitchToDesktop przełącza z trybu gry z powrotem na KDE Plasma.
func SwitchToDesktop() error {
	current, err := session.CurrentMode()
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
		fmt.Println("[gaming-cli] Zamykam tryb gry...")
		if err := runSystemctl("stop", gameModeService); err != nil {
			fmt.Fprintf(os.Stderr, "[gaming-cli] Ostrzeżenie: zatrzymanie trybu gry: %v\n", err)
		}
		time.Sleep(2 * time.Second)
	} else {
		// Spróbuj ubić gamescope-manager jeśli działa jako proces
		killProcess("gamescope-manager")
		killProcess("gamescope")
		time.Sleep(1 * time.Second)
	}

	// Przełącz na TTY1 (Plasma)
	fmt.Printf("[gaming-cli] Przełączam na TTY%s...\n", plasmaTTY)
	if err := switchTTY(plasmaTTY); err != nil {
		fmt.Fprintf(os.Stderr, "[gaming-cli] Ostrzeżenie: przełączenie TTY: %v\n", err)
	}

	// Uruchom SDDM
	fmt.Println("[gaming-cli] Uruchamiam KDE Plasma (SDDM)...")
	if err := runSystemctl("start", plasmaService); err != nil {
		return fmt.Errorf("nie można uruchomić SDDM: %w", err)
	}

	if err := session.SetMode("desktop-mode"); err != nil {
		fmt.Fprintf(os.Stderr, "[gaming-cli] Ostrzeżenie: nie można zapisać stanu: %v\n", err)
	}

	fmt.Println("[gaming-cli] Tryb pulpitu aktywny. KDE Plasma uruchomiona.")
	return nil
}

// launchGamescopeManager uruchamia gamescope-manager jako proces w tle na aktywnym TTY.
func launchGamescopeManager() error {
	gsm, err := exec.LookPath("gamescope-manager")
	if err != nil {
		return fmt.Errorf("gamescope-manager nie znaleziony w PATH: %w", err)
	}

	// Ustaw zmienne środowiskowe dla gamescope na TTY
	env := os.Environ()
	env = setEnv(env, "XDG_SESSION_TYPE", "wayland")
	env = setEnv(env, "XDG_CURRENT_DESKTOP", "gamescope")

	cmd := exec.Command(gsm, "start", "--steam")
	cmd.Env = env
	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("nie można uruchomić gamescope-manager: %w", err)
	}

	fmt.Printf("[gaming-cli] gamescope-manager uruchomiony (PID %d)\n", cmd.Process.Pid)
	// Zapisz PID do pliku stanu
	pidFile := "/var/lib/hackeros-gaming/gamescope-manager.pid"
	_ = os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0644)

	return nil
}

// switchTTY przełącza aktywny terminal wirtualny (TTY).
func switchTTY(tty string) error {
	// chvt wymaga roota lub odpowiednich uprawnień
	cmd := exec.Command("chvt", tty)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runSystemctl uruchamia komendę systemctl.
func runSystemctl(action, service string) error {
	cmd := exec.Command("systemctl", action, service)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// isServiceActive sprawdza czy serwis systemd jest aktywny.
func isServiceActive(service string) bool {
	cmd := exec.Command("systemctl", "is-active", "--quiet", service)
	return cmd.Run() == nil
}

// killProcess wysyła SIGTERM do procesu o podanej nazwie.
func killProcess(name string) {
	_ = exec.Command("pkill", "-TERM", name).Run()
}

// setEnv ustawia zmienną środowiskową w slice'u, nadpisując istniejącą.
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
