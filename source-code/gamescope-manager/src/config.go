package src

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const ConfigFile = "/etc/hackeros/gamescope-manager.hk"

// Config przechowuje pełną konfigurację sesji gamescope + Steam.
type Config struct {
	// --- gamescope ---
	Width       int
	Height      int
	RefreshRate int
	Fullscreen  bool
	VSync       bool
	HDR         bool
	MangoApp    bool
	AutoSteam   bool
	ExtraFlags  []string
	ForceComposite bool

	// --- steam ---
	SteamBigPicture    bool   // -tenfoot
	SteamVR            bool   // -vr
	SteamNoCEFSandbox  bool   // -no-cef-sandbox
	SteamTCP           bool   // -nosharedmemory (fallback TCP)
	SteamBetaChannel   string // "" | "beta" | "steamdeckbeta" | "beta-client"
	SteamLanguage      string // "" = systemowy, "polish", "english", itd.
	SteamExtraFlags    string // dowolne dodatkowe flagi przekazywane do steam
	SteamNoVerify      bool   // -noverifyfiles
	SteamFullDesktop   bool   // -fulldesktopres
	SteamAllowHidCrypt bool   // -allow-hidcrypto
	SteamDisableHWA    bool   // -disable-gpu (wyłącz hardware acceleration w UI)
}

// Default zwraca domyślną konfigurację.
func Default() Config {
	return Config{
		Width:           1920,
		Height:          1080,
		RefreshRate:     60,
		Fullscreen:      true,
		VSync:           true,
		HDR:             false,
		MangoApp:        true,
		AutoSteam:       true,
		ExtraFlags:      []string{},
		ForceComposite:  false,

		SteamBigPicture:    true,
		SteamVR:            false,
		SteamNoCEFSandbox:  true,
		SteamTCP:           false,
		SteamBetaChannel:   "",
		SteamLanguage:      "",
		SteamExtraFlags:    "",
		SteamNoVerify:      false,
		SteamFullDesktop:   false,
		SteamAllowHidCrypt: false,
		SteamDisableHWA:    false,
	}
}

// Load wczytuje konfigurację z pliku .hk, następnie nadpisuje wartościami
// ze zmiennych środowiskowych (HACKEROS_GS_*), a potem z argumentów CLI
// przekazanych przez overrides (klucz=wartość).
func Load(overrides ...string) Config {
	cfg := Default()

	// 1. Plik .hk
	hkcfg, err := LoadHKFile(ConfigFile)
	if err == nil {
		applyHKConfig(&cfg, hkcfg)
	}

	// 2. Zmienne środowiskowe HACKEROS_GS_*
	applyEnvOverrides(&cfg)

	// 3. Argumenty CLI (klucz=wartość)
	for _, ov := range overrides {
		parts := strings.SplitN(ov, "=", 2)
		if len(parts) == 2 {
			_ = SetKey(&cfg, parts[0], parts[1])
		}
	}

	return cfg
}

func applyHKConfig(cfg *Config, hkcfg *HkConfig) {
	sec, ok := hkcfg.Sections["gamescope"]
	if ok && sec.Type == HkMap {
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
		readBool("force_composite", &cfg.ForceComposite)

		if v, err := sec.Get("extra_flags"); err == nil {
			if v.Type == HkArray {
				for _, item := range v.Array {
					if s, e := item.AsString(); e == nil {
						cfg.ExtraFlags = append(cfg.ExtraFlags, s)
					}
				}
			} else if s, e := v.AsString(); e == nil && s != "" {
				cfg.ExtraFlags = strings.Fields(s)
			}
		}
	}

	st, ok := hkcfg.Sections["steam"]
	if ok && st.Type == HkMap {
		readBool2 := func(key string, dest *bool) {
			if v, err := st.Get(key); err == nil {
				if b, err := v.AsBool(); err == nil {
					*dest = b
				}
			}
		}
		readStr := func(key string, dest *string) {
			if v, err := st.Get(key); err == nil {
				if s, err := v.AsString(); err == nil {
					*dest = s
				}
			}
		}
		readBool2("big_picture", &cfg.SteamBigPicture)
		readBool2("vr", &cfg.SteamVR)
		readBool2("no_cef_sandbox", &cfg.SteamNoCEFSandbox)
		readBool2("tcp_mode", &cfg.SteamTCP)
		readBool2("no_verify", &cfg.SteamNoVerify)
		readBool2("full_desktop_res", &cfg.SteamFullDesktop)
		readBool2("allow_hidcrypto", &cfg.SteamAllowHidCrypt)
		readBool2("disable_hwa", &cfg.SteamDisableHWA)
		readStr("beta_channel", &cfg.SteamBetaChannel)
		readStr("language", &cfg.SteamLanguage)
		readStr("extra_flags", &cfg.SteamExtraFlags)
	}
}

