package src

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type EnvInfo struct {
	PrettyName  string
	DebianVer   string
	Kernel      string
	GPU         string
	Gamescope   string
	Steam       string
	CurrentMode string
}

func GatherInfo() EnvInfo {
	mode, _ := CurrentMode()
	return EnvInfo{
		PrettyName:  readOSField("PRETTY_NAME"),
		DebianVer:   readFile("/etc/debian_version"),
		Kernel:      runCmd("uname", "-r"),
		GPU:         detectGPU(),
		Gamescope:   gamescopeVer(),
		Steam:       steamStatus(),
		CurrentMode: mode,
	}
}

func readOSField(key string) string {
	f, _ := parseKeyValueFile(osReleasePath)
	v := f[key]
	if v == "" {
		return "nieznane"
	}
	return v
}

func readFile(path string) string {
	d, err := os.ReadFile(path)
	if err != nil {
		return "nieznane"
	}
	return strings.TrimSpace(string(d))
}

func runCmd(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "nieznane"
	}
	return strings.TrimSpace(string(out))
}

func gamescopeVer() string {
	p, err := exec.LookPath("gamescope")
	if err != nil {
		if _, se := os.Stat("/usr/bin/gamescope"); se == nil {
			p = "/usr/bin/gamescope"
		} else {
			return "NIE ZAINSTALOWANY"
		}
	}
	out, _ := exec.Command(p, "--version").Output()
	v := strings.TrimSpace(string(out))
	if v == "" {
		return fmt.Sprintf("zainstalowany (%s)", p)
	}
	return v
}

func steamStatus() string {
	for _, n := range []string{"steam", "steam-runtime", "steam-native"} {
		if p, err := exec.LookPath(n); err == nil {
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
