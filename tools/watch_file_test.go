package tools

import (
	"github.com/fsnotify/fsnotify"
	"testing"
)

func TestWatchDirectoryCanStartAndStop(t *testing.T) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	w := Watch{watch: watcher}
	w.watchDir(t.TempDir())
	if err := watcher.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