// applyEnvOverrides nadpisuje konfigurację zmiennymi środowiskowymi.
// Prefiks: HACKEROS_GS_ (gamescope) oraz HACKEROS_ST_ (steam).
func applyEnvOverrides(cfg *Config) {
	envInt := func(key string, dest *int) {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dest = n
			}
		}
	}
	envBool := func(key string, dest *bool) {
		if v := os.Getenv(key); v != "" {
			*dest = parseBool(v)
		}
	}
	envStr := func(key string, dest *string) {
		if v := os.Getenv(key); v != "" {
			*dest = v
		}
	}

	envInt("HACKEROS_GS_WIDTH", &cfg.Width)
	envInt("HACKEROS_GS_HEIGHT", &cfg.Height)
	envInt("HACKEROS_GS_REFRESH", &cfg.RefreshRate)
	envBool("HACKEROS_GS_FULLSCREEN", &cfg.Fullscreen)
	envBool("HACKEROS_GS_VSYNC", &cfg.VSync)
	envBool("HACKEROS_GS_HDR", &cfg.HDR)
	envBool("HACKEROS_GS_MANGOAPP", &cfg.MangoApp)
	envBool("HACKEROS_GS_AUTO_STEAM", &cfg.AutoSteam)
	envBool("HACKEROS_GS_FORCE_COMPOSITE", &cfg.ForceComposite)

	envBool("HACKEROS_ST_BIG_PICTURE", &cfg.SteamBigPicture)
	envBool("HACKEROS_ST_VR", &cfg.SteamVR)
	envBool("HACKEROS_ST_NO_CEF_SANDBOX", &cfg.SteamNoCEFSandbox)
	envBool("HACKEROS_ST_TCP", &cfg.SteamTCP)
	envBool("HACKEROS_ST_NO_VERIFY", &cfg.SteamNoVerify)
	envBool("HACKEROS_ST_FULL_DESKTOP", &cfg.SteamFullDesktop)
	envBool("HACKEROS_ST_ALLOW_HIDCRYPTO", &cfg.SteamAllowHidCrypt)
	envBool("HACKEROS_ST_DISABLE_HWA", &cfg.SteamDisableHWA)
	envStr("HACKEROS_ST_BETA_CHANNEL", &cfg.SteamBetaChannel)
	envStr("HACKEROS_ST_LANGUAGE", &cfg.SteamLanguage)
	envStr("HACKEROS_ST_EXTRA_FLAGS", &cfg.SteamExtraFlags)
}

// Save zapisuje konfigurację do pliku .hk.
func Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(ConfigFile), 0755); err != nil {
		return fmt.Errorf("nie można utworzyć katalogu konfiguracji: %w", err)
	}

	hkcfg := &HkConfig{
		Sections:     make(map[string]HkValue),
		SectionOrder: []string{"gamescope", "steam"},
	}

	// sekcja [gamescope]
	gs := hkMap()
	gsSet := func(key string, val HkValue) {
		gs.Map[key] = val
		gs.MapKeys = append(gs.MapKeys, key)
	}
	gsSet("width", hkNum(float64(cfg.Width)))
	gsSet("height", hkNum(float64(cfg.Height)))
	gsSet("refresh", hkNum(float64(cfg.RefreshRate)))
	gsSet("fullscreen", hkBool(cfg.Fullscreen))
	gsSet("vsync", hkBool(cfg.VSync))
	gsSet("hdr", hkBool(cfg.HDR))
	gsSet("mangoapp", hkBool(cfg.MangoApp))
	gsSet("auto_steam", hkBool(cfg.AutoSteam))
	gsSet("force_composite", hkBool(cfg.ForceComposite))
	flags := make([]HkValue, len(cfg.ExtraFlags))
	for i, f := range cfg.ExtraFlags {
		flags[i] = hkStr(f)
	}
	gsSet("extra_flags", hkArr(flags))
	hkcfg.Sections["gamescope"] = gs

	// sekcja [steam]
	st := hkMap()
	stSet := func(key string, val HkValue) {
		st.Map[key] = val
		st.MapKeys = append(st.MapKeys, key)
	}
	stSet("big_picture", hkBool(cfg.SteamBigPicture))
	stSet("vr", hkBool(cfg.SteamVR))
	stSet("no_cef_sandbox", hkBool(cfg.SteamNoCEFSandbox))
	stSet("tcp_mode", hkBool(cfg.SteamTCP))
	stSet("no_verify", hkBool(cfg.SteamNoVerify))
	stSet("full_desktop_res", hkBool(cfg.SteamFullDesktop))
	stSet("allow_hidcrypto", hkBool(cfg.SteamAllowHidCrypt))
	stSet("disable_hwa", hkBool(cfg.SteamDisableHWA))
	stSet("beta_channel", hkStr(cfg.SteamBetaChannel))
	stSet("language", hkStr(cfg.SteamLanguage))
	stSet("extra_flags", hkStr(cfg.SteamExtraFlags))
	hkcfg.Sections["steam"] = st

	header := "! HackerOS Gamescope Manager - konfiguracja (.hk)\n" +
	"! Edytuj recznie lub przez: gamescope-manager config set <klucz> <wartosc>\n" +
	"! Lub przez gaming-cli TUI -> Ustawienia Steam\n" +
	"! Format: https://hackeros-linux-system.github.io/HackerOS-Website/tools-docs/hk.html\n\n"

	content := header + SerializeHK(hkcfg) + "\n"
	return os.WriteFile(ConfigFile, []byte(content), 0644)
}

