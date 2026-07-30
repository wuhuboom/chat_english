package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

func TestWaitForShutdownSignal(t *testing.T) {
	stopSign := make(chan int, 1)
	signals := make(chan os.Signal, 1)
	serverErrors := make(chan error, 1)
	signals <- syscall.SIGTERM

	signal, err := waitForShutdown(stopSign, signals, serverErrors)
	if err != nil {
		t.Fatalf("waitForShutdown() error = %v", err)
	}
	if signal != syscall.SIGTERM {
		t.Fatalf("waitForShutdown() = %v, want SIGTERM", signal)
	}
}

func TestWaitForLegacyStopSignal(t *testing.T) {
	stopSign := make(chan int, 1)
	signals := make(chan os.Signal, 1)
	serverErrors := make(chan error, 1)
	stopSign <- 1

	signal, err := waitForShutdown(stopSign, signals, serverErrors)
	if err != nil {
		t.Fatalf("waitForShutdown() error = %v", err)
	}
	if signal != nil {
		t.Fatalf("waitForShutdown() = %v, want nil for legacy stop", signal)
	}
}

func TestWaitForServerError(t *testing.T) {
	stopSign := make(chan int, 1)
	signals := make(chan os.Signal, 1)
	serverErrors := make(chan error, 1)
	wantErr := errors.New("address already in use")
	serverErrors <- wantErr

	signal, err := waitForShutdown(stopSign, signals, serverErrors)
	if signal != nil {
		t.Fatalf("waitForShutdown() signal = %v, want nil", signal)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("waitForShutdown() error = %v, want %v", err, wantErr)
	}
}

func TestReadAndWriteProcessIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gofly.sock")
	if err := writeProcessIDs(path, 100, 200, 200); err != nil {
		t.Fatalf("writeProcessIDs() error = %v", err)
	}
	got, err := readProcessIDs(path)
	if err != nil {
		t.Fatalf("readProcessIDs() error = %v", err)
	}
	want := []int{100, 200}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readProcessIDs() = %v, want %v", got, want)
	}
}

func TestReadProcessIDsRejectsInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gofly.sock")
	if err := os.WriteFile(path, []byte("bad,0,1"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readProcessIDs(path); err == nil {
		t.Fatal("readProcessIDs() error = nil, want invalid PID error")
	}
}
