package models

import "testing"

func TestMigrationModelsAreUnique(t *testing.T) {
	if err := validateMigrationModels(); err != nil {
		t.Fatal(err)
	}
}

func TestMigrationModelsIncludeWorkflowTables(t *testing.T) {
	required := map[string]bool{
		"schema_migration":   false,
		"conversation":       false,
		"conversation_event": false,
		"check_unread":       false,
		"visitor":            false,
		"message":            false,
		"user":               false,
	}
	for _, name := range migrationModelNames() {
		if _, exists := required[name]; exists {
			required[name] = true
		}
	}
	for name, exists := range required {
		if !exists {
			t.Errorf("required migration table %s is missing", name)
		}
	}
}
