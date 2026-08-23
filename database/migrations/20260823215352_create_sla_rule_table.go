package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260823215352_create_sla_rule_table", UpCreateSlaRuleTable, DownCreateSlaRuleTable)
}

func UpCreateSlaRuleTable(db *gorm.DB) error {
	return db.AutoMigrate(&entities.SLARule{})
}

func DownCreateSlaRuleTable(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.SLARule{})
}
