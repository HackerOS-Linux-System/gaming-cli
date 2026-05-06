package src

import (
	"fmt"
	"os"
	"strings"
)

const stateFile = "/var/lib/hackeros-gaming/current-mode"

func CurrentMode() (string, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "desktop-mode", nil
		}
		return "", fmt.Errorf("nie można odczytać pliku stanu: %w", err)
	}
	m := strings.TrimSpace(string(data))
	if m == "" {
		return "desktop-mode", nil
	}
	return m, nil
}

func SetMode(mode string) error {
	if err := os.MkdirAll("/var/lib/hackeros-gaming", 0755); err != nil {
		return err
	}
	return os.WriteFile(stateFile, []byte(mode+"\n"), 0644)
}
