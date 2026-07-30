package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"go.uber.org/zap"

	"go-fly-muti/common"
)

// SchemaMigration stores the latest successful automatic schema check. The
// migration itself remains idempotent and runs at every service start so newly
// added model fields are picked up without an SQL import.
type SchemaMigration struct {
	ID         uint      `gorm:"primary_key" json:"id"`
	Version    string    `gorm:"type:varchar(50);not null" json:"version"`
	AppliedAt  time.Time `gorm:"not null" json:"applied_at"`
	DurationMs int64     `gorm:"not null;default:0" json:"duration_ms"`
	ModelCount int       `gorm:"not null;default:0" json:"model_count"`
}

func (SchemaMigration) TableName() string {
	return "schema_migration"
}

// These private compatibility models cover tables whose runtime model lives
// in models/v2. Keeping the migration definitions here avoids an import cycle.
type migrationTag struct {
	ID        uint      `gorm:"primary_key"`
	CreatedAt time.Time `gorm:"index"`
	Name      string    `gorm:"type:varchar(100);index"`
	Kefu      string    `gorm:"type:varchar(100);index"`
	EntId     uint      `gorm:"index"`
}

func (migrationTag) TableName() string {
	return "tag"
}

type migrationVisitorTag struct {
	ID        uint      `gorm:"primary_key"`
	CreatedAt time.Time `gorm:"index"`
	VisitorId string    `gorm:"type:varchar(100);index"`
	TagId     uint      `gorm:"index"`
	Kefu      string    `gorm:"type:varchar(100);index"`
	EntId     uint      `gorm:"index"`
}

func (migrationVisitorTag) TableName() string {
	return "visitor_tag"
}

type migrationModel struct {
	name  string
	value interface{}
}

func schemaModels() []migrationModel {
	return []migrationModel{
		{name: "schema_migration", value: &SchemaMigration{}},
		{name: "user", value: &User{}},
		{name: "visitor", value: &Visitor{}},
		{name: "visitor_ext", value: &VisitorExt{}},
		{name: "visitor_attr", value: &Visitor_attr{}},
		{name: "message", value: &Message{}},
		{name: "user_role", value: &User_role{}},
		{name: "role", value: &Role{}},
		{name: "welcome", value: &Welcome{}},
		{name: "ipblack", value: &Ipblack{}},
		{name: "config", value: &Config{}},
		{name: "ent_config", value: &EntConfig{}},
		{name: "about", value: &About{}},
		{name: "reply_group", value: &ReplyGroup{}},
		{name: "reply_item", value: &ReplyItem{}},
		{name: "article_cate", value: &ArticleCate{}},
		{name: "article", value: &Article{}},
		{name: "user_client", value: &User_client{}},
		{name: "tag", value: &migrationTag{}},
		{name: "visitor_tag", value: &migrationVisitorTag{}},
		{name: "ip_auth", value: &IpAuth{}},
		{name: "oauth", value: &Oauth{}},
		{name: "new", value: &New{}},
		{name: "visitor_black", value: &VisitorBlack{}},
		{name: "check_unread", value: &Check_unread{}},
		{name: "conversation", value: &Conversation{}},
		{name: "conversation_event", value: &ConversationEvent{}},
	}
}

type MigrationReport struct {
	Version    string
	ModelCount int
	Duration   time.Duration
}

// RunSchemaMigrations creates missing tables, columns and model-declared
// indexes. GORM v1 AutoMigrate is intentionally non-destructive: it does not
// delete columns or existing data.
func RunSchemaMigrations(db *gorm.DB) (MigrationReport, error) {
	report := MigrationReport{Version: common.Version}
	if db == nil {
		return report, fmt.Errorf("database connection is nil")
	}

	startedAt := time.Now()
	models := schemaModels()
	for _, model := range models {
		if err := db.AutoMigrate(model.value).Error; err != nil {
			return report, fmt.Errorf("auto migrate table %s: %w", model.name, err)
		}
	}

	report.ModelCount = len(models)
	report.Duration = time.Since(startedAt)
	state := SchemaMigration{
		ID:         1,
		Version:    report.Version,
		AppliedAt:  time.Now(),
		DurationMs: report.Duration.Milliseconds(),
		ModelCount: report.ModelCount,
	}
	if err := db.Save(&state).Error; err != nil {
		return report, fmt.Errorf("save schema migration state: %w", err)
	}

	zap.L().Info("database schema is up to date",
		zap.String("component", "migration"),
		zap.String("version", report.Version),
		zap.Int("model_count", report.ModelCount),
		zap.Duration("duration", report.Duration),
	)
	return report, nil
}

func migrationModelNames() []string {
	models := schemaModels()
	names := make([]string, 0, len(models))
	for _, model := range models {
		names = append(names, model.name)
	}
	return names
}

func validateMigrationModels() error {
	seen := make(map[string]struct{})
	for _, name := range migrationModelNames() {
		name = strings.TrimSpace(name)
		if name == "" {
			return fmt.Errorf("migration model has an empty table name")
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate migration table %s", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}
