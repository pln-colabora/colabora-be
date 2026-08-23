package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260823221042_create_permohonan_activity_table", UpCreatePermohonanActivityTable, DownCreatePermohonanActivityTable)
}

func UpCreatePermohonanActivityTable(db *gorm.DB) error {
	return db.AutoMigrate(&entities.PermohonanActivity{})
}

func DownCreatePermohonanActivityTable(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.PermohonanActivity{})
}
