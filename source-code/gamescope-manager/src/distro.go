package src

import (
	"fmt"
	"os"
	"strings"
)

const (
	osReleasePath   = "/etc/os-release"
	kcmAboutRcPath  = "/etc/xdg/kcm-about-distrorc"
	requiredName    = "HackerOS"
	requiredVariant = "Gaming Edition"
)

// CheckDistro weryfikuje czy system to HackerOS Gaming Edition.
// Sprawdza dwa pliki:
//   - /etc/os-release              → NAME=HackerOS
//   - /etc/xdg/kcm-about-distrorc  → Variant=Gaming Edition
//
// Jeśli którykolwiek warunek nie jest spełniony — zwraca błąd.
func CheckDistro() error {
	if err := checkOSRelease(); err != nil {
		return err
	}
	if err := checkKCMAboutDistro(); err != nil {
		return err
	}
	return nil
}

func checkOSRelease() error {
	fields, err := parseKeyValueFile(osReleasePath)
	if err != nil {
		return fmt.Errorf(
			"nie można odczytać %s: %v\n"+
				"To narzędzie działa wyłącznie na HackerOS Gaming Edition.",
			osReleasePath, err,
		)
	}

	name, ok := fields["NAME"]
	if !ok {
		return fmt.Errorf(
			"brak pola NAME w %s.\n"+
				"To narzędzie działa wyłącznie na HackerOS Gaming Edition.",
			osReleasePath,
		)
	}

	if name != requiredName {
		return fmt.Errorf(
			"niezgodna dystrybucja: NAME=%q (oczekiwano %q).\n"+
				"gamescope-manager działa wyłącznie na HackerOS Gaming Edition.",
			name, requiredName,
		)
	}

	return nil
}

func checkKCMAboutDistro() error {
	fields, err := parseINIFile(kcmAboutRcPath)
	if err != nil {
		return fmt.Errorf(
			"nie można odczytać %s: %v\n"+
				"To narzędzie działa wyłącznie na HackerOS Gaming Edition.\n"+
				"Upewnij się że masz zainstalowaną edycję Gaming.",
			kcmAboutRcPath, err,
		)
	}

	variant, ok := fields["Variant"]
	if !ok {
		return fmt.Errorf(
			"brak pola Variant= w %s.\n"+
				"To narzędzie działa wyłącznie na HackerOS Gaming Edition.",
			kcmAboutRcPath,
		)
	}

	if variant != requiredVariant {
		return fmt.Errorf(
			"niezgodna edycja: Variant=%q (oczekiwano %q).\n"+
				"gamescope-manager działa wyłącznie na HackerOS Gaming Edition.",
			variant, requiredVariant,
		)
	}

	return nil
}

// parseKeyValueFile parsuje plik KEY=VALUE (format /etc/os-release).
func parseKeyValueFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
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
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		m[key] = val
	}
	return m, nil
}

// parseINIFile parsuje prosty plik INI (ignoruje nagłówki sekcji).
func parseINIFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		m[key] = val
	}
	return m, nil
}
