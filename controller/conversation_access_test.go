package controller

import (
	"testing"

	"go-fly-muti/models"
)

func TestUserBelongsToEnterprise(t *testing.T) {
	if !userBelongsToEnterprise(models.User{ID: 1}, "1") {
		t.Fatal("enterprise owner should belong to itself")
	}
	if !userBelongsToEnterprise(models.User{ID: 2, Pid: 1}, "1") {
		t.Fatal("sub-agent should belong to its parent enterprise")
	}
	if userBelongsToEnterprise(models.User{ID: 9, Pid: 8}, "1") {
		t.Fatal("foreign agent must not belong to enterprise")
	}
}

func TestValidateKefuConversation(t *testing.T) {
	visitor := models.Visitor{EntId: "1", ToId: "WGG1"}
	visitor.ID = 10

	if err := validateKefuConversation("1", "WGG1", visitor); err != nil {
		t.Fatalf("valid assignment rejected: %v", err)
	}
	if err := validateKefuConversation("1", "WGG2", visitor); err == nil {
		t.Fatal("stale agent assignment should be rejected")
	}
	if err := validateKefuConversation("2", "WGG1", visitor); err == nil {
		t.Fatal("cross-enterprise visitor should be rejected")
	}
}
