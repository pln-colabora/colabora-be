package rbac

const (
	RoleAdmin              = "admin"
	RoleUser               = "user"
	RolePelayananPelanggan = "pelayanan-pelanggan"
	RoleTeknik             = "teknik"
	RolePerencanaan        = "perencanaan"
	RoleKonstruksi         = "konstruksi"
	RoleTransaksiEnergi    = "transaksi-energi"
	RoleJaringan           = "jaringan"
	RoleNps                = "nps"
	RolePdkb               = "pdkb"
	RoleVendorTiang        = "vendor-tiang"
	RoleVendorKonstruksi   = "vendor-konstruksi"
	RoleVendorSrApp        = "vendor-sr-app"
	RoleSuperUser          = "super-user"
)

var validRoles = map[string]bool{
	RoleAdmin: true, RoleUser: true, RolePelayananPelanggan: true, RoleTeknik: true,
	RolePerencanaan: true, RoleKonstruksi: true, RoleTransaksiEnergi: true,
	RoleJaringan: true, RoleNps: true, RolePdkb: true, RoleVendorTiang: true,
	RoleVendorKonstruksi: true, RoleVendorSrApp: true, RoleSuperUser: true,
}

func IsValidRole(role string) bool {
	return validRoles[role]
}

func CanManageAccounts(role string) bool {
	return role == RoleAdmin || role == RoleSuperUser
}

// ValidateRoleUnit ensures operational accounts have an organizational scope.
// Unit names themselves remain configuration data, so this only enforces the
// required/optional relationship to the selected role.
func ValidateRoleUnit(role, unit string) bool {
	if !IsValidRole(role) {
		return false
	}

	switch role {
	case RoleAdmin, RoleUser, RoleSuperUser:
		return true
	default:
		return unit != ""
	}
}
