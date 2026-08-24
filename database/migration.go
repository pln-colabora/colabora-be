package database

import (
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&entities.Migration{},
		&entities.User{},
		&entities.RefreshToken{},
		&entities.SLARule{},
		&entities.Permohonan{},
		&entities.PermohonanActivity{},
		&entities.ActivityLog{},
		&entities.Document{},
	); err != nil {
		return err
	}

	manager := NewMigrationManager(db)
	return manager.Run()
}
