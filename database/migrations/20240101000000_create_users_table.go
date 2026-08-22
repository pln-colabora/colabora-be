package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20240101000000_create_users_table", Up20240101000000CreateUsersTable, Down20240101000000CreateUsersTable)
}

func Up20240101000000CreateUsersTable(db *gorm.DB) error {
	return db.AutoMigrate(&entities.User{})
}

func Down20240101000000CreateUsersTable(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.User{})
}
