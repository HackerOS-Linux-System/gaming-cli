package src

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogEntry zapisuje wpis do pliku logu gamescope-managera.
// Błędy zapisu są cicho ignorowane — log nie jest krytyczny.
func LogEntry(msg string) {
	if err := os.MkdirAll(filepath.Dir(LogFile), 0755); err != nil {
		return
	}
	f, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(f, "[%s] %s\n", ts, msg)
}
