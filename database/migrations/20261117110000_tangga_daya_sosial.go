package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/seeders/seeds"
	"gorm.io/gorm"
)

// The corrected "Tangga Daya PLN" moves sosial S1 to 220–197.000 VA and S2 to
// 210.000–13.800.000 VA and drops S3. Deployments only run migrations, so the
// catalogue is re-synced here; retired rows are deactivated, never deleted.
func init() {
	database.RegisterMigration("20261117110000_tangga_daya_sosial", func(db *gorm.DB) error {
		return seeds.SyncTariffPowerCatalogue(db)
	}, func(db *gorm.DB) error {
		return nil
	})
}
