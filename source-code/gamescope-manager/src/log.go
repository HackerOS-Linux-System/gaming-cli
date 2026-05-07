package src

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func LogEntry(msg string) {
	_ = os.MkdirAll(filepath.Dir(LogFile), 0755)
	f, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
}
