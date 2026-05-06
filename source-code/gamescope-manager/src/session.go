package src

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	PIDFile = "/var/lib/hackeros-gaming/gamescope-manager.pid"
	LogFile = "/var/log/hackeros-gaming/gamescope-manager.log"
)

// IsRunning sprawdza czy sesja gamescope jest aktywna (PID istnieje i proces żyje).
func IsRunning() bool {
	data, err := os.ReadFile(PIDFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	// Sprawdź czy proces żyje (kill -0)
	if err := syscall.Kill(pid, 0); err != nil {
		// Proces nie istnieje — posprzątaj osierocony PID
		_ = os.Remove(PIDFile)
		return false
	}
	return true
}

// ReadPID zwraca PID aktywnej sesji jako string (lub pusty string).
func ReadPID() string {
	data, err := os.ReadFile(PIDFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// WritePID zapisuje PID sesji do pliku.
func WritePID(pid int) error {
	if err := os.MkdirAll(filepath.Dir(PIDFile), 0755); err != nil {
		return fmt.Errorf("nie można utworzyć katalogu dla PID: %w", err)
	}
	return os.WriteFile(PIDFile, []byte(strconv.Itoa(pid)+"\n"), 0644)
}

// RemovePID usuwa plik PID.
func RemovePID() {
	_ = os.Remove(PIDFile)
}

// GetPIDInt zwraca PID aktywnej sesji jako int lub -1 jeśli brak.
func GetPIDInt() int {
	s := ReadPID()
	if s == "" {
		return -1
	}
	pid, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return pid
}
