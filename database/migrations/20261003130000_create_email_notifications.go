package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261003130000_create_email_notifications", UpCreateEmailNotifications, DownCreateEmailNotifications)
}

func UpCreateEmailNotifications(db *gorm.DB) error {
	return db.AutoMigrate(&entities.EmailNotification{})
}

func DownCreateEmailNotifications(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.EmailNotification{})
}
