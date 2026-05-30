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
//
// Rozdzielczość: Width=0 lub Height=0 → auto-detekcja przez DRM/xrandr.
// Użytkownik może jawnie ustawić 0 w TUI lub .hk żeby wymusić auto.
type Config struct {
	// --- gamescope ---
	Width          int    // 0 = auto-detekcja
	Height         int    // 0 = auto-detekcja
	RefreshRate    int    // 0 = auto (gamescope wybierze)
	Fullscreen     bool
	VSync          bool   // adaptive-sync / VRR
	HDR            bool
	MangoApp       bool
	AutoSteam      bool
	ExtraFlags     []string
	ForceComposite bool

	// Skalowanie (upscaling) — jak Steam Deck
	ScalingMode   string // "auto" | "integer" | "fit" | "fill" | "stretch"
	ScalingFilter string // "linear" | "pixel" | "fsr" | "nis" | "nearest"
	FSRSharpness  int    // 0-20, dla FSR (AMD FidelityFX Super Resolution)
	NISSharpness  int    // 0-20, dla NIS (NVIDIA Image Scaling, działa na AMD/Intel też)

	// Wewnętrzna rozdzielczość renderowania (może być niższa niż output)
	// 0 = taka sama jak Width/Height
	InternalWidth  int
	InternalHeight int

	// Connector (wyjście wideo) — "" = auto (najlepszy dostępny)
	OutputConnector string // np. "HDMI-A-1", "DP-1", ""

	// Własna lista refreshrate dla VRR (np. "40,50,60")
	CustomRefreshRates string

	// --- steam ---
	SteamBigPicture    bool
	SteamVR            bool
	SteamNoCEFSandbox  bool
	SteamTCP           bool
	SteamBetaChannel   string
	SteamLanguage      string
	SteamExtraFlags    string
	SteamNoVerify      bool
	SteamFullDesktop   bool
	SteamAllowHidCrypt bool
	SteamDisableHWA    bool
}

