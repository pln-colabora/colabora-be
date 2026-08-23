package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260823221037_create_permohonan_table", UpCreatePermohonanTable, DownCreatePermohonanTable)
}

func UpCreatePermohonanTable(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Permohonan{})
}

func DownCreatePermohonanTable(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.Permohonan{})
}
