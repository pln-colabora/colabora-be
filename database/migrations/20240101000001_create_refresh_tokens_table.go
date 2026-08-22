package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20240101000001_create_refresh_tokens_table", Up20240101000001CreateRefreshTokensTable, Down20240101000001CreateRefreshTokensTable)
}

func Up20240101000001CreateRefreshTokensTable(db *gorm.DB) error {
	return db.AutoMigrate(&entities.RefreshToken{})
}

func Down20240101000001CreateRefreshTokensTable(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.RefreshToken{})
}
