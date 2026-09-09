package database

import (
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	manager := NewMigrationManager(db)
	if err := manager.Run(); err != nil {
		return err
	}

	if err := db.AutoMigrate(
		&entities.Migration{},
		&entities.User{},
		&entities.RefreshToken{},
		&entities.SLARule{},
		&entities.Permohonan{},
		&entities.PermohonanActivity{},
		&entities.ActivityLog{},
		&entities.Document{},
		&entities.DocumentEvidence{},
		&entities.VendorAssignment{},
	); err != nil {
		return err
	}
	return nil
}
