package rbac

import "gorm.io/gorm"

func IsVendor(role string) bool {
	return role == RoleVendorTiang || role == RoleVendorKonstruksi || role == RoleVendorSrApp
}

// ApplyReadScope is the common access predicate for lists and individual resources.
// It must be applied before pagination/counting, never after fetching a page.
func ApplyReadScope(db *gorm.DB, role, unit, userID string) *gorm.DB {
	switch role {
	case RoleSuperUser, RoleNps, RolePerencanaan, RoleKonstruksi, RoleTransaksiEnergi, RoleJaringan, RolePdkb:
		return db
	case RoleTeknik, RolePelayananPelanggan:
		if unit != "" {
			return db.Where("permohonan.ulp_unit = ?", unit)
		}
	case RoleVendorTiang, RoleVendorKonstruksi, RoleVendorSrApp:
		if userID != "" {
			return db.Where("EXISTS (SELECT 1 FROM vendor_assignments va WHERE va.permohonan_id = permohonan.id AND va.vendor_id = ? AND va.vendor_role = ?)", userID, role)
		}
	}
	return db.Where("1 = 0")
}
