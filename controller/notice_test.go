package controller

import (
	"testing"

	"go-fly-muti/models"
)

func TestWelcomeVersionChangesWithContent(t *testing.T) {
	welcomes := []models.Welcome{
		{ID: 1, UserId: "WGG1", Keyword: "welcome", Content: "欢迎咨询", DelaySecond: 1},
	}
	original := welcomeVersion(welcomes)
	welcomes[0].Content = "最新欢迎语"

	if updated := welcomeVersion(welcomes); updated == original {
		t.Fatalf("welcomeVersion() did not change after content update")
	}
}

func TestWelcomeVersionIsStableAcrossQueryOrder(t *testing.T) {
	first := models.Welcome{ID: 1, UserId: "WGG1", Keyword: "welcome", Content: "第一句", DelaySecond: 1}
	second := models.Welcome{ID: 2, UserId: "WGG1", Keyword: "welcome", Content: "第二句", DelaySecond: 2}

	if got, want := welcomeVersion([]models.Welcome{second, first}), welcomeVersion([]models.Welcome{first, second}); got != want {
		t.Fatalf("welcomeVersion() depends on query order: got %q, want %q", got, want)
	}
}

func TestWelcomesForVersion(t *testing.T) {
	welcomes := []models.Welcome{{ID: 1, Content: "欢迎咨询"}}
	version := welcomeVersion(welcomes)

	if got := welcomesForVersion(welcomes, version); len(got) != 0 {
		t.Fatalf("welcomesForVersion() returned %d unchanged welcomes, want 0", len(got))
	}
	if got := welcomesForVersion(welcomes, "old-version"); len(got) != len(welcomes) {
		t.Fatalf("welcomesForVersion() returned %d changed welcomes, want %d", len(got), len(welcomes))
	}
}
