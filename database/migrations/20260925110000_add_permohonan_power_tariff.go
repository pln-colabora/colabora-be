package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260925110000_add_permohonan_power_tariff", UpPermohonanPowerTariff, DownPermohonanPowerTariff)
}

func UpPermohonanPowerTariff(db *gorm.DB) error {
	if err := db.AutoMigrate(&entities.Permohonan{}, &entities.TariffPowerOption{}); err != nil {
		return err
	}
	return nil
}

func DownPermohonanPowerTariff(db *gorm.DB) error {
	if err := db.Migrator().DropTable(&entities.TariffPowerOption{}); err != nil {
		return err
	}
	for _, column := range []string{"tarif", "daya_lama", "daya_baru"} {
		if db.Migrator().HasColumn(&entities.Permohonan{}, column) {
			if err := db.Migrator().DropColumn(&entities.Permohonan{}, column); err != nil {
				return err
			}
		}
	}
	return nil
}