// SetKey ustawia pojedynczy klucz konfiguracji (obie sekcje).
func SetKey(cfg *Config, key, val string) error {
	switch strings.ToLower(key) {
		// gamescope
		case "width":
			v, err := strconv.Atoi(val)
			if err != nil || v <= 0 {
				return fmt.Errorf("width musi byc dodatnia liczba calkowita")
			}
			cfg.Width = v
		case "height":
			v, err := strconv.Atoi(val)
			if err != nil || v <= 0 {
				return fmt.Errorf("height musi byc dodatnia liczba calkowita")
			}
			cfg.Height = v
		case "refresh":
			v, err := strconv.Atoi(val)
			if err != nil || v <= 0 {
				return fmt.Errorf("refresh musi byc dodatnia liczba calkowita")
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
		case "force_composite":
			cfg.ForceComposite = parseBool(val)
		case "extra_flags":
			cfg.ExtraFlags = strings.Fields(val)
			// steam
		case "big_picture", "steam.big_picture":
			cfg.SteamBigPicture = parseBool(val)
		case "vr", "steam.vr":
			cfg.SteamVR = parseBool(val)
		case "no_cef_sandbox", "steam.no_cef_sandbox":
			cfg.SteamNoCEFSandbox = parseBool(val)
		case "tcp_mode", "steam.tcp_mode":
			cfg.SteamTCP = parseBool(val)
		case "no_verify", "steam.no_verify":
			cfg.SteamNoVerify = parseBool(val)
		case "full_desktop_res", "steam.full_desktop_res":
			cfg.SteamFullDesktop = parseBool(val)
		case "allow_hidcrypto", "steam.allow_hidcrypto":
			cfg.SteamAllowHidCrypt = parseBool(val)
		case "disable_hwa", "steam.disable_hwa":
			cfg.SteamDisableHWA = parseBool(val)
		case "beta_channel", "steam.beta_channel":
			cfg.SteamBetaChannel = val
		case "language", "steam.language":
			cfg.SteamLanguage = val
		case "steam_extra_flags", "steam.extra_flags":
			cfg.SteamExtraFlags = val
		default:
			return fmt.Errorf(
				"nieznany klucz: %q\n"+
				"Dostepne (gamescope): width, height, refresh, fullscreen, vsync, hdr,\n"+
				"                      mangoapp, auto_steam, force_composite, extra_flags\n"+
				"Dostepne (steam):     big_picture, vr, no_cef_sandbox, tcp_mode,\n"+
				"                      no_verify, full_desktop_res, allow_hidcrypto,\n"+
				"                      disable_hwa, beta_channel, language, steam_extra_flags",
		     key,
			)
	}
	return nil
}

// Print wyswietla aktualna konfiguracje.
func Print(cfg Config) {
	fmt.Printf("  Plik konfiguracyjny: %s\n\n", ConfigFile)
	fmt.Printf("  [gamescope]\n")
	gsRows := []struct{ k, v string }{
		{"width", fmt.Sprintf("%d", cfg.Width)},
		{"height", fmt.Sprintf("%d", cfg.Height)},
		{"refresh", fmt.Sprintf("%d Hz", cfg.RefreshRate)},
		{"fullscreen", boolStr(cfg.Fullscreen)},
		{"vsync", boolStr(cfg.VSync)},
		{"hdr", boolStr(cfg.HDR)},
		{"mangoapp", boolStr(cfg.MangoApp)},
		{"auto_steam", boolStr(cfg.AutoSteam)},
		{"force_composite", boolStr(cfg.ForceComposite)},
		{"extra_flags", strings.Join(cfg.ExtraFlags, " ")},
	}
	for _, r := range gsRows {
		fmt.Printf("  %-20s = %s\n", r.k, r.v)
	}
	fmt.Printf("\n  [steam]\n")
	stRows := []struct{ k, v string }{
		{"big_picture", boolStr(cfg.SteamBigPicture)},
		{"vr", boolStr(cfg.SteamVR)},
		{"no_cef_sandbox", boolStr(cfg.SteamNoCEFSandbox)},
		{"tcp_mode", boolStr(cfg.SteamTCP)},
		{"no_verify", boolStr(cfg.SteamNoVerify)},
		{"full_desktop_res", boolStr(cfg.SteamFullDesktop)},
		{"allow_hidcrypto", boolStr(cfg.SteamAllowHidCrypt)},
		{"disable_hwa", boolStr(cfg.SteamDisableHWA)},
		{"beta_channel", orDefault(cfg.SteamBetaChannel, "(brak)")},
		{"language", orDefault(cfg.SteamLanguage, "(systemowy)")},
		{"steam_extra_flags", orDefault(cfg.SteamExtraFlags, "(brak)")},
	}
	for _, r := range stRows {
		fmt.Printf("  %-20s = %s\n", r.k, r.v)
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

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
