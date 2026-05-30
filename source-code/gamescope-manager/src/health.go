package src

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// HealthStatus opisuje wynik jednego sprawdzenia.
type HealthStatus struct {
	Name    string
	OK      bool
	Message string
	Fatal   bool // false = ostrzeżenie, true = blokuje start
}

// RunHealthChecks wykonuje wszystkie sprawdzenia przed startem sesji.
// Zwraca listę wyników i bool czy można startować.
func RunHealthChecks(cfg Config) ([]HealthStatus, bool) {
	var results []HealthStatus
	canStart := true

	check := func(name string, fatal bool, fn func() (bool, string)) {
		ok, msg := fn()
		results = append(results, HealthStatus{
			Name:    name,
			OK:      ok,
			Message: msg,
			Fatal:   fatal,
		})
		if !ok && fatal {
			canStart = false
		}
	}

	// === Krytyczne (blokują start) ===

	check("gamescope zainstalowany", true, func() (bool, string) {
		_, err := findBinary("gamescope")
		if err != nil {
			return false, "gamescope nie znaleziony. Zainstaluj: sudo apt install gamescope"
		}
		return true, "OK"
	})

	check("uprawnienia roota", true, func() (bool, string) {
		if os.Geteuid() != 0 {
			return false, "wymagane sudo"
		}
		return true, "OK"
	})

	check("Vulkan / GPU driver", true, func() (bool, string) {
		// Sprawdź czy vulkaninfo działa lub czy jest /dev/dri
		if entries, err := os.ReadDir("/dev/dri"); err == nil && len(entries) > 0 {
			return true, fmt.Sprintf("/dev/dri: %d urzadzen", len(entries))
		}
		return false, "/dev/dri niedostepne — brak sterownikow GPU"
	})

	check("katalogi stanu", true, func() (bool, string) {
		dirs := []string{"/var/lib/hackeros-gaming", "/var/log/hackeros-gaming"}
		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				return false, fmt.Sprintf("nie mozna utworzyc %s: %v", d, err)
			}
		}
		return true, "OK"
	})

	// === Ostrzeżenia (nie blokują) ===

	if cfg.AutoSteam {
		check("Steam zainstalowany", false, func() (bool, string) {
			_, err := findSteam()
			if err != nil {
				return false, "Steam nie znaleziony — sesja uruchomi sie bez Steam"
			}
			return true, "OK"
		})

		check("Steam nie dziala juz w tle", false, func() (bool, string) {
			if isSteamRunning() {
				return false, "Steam juz dziala — gamescope uruchomi nowa instancje Steam BPM"
			}
			return true, "OK"
		})
	}

	check("PipeWire / audio", false, func() (bool, string) {
		if exec.Command("pgrep", "-x", "pipewire").Run() == nil {
			return true, "PipeWire uruchomiony"
		}
		if exec.Command("pgrep", "-x", "pulseaudio").Run() == nil {
			return true, "PulseAudio uruchomiony"
		}
		return false, "brak PipeWire ani PulseAudio — mozliwy brak dzwieku"
	})

	check("dbus sesji", false, func() (bool, string) {
		if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
			return true, "OK (z env)"
		}
		sockPath := fmt.Sprintf("/run/user/%d/bus", os.Getuid())
		if _, err := os.Stat(sockPath); err == nil {
			return true, "OK (" + sockPath + ")"
		}
		return false, "brak sesji dbus — Steam moze nie dzialac poprawnie"
	})

	check("MangoHUD", false, func() (bool, string) {
		if !cfg.MangoApp {
			return true, "wylaczony w konfiguracji"
		}
		if _, err := exec.LookPath("mangohud"); err != nil {
			if _, err2 := exec.LookPath("mangoapp"); err2 != nil {
				return false, "mangoapp/mangohud nie znalezione — overlay FPS niedostepny"
			}
		}
		return true, "OK"
	})

	check("HDR - wsparcie kernela", false, func() (bool, string) {
		if !cfg.HDR {
			return true, "HDR wylaczony w konfiguracji"
		}
		// Sprawdź czy kernel ma wsparcie HDR przez DRM
		entries, err := os.ReadDir("/sys/class/drm")
		if err != nil {
			return false, "nie mozna sprawdzic DRM"
		}
		for _, e := range entries {
			hdrPath := "/sys/class/drm/" + e.Name() + "/hdr_output_metadata"
			if _, err := os.Stat(hdrPath); err == nil {
				return true, "wsparcie HDR w DRM znalezione"
			}
		}
		return false, "brak wsparcia HDR w sterowniku DRM — HDR moze nie dzialac"
	})

	check("brak aktywnej sesji gamescope", false, func() (bool, string) {
		if IsRunning() {
			return false, fmt.Sprintf("sesja juz dziala (PID: %s) — uzyj 'restart'", ReadPID())
		}
		return true, "OK"
	})

	check("tty3 dostepne", false, func() (bool, string) {
		if _, err := os.Stat("/dev/tty3"); err != nil {
			return false, "brak /dev/tty3"
		}
		return true, "OK"
	})

	check("systemd-logind", false, func() (bool, string) {
		if exec.Command("systemctl", "is-active", "--quiet", "systemd-logind").Run() == nil {
			return true, "OK"
		}
		return false, "systemd-logind nieaktywny — mozliwe problemy z sesja"
	})

	// Sprawdź rozdzielczość
	check("rozdzielczosc", false, func() (bool, string) {
		if cfg.Width > 0 && cfg.Height > 0 {
			return true, fmt.Sprintf("%dx%d@%dHz (z konfiguracji)", cfg.Width, cfg.Height, cfg.RefreshRate)
		}
		detected := AutoDetectDisplay()
		if detected.Source == "fallback" {
			return false, "nie wykryto monitora — gamescope uzyje DRM auto-detekcji"
		}
		return true, fmt.Sprintf("auto: %s", detected.FormatResolution())
	})

	return results, canStart
}

// FormatHealthReport formatuje wyniki health check do wyświetlenia.
func FormatHealthReport(results []HealthStatus) string {
	var sb strings.Builder
	for _, r := range results {
		var icon, status string
		if r.OK {
			icon = "  [OK] "
		} else if r.Fatal {
			icon = "  [!!] "
		} else {
			icon = "  [??] "
		}
		status = icon + r.Name
		if !r.OK {
			status += ": " + r.Message
		}
		sb.WriteString(status + "\n")
	}
	return sb.String()
}
