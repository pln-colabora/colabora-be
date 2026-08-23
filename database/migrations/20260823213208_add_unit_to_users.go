package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260823213208_add_unit_to_users", UpAddUnitToUsers, DownAddUnitToUsers)
}

func UpAddUnitToUsers(db *gorm.DB) error {
	return db.AutoMigrate(&entities.User{})
}

func DownAddUnitToUsers(db *gorm.DB) error {
	return db.Migrator().DropColumn(&entities.User{}, "Unit")
}
