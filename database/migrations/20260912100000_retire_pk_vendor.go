package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260912100000_retire_pk_vendor", UpRetirePKVendor, DownRetirePKVendor)
}

// UpRetirePKVendor keeps completed PK Vendor history while making unfinished
// legacy records non-actionable after the supporting node is retired.
func UpRetirePKVendor(db *gorm.DB) error {
	return db.Exec(`UPDATE permohonan_activities
		SET status = 'skipped'
		WHERE workflow_node = 'pk_vendor'
			AND status IN ('locked', 'available', 'in_progress')`).Error
}

func DownRetirePKVendor(db *gorm.DB) error {
	return db.Exec(`UPDATE permohonan_activities
		SET status = 'locked'
		WHERE workflow_node = 'pk_vendor'
			AND status = 'skipped'`).Error
}
