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

func CheckDistro() error {
	if err := checkOSRelease(); err != nil {
		return err
	}
	return checkKCMAboutDistro()
}

func checkOSRelease() error {
	fields, err := parseKeyValueFile(osReleasePath)
	if err != nil {
		return fmt.Errorf("nie można odczytać %s: %v\nTo narzędzie działa wyłącznie na HackerOS Gaming Edition.", osReleasePath, err)
	}
	name := fields["NAME"]
	if name != requiredName {
		return fmt.Errorf("niezgodna dystrybucja: NAME=%q (oczekiwano %q).\ngamescope-manager działa wyłącznie na HackerOS Gaming Edition.", name, requiredName)
	}
	return nil
}

func checkKCMAboutDistro() error {
	fields, err := parseINIFile(kcmAboutRcPath)
	if err != nil {
		return fmt.Errorf("nie można odczytać %s: %v\nTo narzędzie działa wyłącznie na HackerOS Gaming Edition.", kcmAboutRcPath, err)
	}
	variant := fields["Variant"]
	if variant != requiredVariant {
		return fmt.Errorf("niezgodna edycja: Variant=%q (oczekiwano %q).\ngamescope-manager działa wyłącznie na HackerOS Gaming Edition.", variant, requiredVariant)
	}
	return nil
}

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
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		}
	}
	return m, nil
}

func parseINIFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "[") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		}
	}
	return m, nil
}
