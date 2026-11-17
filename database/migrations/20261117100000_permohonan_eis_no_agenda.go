package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

// The EIS application number (NOAGENDA) identifies an imported request, so
// re-importing the same EIS row can be recognised even when one customer has
// several otherwise identical applications.
func init() {
	database.RegisterMigration("20261117100000_permohonan_eis_no_agenda", func(db *gorm.DB) error {
		if !db.Migrator().HasColumn(&entities.Permohonan{}, "EISNoAgenda") {
			if err := db.Migrator().AddColumn(&entities.Permohonan{}, "EISNoAgenda"); err != nil {
				return err
			}
		}
		if !db.Migrator().HasIndex(&entities.Permohonan{}, "EISNoAgenda") {
			return db.Migrator().CreateIndex(&entities.Permohonan{}, "EISNoAgenda")
		}
		return nil
	}, func(db *gorm.DB) error {
		if db.Migrator().HasIndex(&entities.Permohonan{}, "EISNoAgenda") {
			if err := db.Migrator().DropIndex(&entities.Permohonan{}, "EISNoAgenda"); err != nil {
				return err
			}
		}
		return db.Migrator().DropColumn(&entities.Permohonan{}, "EISNoAgenda")
	})
}
