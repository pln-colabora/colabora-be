package rbac

import (
	"strings"

	"github.com/pln-colabora/colabora-be/pkg/workflow"
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

// ApplyScopeMine selects aggregates with at least one actionable persisted node owned
// by the caller. CurrentStage remains presentation metadata and is not consulted.
func ApplyScopeMine(query *gorm.DB, role, unit string) *gorm.DB {
	if role == RoleSuperUser {
		return query.Where("1 = 0")
	}

	var jtrJtmNodes []string
	var plgTMNodes []string
	for _, definition := range workflow.Definitions() {
		if definition.OwnerJTRJTM == role {
			jtrJtmNodes = append(jtrJtmNodes, string(definition.Code))
		}
		if definition.OwnerPLGTM == role {
			plgTMNodes = append(plgTMNodes, string(definition.Code))
		}
	}
	if len(jtrJtmNodes) == 0 && len(plgTMNodes) == 0 {
		return query.Where("1 = 0")
	}

	conditions := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if len(jtrJtmNodes) > 0 {
		conditions = append(conditions, "(permohonan.jenis_sambungan IN ? AND mine_node.workflow_node IN ?)")
		args = append(args, []string{JenisSambunganJTR, JenisSambunganJTMGardu}, jtrJtmNodes)
	}
	if len(plgTMNodes) > 0 {
		conditions = append(conditions, "(permohonan.jenis_sambungan IN ? AND mine_node.workflow_node IN ?)")
		args = append(args, []string{JenisSambunganPlgTmKurang5, JenisSambunganPlgTmLebih5}, plgTMNodes)
	}
	ownership := strings.Join(conditions, " OR ")
	query = query.Where(
		"EXISTS (SELECT 1 FROM permohonan_activities mine_node WHERE mine_node.permohonan_id = permohonan.id AND mine_node.status IN ('available', 'in_progress') AND ("+ownership+"))",
		args...,
	)

	if ulpScopedRoles[role] {
		query = query.Where("permohonan.ulp_unit = ?", unit)
	}
	return query
}