// Default zwraca domyślną konfigurację.
// Width=0, Height=0 — wymusza auto-detekcję rozdzielczości.
func Default() Config {
	return Config{
		Width:          0, // auto
		Height:         0, // auto
		RefreshRate:    0, // auto
		Fullscreen:     true,
		VSync:          true,
		HDR:            false,
		MangoApp:       true,
		AutoSteam:      true,
		ExtraFlags:     []string{},
		ForceComposite: false,

		ScalingMode:   "auto",
		ScalingFilter: "linear",
		FSRSharpness:  5,
		NISSharpness:  5,

		InternalWidth:  0,
		InternalHeight: 0,
		OutputConnector: "",
		CustomRefreshRates: "",

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

// Load wczytuje konfigurację z pliku .hk, env vars, opcjonalnych overrides CLI.
func Load(overrides ...string) Config {
	cfg := Default()

	hkcfg, err := LoadHKFile(ConfigFile)
	if err == nil {
		applyHKConfig(&cfg, hkcfg)
	}

	applyEnvOverrides(&cfg)

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
		rInt := func(key string, dest *int) {
			if v, err := sec.Get(key); err == nil {
				if n, err := v.AsNumber(); err == nil {
					*dest = int(n)
				}
			}
		}
		rBool := func(key string, dest *bool) {
			if v, err := sec.Get(key); err == nil {
				if b, err := v.AsBool(); err == nil {
					*dest = b
				}
			}
		}
		rStr := func(key string, dest *string) {
			if v, err := sec.Get(key); err == nil {
				if s, err := v.AsString(); err == nil {
					*dest = s
				}
			}
		}

		rInt("width", &cfg.Width)
		rInt("height", &cfg.Height)
		rInt("refresh", &cfg.RefreshRate)
		rBool("fullscreen", &cfg.Fullscreen)
		rBool("vsync", &cfg.VSync)
		rBool("hdr", &cfg.HDR)
		rBool("mangoapp", &cfg.MangoApp)
		rBool("auto_steam", &cfg.AutoSteam)
		rBool("force_composite", &cfg.ForceComposite)
		rStr("scaling_mode", &cfg.ScalingMode)
		rStr("scaling_filter", &cfg.ScalingFilter)
		rInt("fsr_sharpness", &cfg.FSRSharpness)
		rInt("nis_sharpness", &cfg.NISSharpness)
		rInt("internal_width", &cfg.InternalWidth)
		rInt("internal_height", &cfg.InternalHeight)
		rStr("output_connector", &cfg.OutputConnector)
		rStr("custom_refresh_rates", &cfg.CustomRefreshRates)

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
		rBool2 := func(key string, dest *bool) {
			if v, err := st.Get(key); err == nil {
				if b, err := v.AsBool(); err == nil {
					*dest = b
				}
			}
		}
		rStr2 := func(key string, dest *string) {
			if v, err := st.Get(key); err == nil {
				if s, err := v.AsString(); err == nil {
					*dest = s
				}
			}
		}
		rBool2("big_picture", &cfg.SteamBigPicture)
		rBool2("vr", &cfg.SteamVR)
		rBool2("no_cef_sandbox", &cfg.SteamNoCEFSandbox)
		rBool2("tcp_mode", &cfg.SteamTCP)
		rBool2("no_verify", &cfg.SteamNoVerify)
		rBool2("full_desktop_res", &cfg.SteamFullDesktop)
		rBool2("allow_hidcrypto", &cfg.SteamAllowHidCrypt)
		rBool2("disable_hwa", &cfg.SteamDisableHWA)
		rStr2("beta_channel", &cfg.SteamBetaChannel)
		rStr2("language", &cfg.SteamLanguage)
		rStr2("extra_flags", &cfg.SteamExtraFlags)
	}
}

func applyEnvOverrides(cfg *Config) {
	eInt := func(key string, dest *int) {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				*dest = n
			}
		}
	}
	eBool := func(key string, dest *bool) {
		if v := os.Getenv(key); v != "" {
			*dest = parseBool(v)
		}
	}
	eStr := func(key string, dest *string) {
		if v := os.Getenv(key); v != "" {
			*dest = v
		}
	}

	eInt("HACKEROS_GS_WIDTH", &cfg.Width)
	eInt("HACKEROS_GS_HEIGHT", &cfg.Height)
	eInt("HACKEROS_GS_REFRESH", &cfg.RefreshRate)
	eBool("HACKEROS_GS_FULLSCREEN", &cfg.Fullscreen)
	eBool("HACKEROS_GS_VSYNC", &cfg.VSync)
	eBool("HACKEROS_GS_HDR", &cfg.HDR)
	eBool("HACKEROS_GS_MANGOAPP", &cfg.MangoApp)
	eBool("HACKEROS_GS_AUTO_STEAM", &cfg.AutoSteam)
	eBool("HACKEROS_GS_FORCE_COMPOSITE", &cfg.ForceComposite)
	eStr("HACKEROS_GS_SCALING_MODE", &cfg.ScalingMode)
	eStr("HACKEROS_GS_SCALING_FILTER", &cfg.ScalingFilter)
	eStr("HACKEROS_GS_OUTPUT_CONNECTOR", &cfg.OutputConnector)

	eBool("HACKEROS_ST_BIG_PICTURE", &cfg.SteamBigPicture)
	eBool("HACKEROS_ST_VR", &cfg.SteamVR)
	eBool("HACKEROS_ST_NO_CEF_SANDBOX", &cfg.SteamNoCEFSandbox)
	eBool("HACKEROS_ST_TCP", &cfg.SteamTCP)
	eBool("HACKEROS_ST_NO_VERIFY", &cfg.SteamNoVerify)
	eBool("HACKEROS_ST_FULL_DESKTOP", &cfg.SteamFullDesktop)
	eBool("HACKEROS_ST_ALLOW_HIDCRYPTO", &cfg.SteamAllowHidCrypt)
	eBool("HACKEROS_ST_DISABLE_HWA", &cfg.SteamDisableHWA)
	eStr("HACKEROS_ST_BETA_CHANNEL", &cfg.SteamBetaChannel)
	eStr("HACKEROS_ST_LANGUAGE", &cfg.SteamLanguage)
	eStr("HACKEROS_ST_EXTRA_FLAGS", &cfg.SteamExtraFlags)
}

// Save zapisuje konfigurację do pliku .hk.
func Save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(ConfigFile), 0755); err != nil {
		return fmt.Errorf("nie mozna utworzyc katalogu konfiguracji: %w", err)
	}

	hkcfg := &HkConfig{
		Sections:     make(map[string]HkValue),
		SectionOrder: []string{"gamescope", "steam"},
	}

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
	gsSet("scaling_mode", hkStr(cfg.ScalingMode))
	gsSet("scaling_filter", hkStr(cfg.ScalingFilter))
	gsSet("fsr_sharpness", hkNum(float64(cfg.FSRSharpness)))
	gsSet("nis_sharpness", hkNum(float64(cfg.NISSharpness)))
	gsSet("internal_width", hkNum(float64(cfg.InternalWidth)))
	gsSet("internal_height", hkNum(float64(cfg.InternalHeight)))
	gsSet("output_connector", hkStr(cfg.OutputConnector))
	gsSet("custom_refresh_rates", hkStr(cfg.CustomRefreshRates))
	flags := make([]HkValue, len(cfg.ExtraFlags))
	for i, f := range cfg.ExtraFlags {
		flags[i] = hkStr(f)
	}
	gsSet("extra_flags", hkArr(flags))
	hkcfg.Sections["gamescope"] = gs

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

	header := "! HackerOS Gamescope Manager - konfiguracja (.hk) v0.2\n" +
	"! Edytuj recznie lub przez: sudo gamescope-manager config set <klucz> <wartosc>\n" +
	"! Lub przez: gaming-cli tui -> Ustawienia Steam\n" +
	"! width=0 / height=0 = auto-detekcja rozdzielczosci przez DRM\n" +
	"! Format: https://hackeros-linux-system.github.io/HackerOS-Website/tools-docs/hk.html\n\n"

	content := header + SerializeHK(hkcfg) + "\n"
	return os.WriteFile(ConfigFile, []byte(content), 0644)
}

