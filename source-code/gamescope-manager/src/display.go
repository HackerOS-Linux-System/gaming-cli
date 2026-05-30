package src

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// DisplayInfo przechowuje wykrytą rozdzielczość i częstotliwość odświeżania.
type DisplayInfo struct {
	Width       int
	Height      int
	RefreshRate int
	Connector   string // np. "HDMI-A-1", "DP-1", "eDP-1"
	Source      string // skąd wykryto: "drm", "xrandr", "edid", "fallback"
}

// AutoDetectDisplay wykrywa natywną rozdzielczość monitora.
// Kolejność prób: DRM sysfs → xrandr → drm_info → fallback 1920x1080@60.
//
// Jeśli użytkownik ustawił Width/Height w konfiguracji (!=0), funkcja
// NICZEGO nie nadpisuje — respektuje wybór użytkownika.
func AutoDetectDisplay() DisplayInfo {
	// 1. DRM sysfs (/sys/class/drm) — najbardziej niezawodne, bez X/Wayland
	if info, ok := detectViaDRM(); ok {
		return info
	}

	// 2. xrandr — jeśli jesteśmy w X11 (np. zagnieżdżone uruchomienie)
	if info, ok := detectViaXrandr(); ok {
		return info
	}

	// 3. drm_info — jeśli zainstalowane
	if info, ok := detectViaDRMInfo(); ok {
		return info
	}

	// 4. Fallback — 1920×1080@60 (gamescope sam wykryje przez DRM przy embedded)
	return DisplayInfo{
		Width:       0, // 0 = niech gamescope samo wykryje z DRM
		Height:      0,
		RefreshRate: 60,
		Connector:   "",
		Source:      "fallback",
	}
}

// ── DRM sysfs ─────────────────────────────────────────────────────────────────

// detectViaDRM skanuje /sys/class/drm/*/modes i czyta natywną rozdzielczość.
func detectViaDRM() (DisplayInfo, bool) {
	const drmBase = "/sys/class/drm"

	entries, err := os.ReadDir(drmBase)
	if err != nil {
		return DisplayInfo{}, false
	}

	// Preferuj złącza zewnętrzne (HDMI, DP) nad wewnętrznymi (eDP, LVDS)
	// bo gracz prawdopodobnie gra na TV/monitorze
	type candidate struct {
		info     DisplayInfo
		priority int // im wyższy, tym lepszy
	}
	var candidates []candidate

	for _, e := range entries {
		name := e.Name()
		// pomijamy "version", "card0", itp. — chcemy np. "card0-HDMI-A-1"
		if !strings.Contains(name, "-") {
			continue
		}

		modesPath := filepath.Join(drmBase, name, "modes")
		statusPath := filepath.Join(drmBase, name, "status")

		// Sprawdź czy złącze jest podłączone
		statusData, err := os.ReadFile(statusPath)
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(statusData)) != "connected" {
			continue
		}

		// Wczytaj dostępne tryby (pierwsza linia = najwyższy preferowany)
		modesData, err := os.ReadFile(modesPath)
		if err != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(modesData)), "\n")
		if len(lines) == 0 || lines[0] == "" {
			continue
		}

		// Format trybu: "1920x1080" lub "1920x1080@60"
		w, h, r := parseModeLine(lines[0])
		if w == 0 || h == 0 {
			continue
		}

		// Jeśli nie ma refresh w modes, spróbuj z pliku "current_mode" lub "vrr_capable"
		if r == 0 {
			r = readRefreshFromSysfs(filepath.Join(drmBase, name))
			if r == 0 {
				r = 60
			}
		}

		// Wyodrębnij nazwę złącza (np. "card0-HDMI-A-1" → "HDMI-A-1")
		parts := strings.SplitN(name, "-", 2)
		connector := name
		if len(parts) == 2 {
			connector = parts[1]
		}

		priority := connectorPriority(connector)
		candidates = append(candidates, candidate{
			info: DisplayInfo{
				Width:       w,
				Height:      h,
				RefreshRate: r,
				Connector:   connector,
				Source:      "drm",
			},
			priority: priority,
		})
	}

	if len(candidates) == 0 {
		return DisplayInfo{}, false
	}

	// Wybierz złącze o najwyższym priorytecie
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].priority > candidates[j].priority
	})

	return candidates[0].info, true
}

