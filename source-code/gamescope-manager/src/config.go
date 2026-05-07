package src

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const ConfigFile = "/etc/hackeros/gamescope-manager.hk"

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

// Load wczytuje konfigurację z pliku .hk.
// Jeśli plik nie istnieje — zwraca Default().
func Load() Config {
	cfg := Default()
	hkcfg, err := LoadHKFile(ConfigFile)
	if err != nil {
		return cfg
	}
	sec, ok := hkcfg.Sections["gamescope"]
	if !ok || sec.Type != HkMap {
		return cfg
	}
	readInt := func(key string, dest *int) {
		if v, err := sec.Get(key); err == nil {
			if n, err := v.AsNumber(); err == nil {
				*dest = int(n)
			}
		}
	}
	readBool := func(key string, dest *bool) {
		if v, err := sec.Get(key); err == nil {
			if b, err := v.AsBool(); err == nil {
				*dest = b
			}
		}
	}
	readInt("width", &cfg.Width)
	readInt("height", &cfg.Height)
	readInt("refresh", &cfg.RefreshRate)
	readBool("fullscreen", &cfg.Fullscreen)
	readBool("vsync", &cfg.VSync)
	readBool("hdr", &cfg.HDR)
	readBool("mangoapp", &cfg.MangoApp)
	readBool("auto_steam", &cfg.AutoSteam)

	if v, err := sec.Get("extra_flags"); err == nil && v.Type == HkArray {
		for _, item := range v.Array {
			if s, err := item.AsString(); err == nil {
				cfg.ExtraFlags = append(cfg.ExtraFlags, s)
			}
		}
	} else if v, err := sec.Get("extra_flags"); err == nil {
		if s, err := v.AsString(); err == nil && s != "" {
			cfg.ExtraFlags = strings.Fields(s)
		}
	}

	return cfg
}

// Save zapisuje konfigurację do pliku .hk.
func Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(ConfigFile), 0755); err != nil {
		return fmt.Errorf("nie można utworzyć katalogu konfiguracji: %w", err)
	}

	hkcfg := &HkConfig{
		Sections:     make(map[string]HkValue),
		SectionOrder: []string{"gamescope"},
	}

	section := hkMap()
	setHK := func(key string, val HkValue) {
		section.Map[key] = val
		section.MapKeys = append(section.MapKeys, key)
	}

	setHK("width", hkNum(float64(cfg.Width)))
	setHK("height", hkNum(float64(cfg.Height)))
	setHK("refresh", hkNum(float64(cfg.RefreshRate)))
	setHK("fullscreen", hkBool(cfg.Fullscreen))
	setHK("vsync", hkBool(cfg.VSync))
	setHK("hdr", hkBool(cfg.HDR))
	setHK("mangoapp", hkBool(cfg.MangoApp))
	setHK("auto_steam", hkBool(cfg.AutoSteam))

	flags := make([]HkValue, len(cfg.ExtraFlags))
	for i, f := range cfg.ExtraFlags {
		flags[i] = hkStr(f)
	}
	setHK("extra_flags", hkArr(flags))

	hkcfg.Sections["gamescope"] = section

	header := "! HackerOS Gamescope Manager — konfiguracja (.hk)\n" +
		"! Edytuj ręcznie lub przez: gamescope-manager config set <klucz> <wartość>\n" +
		"! Format: https://hackeros-linux-system.github.io/HackerOS-Website/tools-docs/hk.html\n\n"

	content := header + SerializeHK(hkcfg) + "\n"
	return os.WriteFile(ConfigFile, []byte(content), 0644)
}

// SetKey ustawia pojedynczy klucz konfiguracji.
func SetKey(cfg *Config, key, val string) error {
	switch strings.ToLower(key) {
	case "width":
		v, err := strconv.Atoi(val)
		if err != nil || v <= 0 {
			return fmt.Errorf("width musi być dodatnią liczbą całkowitą")
		}
		cfg.Width = v
	case "height":
		v, err := strconv.Atoi(val)
		if err != nil || v <= 0 {
			return fmt.Errorf("height musi być dodatnią liczbą całkowitą")
		}
		cfg.Height = v
	case "refresh":
		v, err := strconv.Atoi(val)
		if err != nil || v <= 0 {
			return fmt.Errorf("refresh musi być dodatnią liczbą całkowitą")
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
			"nieznany klucz: %q\n"+
				"Dostępne: width, height, refresh, fullscreen, vsync, hdr, mangoapp, auto_steam, extra_flags",
			key,
		)
	}
	return nil
}

// Print wyświetla aktualną konfigurację.
func Print(cfg Config) {
	fmt.Printf("  Plik konfiguracyjny: %s\n\n", ConfigFile)
	rows := []struct{ k, v string }{
		{"width",       fmt.Sprintf("%d", cfg.Width)},
		{"height",      fmt.Sprintf("%d", cfg.Height)},
		{"refresh",     fmt.Sprintf("%d Hz", cfg.RefreshRate)},
		{"fullscreen",  boolStr(cfg.Fullscreen)},
		{"vsync",       boolStr(cfg.VSync)},
		{"hdr",         boolStr(cfg.HDR)},
		{"mangoapp",    boolStr(cfg.MangoApp)},
		{"auto_steam",  boolStr(cfg.AutoSteam)},
		{"extra_flags", strings.Join(cfg.ExtraFlags, " ")},
	}
	for _, r := range rows {
		fmt.Printf("  %-16s = %s\n", r.k, r.v)
	}
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes" || s == "on"
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
