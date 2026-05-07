package src

import (
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

func IsRunning() bool {
	data, err := os.ReadFile(PIDFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return false
	}
	if err := syscall.Kill(pid, 0); err != nil {
		_ = os.Remove(PIDFile)
		return false
	}
	return true
}

func ReadPID() string {
	data, err := os.ReadFile(PIDFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func WritePID(pid int) error {
	if err := os.MkdirAll(filepath.Dir(PIDFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(PIDFile, []byte(strconv.Itoa(pid)+"\n"), 0644)
}

func RemovePID() { _ = os.Remove(PIDFile) }

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
