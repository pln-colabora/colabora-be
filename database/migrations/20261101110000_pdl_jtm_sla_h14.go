package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261101110000_pdl_jtm_sla_h14", upPDLJTMSLAH14, downPDLJTMSLAH14)
}

func upPDLJTMSLAH14(db *gorm.DB) error {
	return db.Exec(`UPDATE sla_rules SET offset_days = 14, updated_at = NOW()
		WHERE activity_number = 15 AND jenis_sambungan = 'JTM/Gardu'`).Error
}

func downPDLJTMSLAH14(db *gorm.DB) error {
	return db.Exec(`UPDATE sla_rules SET offset_days = 10, updated_at = NOW()
		WHERE activity_number = 15 AND jenis_sambungan = 'JTM/Gardu'`).Error
}
