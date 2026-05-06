package session

import (
	"fmt"
	"os"
	"strings"
)

const stateFile = "/var/lib/hackeros-gaming/current-mode"

// CurrentMode zwraca aktywny tryb: "game-mode" lub "desktop-mode"
func CurrentMode() (string, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "desktop-mode", nil
		}
		return "", fmt.Errorf("nie można odczytać pliku stanu: %w", err)
	}
	mode := strings.TrimSpace(string(data))
	if mode == "" {
		return "desktop-mode", nil
	}
	return mode, nil
}

// SetMode zapisuje aktualny tryb do pliku stanu
func SetMode(mode string) error {
	dir := "/var/lib/hackeros-gaming"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("nie można utworzyć katalogu stanu %q: %w", dir, err)
	}
	if err := os.WriteFile(stateFile, []byte(mode+"\n"), 0644); err != nil {
		return fmt.Errorf("nie można zapisać stanu trybu: %w", err)
	}
	return nil
}
