package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/database/seeders/seeds"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261101100000_tariff_detail_and_priority", upTariffDetailAndPriority, downTariffDetailAndPriority)
}

var tariffDetailPermohonanColumns = []string{"golongan_tarif", "golongan_tarif_lama", "pembatas_baru", "pembatas_lama", "priority"}

// upTariffDetailAndPriority adds nullable tariff-detail and priority columns,
// then replaces the legacy power bands with the exact "Tangga Daya PLN" steps.
// Deployments only run migrations, so the catalogue sync must happen here;
// legacy rows are deactivated rather than deleted.
func upTariffDetailAndPriority(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&entities.Permohonan{}, &entities.TariffPowerOption{}); err != nil {
			return err
		}
		return seeds.SyncTariffPowerCatalogue(tx)
	})
}

func downTariffDetailAndPriority(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		for _, column := range tariffDetailPermohonanColumns {
			if tx.Migrator().HasColumn(&entities.Permohonan{}, column) {
				if err := tx.Migrator().DropColumn(&entities.Permohonan{}, column); err != nil {
					return err
				}
			}
		}
		for _, column := range []string{"daya_label", "pembatas"} {
			if tx.Migrator().HasColumn(&entities.TariffPowerOption{}, column) {
				if err := tx.Migrator().DropColumn(&entities.TariffPowerOption{}, column); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