// SetKey ustawia pojedynczy klucz konfiguracji.
func SetKey(cfg *Config, key, val string) error {
	atoiPos := func(s string) (int, error) {
		v, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("oczekiwano liczby calkowitej, dostano: %q", s)
		}
		if v < 0 {
			return 0, fmt.Errorf("wartosc musi byc >= 0")
		}
		return v, nil
	}

	switch strings.ToLower(key) {
		// gamescope
		case "width":
			v, err := atoiPos(val)
			if err != nil {
				return fmt.Errorf("width: %w (0 = auto)", err)
			}
			cfg.Width = v
		case "height":
			v, err := atoiPos(val)
			if err != nil {
				return fmt.Errorf("height: %w (0 = auto)", err)
			}
			cfg.Height = v
		case "refresh":
			v, err := atoiPos(val)
			if err != nil {
				return fmt.Errorf("refresh: %w (0 = auto)", err)
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
		case "scaling_mode":
			valid := map[string]bool{"auto": true, "integer": true, "fit": true, "fill": true, "stretch": true}
			if !valid[val] {
				return fmt.Errorf("scaling_mode: nieprawidlowa wartosc %q (dostepne: auto, integer, fit, fill, stretch)", val)
			}
			cfg.ScalingMode = val
		case "scaling_filter":
			valid := map[string]bool{"linear": true, "pixel": true, "fsr": true, "nis": true, "nearest": true}
			if !valid[val] {
				return fmt.Errorf("scaling_filter: nieprawidlowa wartosc %q (dostepne: linear, pixel, fsr, nis, nearest)", val)
			}
			cfg.ScalingFilter = val
		case "fsr_sharpness":
			v, err := strconv.Atoi(val)
			if err != nil || v < 0 || v > 20 {
				return fmt.Errorf("fsr_sharpness: zakres 0-20")
			}
			cfg.FSRSharpness = v
		case "nis_sharpness":
			v, err := strconv.Atoi(val)
			if err != nil || v < 0 || v > 20 {
				return fmt.Errorf("nis_sharpness: zakres 0-20")
			}
			cfg.NISSharpness = v
		case "internal_width":
			v, err := atoiPos(val)
			if err != nil {
				return err
			}
			cfg.InternalWidth = v
		case "internal_height":
			v, err := atoiPos(val)
			if err != nil {
				return err
			}
			cfg.InternalHeight = v
		case "output_connector":
			cfg.OutputConnector = val
		case "custom_refresh_rates":
			cfg.CustomRefreshRates = val
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
				"  mangoapp, auto_steam, force_composite, extra_flags,\n"+
				"  scaling_mode, scaling_filter, fsr_sharpness, nis_sharpness,\n"+
				"  internal_width, internal_height, output_connector, custom_refresh_rates\n"+
				"Dostepne (steam): big_picture, vr, no_cef_sandbox, tcp_mode,\n"+
				"  no_verify, full_desktop_res, allow_hidcrypto, disable_hwa,\n"+
				"  beta_channel, language, steam_extra_flags",
		     key,
			)
	}
	return nil
}

// Print wyświetla aktualną konfigurację.
func Print(cfg Config) {
	fmt.Printf("  Plik konfiguracyjny: %s\n\n", ConfigFile)

	widthStr := autoOrVal(cfg.Width)
	heightStr := autoOrVal(cfg.Height)
	refreshStr := autoOrVal(cfg.RefreshRate)

	fmt.Printf("  [gamescope]\n")
	gsRows := []struct{ k, v string }{
		{"width", widthStr},
		{"height", heightStr},
		{"refresh", refreshStr},
		{"fullscreen", boolStr(cfg.Fullscreen)},
		{"vsync", boolStr(cfg.VSync)},
		{"hdr", boolStr(cfg.HDR)},
		{"mangoapp", boolStr(cfg.MangoApp)},
		{"auto_steam", boolStr(cfg.AutoSteam)},
		{"force_composite", boolStr(cfg.ForceComposite)},
		{"scaling_mode", cfg.ScalingMode},
		{"scaling_filter", cfg.ScalingFilter},
		{"fsr_sharpness", strconv.Itoa(cfg.FSRSharpness)},
		{"nis_sharpness", strconv.Itoa(cfg.NISSharpness)},
		{"internal_width", autoOrVal(cfg.InternalWidth)},
		{"internal_height", autoOrVal(cfg.InternalHeight)},
		{"output_connector", orDefault(cfg.OutputConnector, "(auto)")},
		{"custom_refresh_rates", orDefault(cfg.CustomRefreshRates, "(brak)")},
		{"extra_flags", strings.Join(cfg.ExtraFlags, " ")},
	}
	for _, r := range gsRows {
		fmt.Printf("  %-22s = %s\n", r.k, r.v)
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
		fmt.Printf("  %-22s = %s\n", r.k, r.v)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

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

func autoOrVal(v int) string {
	if v == 0 {
		return "0 (auto)"
	}
	return strconv.Itoa(v)
}