// connectorPriority nadaje priorytety złączom (wyższy = preferowany).
func connectorPriority(connector string) int {
	lo := strings.ToLower(connector)
	switch {
		case strings.HasPrefix(lo, "hdmi"):
			return 100
		case strings.HasPrefix(lo, "dp-") || strings.HasPrefix(lo, "displayport"):
			return 90
		case strings.HasPrefix(lo, "usb-c") || strings.Contains(lo, "usb"):
			return 80
		case strings.HasPrefix(lo, "edp") || strings.HasPrefix(lo, "lvds"):
			return 50 // wbudowany panel laptopa — niższy priorytet
		default:
			return 60
	}
}

// parseModeLine parsuje linie w formacie "1920x1080" lub "1920x1080@60.00"
func parseModeLine(line string) (w, h, r int) {
	line = strings.TrimSpace(line)
	// Wyodrębnij opcjonalny refresh po "@"
	if idx := strings.Index(line, "@"); idx >= 0 {
		rStr := strings.TrimSpace(line[idx+1:])
		if rf, err := strconv.ParseFloat(rStr, 64); err == nil {
			r = int(rf)
		}
		line = line[:idx]
	}
	// Podziel na WxH
	parts := strings.SplitN(line, "x", 2)
	if len(parts) != 2 {
		return 0, 0, r
	}
	wv, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	hv, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0, r
	}
	return wv, hv, r
}

// readRefreshFromSysfs próbuje odczytać refresh z pliku "current_mode" lub skanuje modes.
func readRefreshFromSysfs(connectorDir string) int {
	// Niektóre sterowniki mają plik "current_mode" z formatem "1920x1080@60"
	data, err := os.ReadFile(filepath.Join(connectorDir, "current_mode"))
	if err == nil {
		_, _, r := parseModeLine(strings.TrimSpace(string(data)))
		if r > 0 {
			return r
		}
	}

	// Sprawdź vrr_capable — jeśli monitor obsługuje VRR, spróbuj wyższy refresh
	// To tylko heurystyka
	return 0
}

// ── xrandr ────────────────────────────────────────────────────────────────────

func detectViaXrandr() (DisplayInfo, bool) {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return DisplayInfo{}, false
	}
	out, err := exec.Command("xrandr", "--current").Output()
	if err != nil {
		return DisplayInfo{}, false
	}

	// Szukaj linii: "   1920x1080     60.00*+"
	// lub: "HDMI-1 connected 1920x1080+0+0"
	var w, h, r int
	var connector string

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var currentConnector string
	for scanner.Scan() {
		line := scanner.Text()
		// Linia złącza: "HDMI-1 connected ..."
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && parts[1] == "connected" {
				currentConnector = parts[0]
				// Może mieć rozdzielczość w tej samej linii: "HDMI-1 connected 1920x1080+0+0"
				for _, p := range parts[2:] {
					if strings.Contains(p, "x") && strings.Contains(p, "+") {
						res := strings.SplitN(p, "+", 2)[0]
						pw, ph, _ := parseModeLine(res)
						if pw > 0 && ph > 0 && w == 0 {
							w, h = pw, ph
							connector = currentConnector
						}
					}
				}
			}
			continue
		}
		// Linia trybu:  "   1920x1080     60.00*+"
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if strings.Contains(fields[0], "x") {
			pw, ph, _ := parseModeLine(fields[0])
			if pw == 0 || ph == 0 {
				continue
			}
			// Szukaj trybu z gwiazdką (*) = aktywny
			for _, f := range fields[1:] {
				if strings.Contains(f, "*") {
					if rf, err := strconv.ParseFloat(strings.Trim(f, "+* "), 64); err == nil {
						if w == 0 || (pw*ph > w*h) {
							w, h = pw, ph
							r = int(rf)
							connector = currentConnector
						}
					}
				}
			}
		}
	}

	if w == 0 || h == 0 {
		return DisplayInfo{}, false
	}
	if r == 0 {
		r = 60
	}
	return DisplayInfo{
		Width:       w,
		Height:      h,
		RefreshRate: r,
		Connector:   connector,
		Source:      "xrandr",
	}, true
}

