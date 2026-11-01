package entities

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TariffPowerOption is the configurable tariff/power catalogue used by the
// Activity #1 form. Each row is one exact power step (daya_min = daya_max) from
// the official "Tangga Daya PLN" table; a nil maximum is reserved for the
// unbounded I4 row.
type TariffPowerOption struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Tarif          string    `gorm:"type:varchar(30);not null;uniqueIndex:idx_tariff_power_option" json:"tarif"`
	GolonganTarif  string    `gorm:"type:varchar(30);not null;uniqueIndex:idx_tariff_power_option" json:"golongan_tarif"`
	JenisSambungan string    `gorm:"type:varchar(30);not null;uniqueIndex:idx_tariff_power_option" json:"jenis_sambungan"`
	DayaMin        int64     `gorm:"type:bigint;not null;uniqueIndex:idx_tariff_power_option" json:"daya_min"`
	DayaMax        *int64    `gorm:"type:bigint;uniqueIndex:idx_tariff_power_option" json:"daya_max"`
	Label          string    `gorm:"type:varchar(80);not null" json:"label"`
	DayaLabel      string    `gorm:"type:varchar(80)" json:"daya_label"`
	Pembatas       *string   `gorm:"type:varchar(30)" json:"pembatas"`
	Active         bool      `gorm:"not null;default:true" json:"active"`
	Timestamp
}

func (TariffPowerOption) TableName() string { return "tariff_power_options" }

func (o *TariffPowerOption) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}
