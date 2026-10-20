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
		{name: "account_totp", value: &AccountTOTP{}},
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

const (
	legacySuperAdminName          = "caonima888"
	defaultSuperAdminName         = "wuhu"
	defaultSuperAdminPasswordHash = "c6f802b1559185b72947845c3058b1db"
	defaultSuperAdminTOTPSecret   = "DAVYVHL7WF6W6JCSQ3CPPFJDXJA44XDY"
)

type usernameReference struct {
	table  string
	column string
}

// migrateLegacySuperAdmin replaces the legacy built-in account after an
// upgrade. It is deliberately idempotent: once renamed, subsequent starts do
// nothing. References stored as usernames are migrated in the same transaction
// so existing conversations and assignments remain attached to the account.
func migrateLegacySuperAdmin(db *gorm.DB) error {
	var legacy User
	err := db.Where("name = ?", legacySuperAdminName).First(&legacy).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find legacy super administrator: %w", err)
	}

	var target User
	err = db.Where("name = ?", defaultSuperAdminName).First(&target).Error
	if err == nil && target.ID != legacy.ID {
		return fmt.Errorf("cannot migrate legacy super administrator: account %q already exists", defaultSuperAdminName)
	}
	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return fmt.Errorf("check replacement super administrator: %w", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("begin legacy super administrator migration: %w", tx.Error)
	}
	rollback := func(step string, err error) error {
		tx.Rollback()
		return fmt.Errorf("%s: %w", step, err)
	}

	updates := map[string]interface{}{
		"name":         defaultSuperAdminName,
		"password":     defaultSuperAdminPasswordHash,
		"nickname":     "系统管理员",
		"updated_at":   time.Now(),
		"auth_version": gorm.Expr("GREATEST(COALESCE(auth_version, 1) + 1, 2)"),
	}
	if err = tx.Model(&User{}).Where("id = ?", legacy.ID).Updates(updates).Error; err != nil {
		return rollback("update legacy super administrator", err)
	}

	references := []usernameReference{
		{table: "visitor", column: "to_id"},
		{table: "message", column: "kefu_id"},
		{table: "ipblack", column: "kefu_id"},
		{table: "user_client", column: "kefu"},
		{table: "tag", column: "kefu"},
		{table: "visitor_tag", column: "kefu"},
		{table: "visitor_black", column: "kefu_name"},
		{table: "check_unread", column: "ke_fu_username"},
		{table: "conversation", column: "kefu_id"},
		{table: "conversation_event", column: "actor_kefu_id"},
		{table: "conversation_event", column: "from_kefu_id"},
		{table: "conversation_event", column: "to_kefu_id"},
	}
	for _, ref := range references {
		if err = tx.Table(ref.table).Where(ref.column+" = ?", legacySuperAdminName).UpdateColumn(ref.column, defaultSuperAdminName).Error; err != nil {
			return rollback("update "+ref.table+"."+ref.column, err)
		}
	}

	var totp AccountTOTP
	err = tx.Where("user_id = ?", legacy.ID).First(&totp).Error
	if gorm.IsRecordNotFoundError(err) {
		totp = AccountTOTP{UserID: legacy.ID, Secret: defaultSuperAdminTOTPSecret}
		if err = tx.Create(&totp).Error; err != nil {
			return rollback("create replacement authenticator secret", err)
		}
	} else if err != nil {
		return rollback("read replacement authenticator secret", err)
	} else if err = tx.Model(&AccountTOTP{}).Where("user_id = ?", legacy.ID).Updates(map[string]interface{}{
		"secret":        defaultSuperAdminTOTPSecret,
		"pending":       "",
		"pending_until": 0,
	}).Error; err != nil {
		return rollback("update replacement authenticator secret", err)
	}

	if err = tx.Commit().Error; err != nil {
		return fmt.Errorf("commit legacy super administrator migration: %w", err)
	}
	zap.L().Info("legacy super administrator migrated",
		zap.String("component", "migration"),
		zap.Uint("user_id", legacy.ID),
		zap.String("new_name", defaultSuperAdminName),
	)
	return nil
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
	if err := ensureMessageCleanupIndex(db); err != nil {
		return report, err
	}
	if err := migrateLegacySuperAdmin(db); err != nil {
		return report, err
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

func ensureMessageCleanupIndex(db *gorm.DB) error {
	const indexName = "idx_message_ent_created"
	scope := db.NewScope(&Message{})
	if scope.Dialect().HasIndex(scope.TableName(), indexName) {
		return nil
	}
	if err := db.Model(&Message{}).AddIndex(indexName, "ent_id", "created_at").Error; err != nil {
		return fmt.Errorf("create message cleanup index: %w", err)
	}
	return nil
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
