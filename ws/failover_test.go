package ws

import (
	"testing"

	"go-fly-muti/models"
)

func TestAvailableFailoverKefus(t *testing.T) {
	users := []models.User{
		{Name: "WGG", OnlineStatus: 1, RecNum: 1},
		{Name: "WGG1", OnlineStatus: 1, RecNum: 2},
		{Name: "WGG2", OnlineStatus: 1, RecNum: 3},
		{Name: "WGG3", OnlineStatus: 0},
	}
	online := map[string]bool{"WGG": true, "WGG1": true, "WGG2": true, "WGG3": true}
	available := availableFailoverKefus(
		users,
		"WGG",
		func(name string) bool { return online[name] },
		map[string]int{"WGG1": 4, "WGG2": 1},
	)
	if len(available) != 2 {
		t.Fatalf("availableFailoverKefus() length = %d, want 2", len(available))
	}
	if available[0].Name != "WGG2" || available[1].Name != "WGG1" {
		t.Fatalf("availableFailoverKefus() order = %q, %q; want WGG2, WGG1", available[0].Name, available[1].Name)
	}
}
