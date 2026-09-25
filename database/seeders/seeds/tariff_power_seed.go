package seeds

import (
	"errors"

	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"gorm.io/gorm"
)

type tariffBand struct {
	tarif, golongan, label string
	min, max               int64
	maxSet                 bool
}

// These bands mirror published PLN/ESDM tariff boundaries. The operational
// dropdown catalogue can replace or extend them without a schema change.
func ListTariffPowerSeeder(db *gorm.DB) error {
	bands := []tariffBand{
		{rbac.TarifSosial, "S-1", "450 VA", 450, 450, true},
		{rbac.TarifSosial, "S-1", "900 VA", 900, 900, true},
		{rbac.TarifSosial, "S-1", "1.300 VA", 1300, 1300, true},
		{rbac.TarifSosial, "S-1", "2.200 VA", 2200, 2200, true},
		{rbac.TarifSosial, "S-1", "3.500 VA s.d. 200 kVA", 3500, 200000, true},
		{rbac.TarifSosial, "S-2", "> 200 kVA", 200001, 0, false},
		{rbac.TarifRumahTangga, "R-1", "450 VA", 450, 450, true},
		{rbac.TarifRumahTangga, "R-1", "900 VA", 900, 900, true},
		{rbac.TarifRumahTangga, "R-1", "1.300 VA", 1300, 1300, true},
		{rbac.TarifRumahTangga, "R-1", "2.200 VA", 2200, 2200, true},
		{rbac.TarifRumahTangga, "R-2", "3.500 VA s.d. 5.500 VA", 3500, 5500, true},
		{rbac.TarifRumahTangga, "R-3", "6.600 VA atau lebih", 6600, 0, false},
		{rbac.TarifBisnis, "B-1", "450 VA", 450, 450, true},
		{rbac.TarifBisnis, "B-1", "900 VA", 900, 900, true},
		{rbac.TarifBisnis, "B-1", "1.300 VA", 1300, 1300, true},
		{rbac.TarifBisnis, "B-1", "2.200 VA s.d. 5.500 VA", 2200, 5500, true},
		{rbac.TarifBisnis, "B-2", "6.600 VA s.d. 200 kVA", 6600, 200000, true},
		{rbac.TarifBisnis, "B-3", "> 200 kVA", 200001, 0, false},
		{rbac.TarifIndustri, "I-1", "450 VA", 450, 450, true},
		{rbac.TarifIndustri, "I-1", "900 VA", 900, 900, true},
		{rbac.TarifIndustri, "I-1", "1.300 VA", 1300, 1300, true},
		{rbac.TarifIndustri, "I-1", "2.200 VA", 2200, 2200, true},
		{rbac.TarifIndustri, "I-1", "3.500 VA s.d. 14 kVA", 3500, 14000, true},
		{rbac.TarifIndustri, "I-2", "> 14 kVA s.d. 200 kVA", 14001, 200000, true},
		{rbac.TarifIndustri, "I-3", "> 200 kVA s.d. kurang dari 30.000 kVA", 200001, 29999999, true},
		{rbac.TarifIndustri, "I-4", "30.000 kVA atau lebih", 30000000, 0, false},
		{rbac.TarifPemerintah, "P-1", "450 VA", 450, 450, true},
		{rbac.TarifPemerintah, "P-1", "900 VA", 900, 900, true},
		{rbac.TarifPemerintah, "P-1", "1.300 VA", 1300, 1300, true},
		{rbac.TarifPemerintah, "P-1", "2.200 VA s.d. 5.500 VA", 2200, 5500, true},
		{rbac.TarifPemerintah, "P-1", "6.600 VA s.d. 200 kVA", 6600, 200000, true},
		{rbac.TarifPemerintah, "P-2", "> 200 kVA", 200001, 0, false},
	}

	for _, connection := range rbac.AllJenisSambungan {
		for _, band := range bands {
			var max *int64
			if band.maxSet {
				value := band.max
				max = &value
			}
			query := db.Where("tarif = ? AND golongan_tarif = ? AND jenis_sambungan = ? AND daya_min = ?", band.tarif, band.golongan, connection, band.min)
			if band.maxSet {
				query = query.Where("daya_max = ?", band.max)
			} else {
				query = query.Where("daya_max IS NULL")
			}
			var existing entities.TariffPowerOption
			err := query.First(&existing).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := db.Create(&entities.TariffPowerOption{
					Tarif: band.tarif, GolonganTarif: band.golongan, JenisSambungan: connection,
					DayaMin: band.min, DayaMax: max, Label: band.label, Active: true,
				}).Error; err != nil {
					return err
				}
			}
		}
	}
	return nil
}
