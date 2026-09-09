package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260908130000_create_vendor_assignments", UpVendorAssignments, DownVendorAssignments)
}
func UpVendorAssignments(db *gorm.DB) error { return db.AutoMigrate(&entities.VendorAssignment{}) }
func DownVendorAssignments(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.VendorAssignment{})
}
