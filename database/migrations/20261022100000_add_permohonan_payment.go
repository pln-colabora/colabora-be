package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261022100000_add_permohonan_payment", upPermohonanPayment, downPermohonanPayment)
}

func upPermohonanPayment(db *gorm.DB) error {
	return db.Exec("ALTER TABLE permohonan ADD COLUMN IF NOT EXISTS tanggal_pembayaran date, ADD COLUMN IF NOT EXISTS nominal_pembayaran bigint").Error
}

func downPermohonanPayment(db *gorm.DB) error {
	return db.Exec("ALTER TABLE permohonan DROP COLUMN IF EXISTS tanggal_pembayaran, DROP COLUMN IF EXISTS nominal_pembayaran").Error
}