// ── drm_info ──────────────────────────────────────────────────────────────────

func detectViaDRMInfo() (DisplayInfo, bool) {
	if _, err := exec.LookPath("drm_info"); err != nil {
		return DisplayInfo{}, false
	}
	out, err := exec.Command("drm_info", "-j").Output()
	if err != nil {
		return DisplayInfo{}, false
	}

	// Prosta parsownia JSON: szukaj "clock" i rozdzielczości
	// drm_info -j daje dość skomplikowany output, wyciągamy najważniejsze
	content := string(out)
	var w, h, r int

	// Szukaj wzorca rozdzielczości w JSON
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "\"hdisplay\"") {
			val := extractJSONInt(line)
			if val > 0 {
				w = val
			}
		}
		if strings.Contains(line, "\"vdisplay\"") {
			val := extractJSONInt(line)
			if val > 0 {
				h = val
			}
		}
		if strings.Contains(line, "\"vrefresh\"") {
			val := extractJSONInt(line)
			if val > 0 {
				r = val
			}
		}
		if w > 0 && h > 0 && r > 0 {
			break
		}
	}

	if w == 0 || h == 0 {
		return DisplayInfo{}, false
	}
	if r == 0 {
		r = 60
	}
	return DisplayInfo{
		Width:       w,
		Height:      h,
		RefreshRate: r,
		Connector:   "",
		Source:      "drm_info",
	}, true
}

func extractJSONInt(line string) int {
	// Format: "   \"key\": 1920,"
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return 0
	}
	val := strings.Trim(strings.TrimSpace(parts[1]), ", \t")
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return n
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// ResolveResolution zwraca ostateczną rozdzielczość do użycia przez gamescope.
// Jeśli cfg.Width/Height == 0 (domyślne lub nie ustawione przez użytkownika),
// uruchamia auto-detekcję. Zwraca też DisplayInfo z informacją o źródle.
func ResolveResolution(cfg Config) (w, h, refresh int, info DisplayInfo) {
	userSetWidth := cfg.Width != 1920 || cfg.Height != 1080 // heurystyka: != defaults
	// Lepsza heurystyka: specjalna wartość 0 oznacza "auto"
	if cfg.Width == 0 || cfg.Height == 0 {
		detected := AutoDetectDisplay()
		if detected.Width == 0 {
			// Gamescope wykryje sam przez DRM — nie podawaj -w/-h/-W/-H
			return 0, 0, detected.RefreshRate, detected
		}
		return detected.Width, detected.Height, detected.RefreshRate, detected
	}
	_ = userSetWidth
	return cfg.Width, cfg.Height, cfg.RefreshRate, DisplayInfo{
		Width:       cfg.Width,
		Height:      cfg.Height,
		RefreshRate: cfg.RefreshRate,
		Source:      "config",
	}
}

// FormatResolution zwraca czytelny string rozdzielczości.
func (d DisplayInfo) FormatResolution() string {
	if d.Width == 0 {
		return fmt.Sprintf("auto (DRM) @%dHz [%s]", d.RefreshRate, d.Source)
	}
	s := fmt.Sprintf("%dx%d@%dHz [%s]", d.Width, d.Height, d.RefreshRate, d.Source)
	if d.Connector != "" {
		s += " on " + d.Connector
	}
	return s
}
