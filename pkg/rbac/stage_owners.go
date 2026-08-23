package rbac

import (
	"strings"

	"gorm.io/gorm"
)

// ActivityStageMap maps a 1-17 swimlane activity number to its 1-7 UI stage.
var ActivityStageMap = map[int16]int16{
	1:  1,
	2:  2,
	3:  3,
	4:  3,
	5:  3,
	6:  4,
	7:  4,
	8:  4,
	9:  4,
	10: 4,
	11: 5,
	12: 5,
	13: 6,
	14: 6,
	15: 7,
	16: 7,
	17: 7,
}

// StageOwners maps each of the 7 UI stages to the fn(s) that own it.
var StageOwners = map[int16][]string{
	1: {RolePelayananPelanggan},
	2: {RoleTeknik},
	3: {RolePerencanaan, RoleTeknik, RoleNps},
	4: {RolePerencanaan, RoleKonstruksi, RoleTransaksiEnergi},
	5: {RoleVendorTiang, RoleVendorKonstruksi, RolePdkb},
	6: {RoleTeknik, RoleJaringan, RoleVendorSrApp, RoleVendorKonstruksi},
	7: {RolePelayananPelanggan},
}

// stage3JenisRestriction / stage6JenisRestriction: fn -> allowed JenisSambungan values,
// only for roles whose ownership at that stage is split by connection type.
// A role absent from the map for that stage has no restriction there.
var stage3JenisRestriction = map[string][]string{
	RoleTeknik:      {JenisSambunganJTR, JenisSambunganJTMGardu},
	RolePerencanaan: {JenisSambunganPlgTmKurang5, JenisSambunganPlgTmLebih5},
}

var stage6JenisRestriction = map[string][]string{
	RoleTeknik:           {JenisSambunganJTR, JenisSambunganJTMGardu},
	RoleJaringan:         {JenisSambunganPlgTmKurang5, JenisSambunganPlgTmLebih5},
	RoleVendorSrApp:      {JenisSambunganJTR, JenisSambunganJTMGardu},
	RoleVendorKonstruksi: {JenisSambunganPlgTmKurang5, JenisSambunganPlgTmLebih5},
}

var ulpScopedRoles = map[string]bool{
	RolePelayananPelanggan: true,
	RoleTeknik:             true,
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

// OwnsStage is the server-side equivalent of the mockup's colaboraOwnsStage(): does this
// role/unit own the permohonan's current stage right now?
func OwnsStage(role, unit string, currentStage int16, jenisSambungan, ulpUnit, ownerFnOverride string) bool {
	owners := StageOwners[currentStage]
	if trimmed := strings.TrimSpace(ownerFnOverride); trimmed != "" {
		var override []string
		for _, fn := range strings.Split(trimmed, ",") {
			override = append(override, strings.TrimSpace(fn))
		}
		owners = override
	}

	if !contains(owners, role) {
		return false
	}

	var restriction map[string][]string
	switch currentStage {
	case 3:
		restriction = stage3JenisRestriction
	case 6:
		restriction = stage6JenisRestriction
	}
	if restriction != nil {
		if allowed, restricted := restriction[role]; restricted && !contains(allowed, jenisSambungan) {
			return false
		}
	}

	if ulpScopedRoles[role] && unit != ulpUnit {
		return false
	}

	return true
}

// ApplyScopeMine is the SQL-composed equivalent of OwnsStage, for filtering a list of
// permohonan down to the rows the given role/unit owns right now (`scope=mine`).
func ApplyScopeMine(query *gorm.DB, role, unit string) *gorm.DB {
	if role == RoleSuperUser {
		// super-user is read-only cross-unit monitoring, never a stage owner.
		return query.Where("1 = 0")
	}

	var ownedStages []int16
	for stage, owners := range StageOwners {
		if contains(owners, role) {
			ownedStages = append(ownedStages, stage)
		}
	}
	if len(ownedStages) == 0 {
		return query.Where("1 = 0")
	}

	query = query.Where("current_stage IN (?)", ownedStages)

	if allowed, restricted := stage3JenisRestriction[role]; restricted {
		query = query.Where("current_stage != 3 OR jenis_sambungan IN (?)", allowed)
	}
	if allowed, restricted := stage6JenisRestriction[role]; restricted {
		query = query.Where("current_stage != 6 OR jenis_sambungan IN (?)", allowed)
	}

	if ulpScopedRoles[role] {
		query = query.Where("ulp_unit = ?", unit)
	}

	query = query.Where(
		"owner_fn_override = '' OR owner_fn_override IS NULL OR (',' || owner_fn_override || ',') LIKE ?",
		"%,"+role+",%",
	)

	return query
}
