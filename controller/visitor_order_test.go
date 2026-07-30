package controller

import (
	"testing"
	"time"

	"go-fly-muti/ws"
)

func TestSortMapToSliceUsesLatestActivityFirst(t *testing.T) {
	t.Parallel()

	now := time.Now()
	visitors := map[string]*ws.User{
		"visitor-old": {
			Id:         "visitor-old",
			UpdateTime: now.Add(-time.Minute),
		},
		"visitor-new": {
			Id:         "visitor-new",
			UpdateTime: now,
		},
		"visitor-middle": {
			Id:         "visitor-middle",
			UpdateTime: now.Add(-time.Second),
		},
	}

	sorted := sortMapToSlice(visitors)
	got := []string{sorted[0].State().Id, sorted[1].State().Id, sorted[2].State().Id}
	want := []string{"visitor-new", "visitor-middle", "visitor-old"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("position %d = %q, want %q (full order: %v)", index, got[index], want[index], got)
		}
	}
}
