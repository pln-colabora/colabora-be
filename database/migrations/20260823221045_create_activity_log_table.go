package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260823221045_create_activity_log_table", UpCreateActivityLogTable, DownCreateActivityLogTable)
}

func UpCreateActivityLogTable(db *gorm.DB) error {
	return db.AutoMigrate(&entities.ActivityLog{})
}

func DownCreateActivityLogTable(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.ActivityLog{})
}
