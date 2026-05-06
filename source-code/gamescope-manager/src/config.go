package src

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const ConfigFile = "/etc/hackeros/gamescope-manager.conf"

// Config przechowuje konfigurację sesji gamescope.
type Config struct {
	Width       int
	Height      int
	RefreshRate int
	Fullscreen  bool
	VSync       bool
	HDR         bool
	MangoApp    bool
	AutoSteam   bool
	ExtraFlags  []string
}

// Default zwraca domyślną konfigurację.
func Default() Config {
	return Config{
		Width:       1920,
		Height:      1080,
		RefreshRate: 60,
		Fullscreen:  true,
		VSync:       true,
		HDR:         false,
		MangoApp:    true,
		AutoSteam:   true,
		ExtraFlags:  []string{},
	}
}

// Load wczytuje konfigurację z pliku. Jeśli plik nie istnieje — zwraca Default().
func Load() Config {
	cfg := Default()
	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		return cfg
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		_ = SetKey(&cfg, key, val)
	}
	return cfg
}

// Save zapisuje konfigurację do pliku.
func Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(ConfigFile), 0755); err != nil {
		return fmt.Errorf("nie można utworzyć katalogu konfiguracji: %w", err)
	}
	lines := []string{
		"# HackerOS Gamescope Manager — konfiguracja",
		"# Edytuj ręcznie lub przez: gamescope-manager config set <klucz> <wartość>",
		"",
		fmt.Sprintf("width       = %d", cfg.Width),
		fmt.Sprintf("height      = %d", cfg.Height),
		fmt.Sprintf("refresh     = %d", cfg.RefreshRate),
		fmt.Sprintf("fullscreen  = %v", cfg.Fullscreen),
		fmt.Sprintf("vsync       = %v", cfg.VSync),
		fmt.Sprintf("hdr         = %v", cfg.HDR),
		fmt.Sprintf("mangoapp    = %v", cfg.MangoApp),
		fmt.Sprintf("auto_steam  = %v", cfg.AutoSteam),
		fmt.Sprintf("extra_flags = %s", strings.Join(cfg.ExtraFlags, " ")),
	}
	return os.WriteFile(ConfigFile, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// SetKey ustawia pojedynczy klucz konfiguracji.
func SetKey(cfg *Config, key, val string) error {
	switch strings.ToLower(key) {
	case "width":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("width musi być liczbą całkowitą")
		}
		cfg.Width = v
	case "height":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("height musi być liczbą całkowitą")
		}
		cfg.Height = v
	case "refresh":
		v, err := strconv.Atoi(val)
		if err != nil {
			return fmt.Errorf("refresh musi być liczbą całkowitą")
		}
		cfg.RefreshRate = v
	case "fullscreen":
		cfg.Fullscreen = parseBool(val)
	case "vsync":
		cfg.VSync = parseBool(val)
	case "hdr":
		cfg.HDR = parseBool(val)
	case "mangoapp":
		cfg.MangoApp = parseBool(val)
	case "auto_steam":
		cfg.AutoSteam = parseBool(val)
	case "extra_flags":
		cfg.ExtraFlags = strings.Fields(val)
	default:
		return fmt.Errorf(
			"nieznany klucz: %q\nDostępne klucze: width, height, refresh, fullscreen, vsync, hdr, mangoapp, auto_steam, extra_flags",
			key,
		)
	}
	return nil
}

// Print wyświetla aktualną konfigurację.
func Print(cfg Config) {
	fmt.Println("[gamescope-manager] Aktualna konfiguracja:")
	fmt.Printf("  width       = %d\n", cfg.Width)
	fmt.Printf("  height      = %d\n", cfg.Height)
	fmt.Printf("  refresh     = %d Hz\n", cfg.RefreshRate)
	fmt.Printf("  fullscreen  = %v\n", cfg.Fullscreen)
	fmt.Printf("  vsync       = %v\n", cfg.VSync)
	fmt.Printf("  hdr         = %v\n", cfg.HDR)
	fmt.Printf("  mangoapp    = %v\n", cfg.MangoApp)
	fmt.Printf("  auto_steam  = %v\n", cfg.AutoSteam)
	if len(cfg.ExtraFlags) > 0 {
		fmt.Printf("  extra_flags = %s\n", strings.Join(cfg.ExtraFlags, " "))
	} else {
		fmt.Println("  extra_flags = (brak)")
	}
	fmt.Printf("\n  Plik: %s\n", ConfigFile)
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}
