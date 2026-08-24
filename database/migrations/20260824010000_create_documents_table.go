package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260824010000_create_documents_table", UpCreateDocumentsTable, DownCreateDocumentsTable)
}

func UpCreateDocumentsTable(db *gorm.DB) error {
	return db.AutoMigrate(&entities.Document{})
}

func DownCreateDocumentsTable(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.Document{})
}
