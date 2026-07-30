package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const pidFileName = "gofly.sock"

func readProcessIDs(path string) ([]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	seen := make(map[int]struct{})
	pids := make([]int, 0, 2)
	for _, field := range strings.Split(strings.TrimSpace(string(data)), ",") {
		pid, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil || pid <= 1 {
			continue
		}
		if _, exists := seen[pid]; exists {
			continue
		}
		seen[pid] = struct{}{}
		pids = append(pids, pid)
	}
	if len(pids) == 0 {
		return nil, fmt.Errorf("PID 文件 %s 中没有有效进程", path)
	}
	return pids, nil
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, os.ErrPermission)
}

func isGoFlyProcess(pid int) bool {
	output, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return false
	}
	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 2 || filepath.Base(fields[0]) != "main" {
		return false
	}
	for _, field := range fields[1:] {
		if field == "server" {
			return true
		}
	}
	return false
}

func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !isProcessRunning(pid) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return !isProcessRunning(pid)
}

func stopProcesses(pidPath string) error {
	pids, err := readProcessIDs(pidPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, pid := range pids {
		if !isProcessRunning(pid) {
			continue
		}
		if !isGoFlyProcess(pid) {
			return fmt.Errorf("拒绝停止 PID %d：进程不是 GOFLY server", pid)
		}
		process, findErr := os.FindProcess(pid)
		if findErr != nil {
			return findErr
		}
		if signalErr := process.Signal(syscall.SIGTERM); signalErr != nil && !errors.Is(signalErr, os.ErrProcessDone) {
			return fmt.Errorf("停止 PID %d: %w", pid, signalErr)
		}
	}

	for _, pid := range pids {
		if !waitForProcessExit(pid, 5*time.Second) {
			return fmt.Errorf("等待 PID %d 退出超时", pid)
		}
	}
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func writeProcessIDs(path string, pids ...int) error {
	values := make([]string, 0, len(pids))
	for _, pid := range pids {
		if pid > 1 {
			values = append(values, strconv.Itoa(pid))
		}
	}
	if len(values) == 0 {
		return fmt.Errorf("没有可写入的进程 PID")
	}
	return os.WriteFile(path, []byte(strings.Join(values, ",")), 0644)
}
