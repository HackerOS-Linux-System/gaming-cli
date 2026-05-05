package distro

import (
	"fmt"
	"os"
	"strings"
)

// Check weryfikuje czy system to HackerOS Gaming Edition na Debianie Testing (Forky).
// Narzędzie NIE jest przeznaczone dla handheldów — tylko PC i laptopy.
func Check() error {
	// Odczytaj /etc/os-release
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return fmt.Errorf("nie można odczytać /etc/os-release: %w", err)
	}

	fields := parseOSRelease(string(data))

	// Sprawdź czy to HackerOS
	id := strings.ToLower(fields["ID"])
	idLike := strings.ToLower(fields["ID_LIKE"])
	prettyName := strings.ToLower(fields["PRETTY_NAME"])

	isHackerOS := strings.Contains(id, "hackeros") ||
		strings.Contains(idLike, "hackeros") ||
		strings.Contains(prettyName, "hackeros")

	if !isHackerOS {
		return fmt.Errorf(
			"niezgodna dystrybucja: %q\n"+
				"gaming-cli działa wyłącznie na HackerOS Gaming Edition (Debian Testing).\n"+
				"Handheldowe urządzenia nie są obsługiwane.",
			fields["PRETTY_NAME"],
		)
	}

	// Sprawdź czy to edycja Gaming (plik znacznikowy tworzony przez instalator)
	if _, err := os.Stat("/etc/hackeros-gaming-edition"); os.IsNotExist(err) {
		// Miękkie ostrzeżenie — nie blokuj, bo podczas development może nie istnieć
		fmt.Fprintln(os.Stderr, "[gaming-cli] OSTRZEŻENIE: Brak /etc/hackeros-gaming-edition — upewnij się że używasz Gaming Edition.")
	}

	// Sprawdź czy to Debian Testing (Forky) — przez /etc/debian_version
	debVer, _ := readFile("/etc/debian_version")
	debVer = strings.TrimSpace(debVer)
	// Debian Testing ma "trixie/sid" lub "forky/sid" lub po prostu "testing"
	isDebianTesting := strings.Contains(strings.ToLower(debVer), "forky") ||
		strings.Contains(strings.ToLower(debVer), "testing") ||
		strings.Contains(strings.ToLower(debVer), "sid") ||
		strings.Contains(strings.ToLower(debVer), "trixie")

	if !isDebianTesting {
		fmt.Fprintf(os.Stderr,
			"[gaming-cli] OSTRZEŻENIE: Wykryto Debian %q — gaming-cli jest przeznaczony dla Debian Testing (Forky).\n",
			debVer,
		)
	}

	return nil
}

func parseOSRelease(content string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
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
	return m
}

func readFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
