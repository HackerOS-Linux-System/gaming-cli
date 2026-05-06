package src

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// PrintInfo wyświetla informacje o środowisku gaming.
func PrintInfo() {
	kernel := runCmd("uname", "-r")
	prettyName := readOSReleaseField("PRETTY_NAME")
	versionID := readOSReleaseField("VERSION_ID")

	gsVersion := gamescopeVersion()
	steamStatus := steamInfo()
	currentMode, _ := CurrentMode()
	gpu := detectGPU()
	debianVer := debianVersion()

	fmt.Println()
	fmt.Println("  Środowisko HackerOS Gaming Edition")
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Dystrybucja  : %s %s\n", prettyName, versionID)
	fmt.Printf("  Debian       : %s\n", debianVer)
	fmt.Printf("  Jądro        : %s\n", kernel)
	fmt.Printf("  GPU          : %s\n", gpu)
	fmt.Printf("  Gamescope    : %s\n", gsVersion)
	fmt.Printf("  Steam        : %s\n", steamStatus)
	fmt.Printf("  Tryb         : %s\n", currentMode)
	fmt.Println("  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

func gamescopeVersion() string {
	p, err := exec.LookPath("gamescope")
	if err != nil {
		return "NIE ZAINSTALOWANY"
	}
	out, err := exec.Command(p, "--version").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		return fmt.Sprintf("zainstalowany (%s)", p)
	}
	return strings.TrimSpace(string(out))
}

func steamInfo() string {
	for _, name := range []string{"steam", "steam-runtime", "steam-native"} {
		if p, err := exec.LookPath(name); err == nil {
			return fmt.Sprintf("zainstalowany (%s)", p)
		}
	}
	return "NIE ZAINSTALOWANY"
}

func detectGPU() string {
	out, err := exec.Command("lspci").Output()
	if err != nil {
		return "nieznane (brak lspci)"
	}
	var gpus []string
	for _, line := range strings.Split(string(out), "\n") {
		lo := strings.ToLower(line)
		if strings.Contains(lo, "vga") || strings.Contains(lo, "3d controller") || strings.Contains(lo, "display controller") {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				gpus = append(gpus, strings.TrimSpace(parts[1]))
			}
		}
	}
	if len(gpus) == 0 {
		return "nieznane"
	}
	return strings.Join(gpus, " | ")
}

func debianVersion() string {
	data, err := os.ReadFile("/etc/debian_version")
	if err != nil {
		return "nieznane"
	}
	return strings.TrimSpace(string(data))
}

func readOSReleaseField(key string) string {
	fields, err := parseKeyValueFile(osReleasePath)
	if err != nil {
		return "nieznane"
	}
	v, ok := fields[key]
	if !ok {
		return "nieznane"
	}
	return v
}

func runCmd(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "nieznane"
	}
	return strings.TrimSpace(string(out))
}
