package controller

import (
	"testing"

	"go-fly-muti/models"
)

func TestSelectAvailableKefu(t *testing.T) {
	users := []models.User{
		{Name: "WGG", OnlineStatus: 1, RecNum: 3},
		{Name: "WGG1", OnlineStatus: 1, RecNum: 2},
		{Name: "WGG2", OnlineStatus: 1, RecNum: 1},
	}

	tests := []struct {
		name       string
		preferred  []string
		online     map[string]bool
		loads      map[string]int
		want       string
		wantOnline bool
		wantReason string
	}{
		{
			name:       "requested online agent remains sticky",
			preferred:  []string{"WGG1"},
			online:     map[string]bool{"WGG1": true, "WGG2": true},
			loads:      map[string]int{"WGG1": 5, "WGG2": 0},
			want:       "WGG1",
			wantOnline: true,
			wantReason: "preferred",
		},
		{
			name:       "offline requested agent falls through to previous online agent",
			preferred:  []string{"WGG1", "WGG2"},
			online:     map[string]bool{"WGG2": true},
			want:       "WGG2",
			wantOnline: true,
			wantReason: "preferred",
		},
		{
			name:       "offline preferred agents use least active load",
			preferred:  []string{"WGG"},
			online:     map[string]bool{"WGG1": true, "WGG2": true},
			loads:      map[string]int{"WGG1": 4, "WGG2": 1},
			want:       "WGG2",
			wantOnline: true,
			wantReason: "least_loaded",
		},
		{
			name:       "manual offline state excludes websocket connection",
			online:     map[string]bool{"WGG": true},
			wantOnline: false,
			wantReason: "offline",
		},
	}

	usersWithManualOffline := append([]models.User(nil), users...)
	usersWithManualOffline[0].OnlineStatus = 0
	usersWithManualOffline[1].OnlineStatus = 0
	usersWithManualOffline[2].OnlineStatus = 0

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testUsers := users
			if test.name == "manual offline state excludes websocket connection" {
				testUsers = usersWithManualOffline
			}
			got, gotOnline, gotReason := selectAvailableKefu(
				testUsers,
				test.preferred,
				func(name string) bool { return test.online[name] },
				test.loads,
			)
			if got.Name != test.want || gotOnline != test.wantOnline || gotReason != test.wantReason {
				t.Fatalf("selectAvailableKefu() = (%q, %v, %q), want (%q, %v, %q)",
					got.Name, gotOnline, gotReason, test.want, test.wantOnline, test.wantReason)
			}
		})
	}
}

func TestFallbackKefuStaysInsideEnterprise(t *testing.T) {
	ent := models.User{Name: "WGG"}
	users := []models.User{
		ent,
		{Name: "WGG1"},
	}

	if got := fallbackKefu(users, []string{"foreign-agent", "WGG1"}, ent); got.Name != "WGG1" {
		t.Fatalf("fallbackKefu() = %q, want WGG1", got.Name)
	}
	if got := fallbackKefu(users, []string{"foreign-agent"}, ent); got.Name != "WGG" {
		t.Fatalf("fallbackKefu() = %q, want WGG", got.Name)
	}
}
