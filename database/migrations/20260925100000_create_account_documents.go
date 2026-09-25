package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260925100000_create_account_documents", UpCreateAccountDocuments, DownCreateAccountDocuments)
}

func UpCreateAccountDocuments(db *gorm.DB) error {
	return db.AutoMigrate(&entities.AccountDocument{})
}

func DownCreateAccountDocuments(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.AccountDocument{})
}
