package rbac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOwnsActivity(t *testing.T) {
	tests := []struct {
		name            string
		role            string
		unit            string
		activityNumber  int16
		jenisSambungan  string
		ulpUnit         string
		ownerFnOverride string
		want            bool
	}{
		{
			name:           "unknown activity number never has an owner",
			role:           RoleTeknik,
			activityNumber: 99,
			want:           false,
		},
		{
			name:           "stage 3 activity owned by teknik for JTR",
			role:           RoleTeknik,
			unit:           "ULP Taman",
			activityNumber: 3,
			jenisSambungan: JenisSambunganJTR,
			ulpUnit:        "ULP Taman",
			want:           true,
		},
		{
			name:           "stage 3 activity owned by perencanaan for PLG TM",
			role:           RolePerencanaan,
			activityNumber: 4,
			jenisSambungan: JenisSambunganPlgTmLebih5,
			want:           true,
		},
		{
			name:           "stage 3 activity rejects teknik for PLG TM",
			role:           RoleTeknik,
			unit:           "ULP Taman",
			activityNumber: 3,
			jenisSambungan: JenisSambunganPlgTmLebih5,
			ulpUnit:        "ULP Taman",
			want:           false,
		},
		{
			name:           "stage 6 activity owned by vendor-sr-app for JTR",
			role:           RoleVendorSrApp,
			activityNumber: 14,
			jenisSambungan: JenisSambunganJTR,
			want:           true,
		},
		{
			name:           "stage 6 activity rejects vendor-sr-app for PLG TM",
			role:           RoleVendorSrApp,
			activityNumber: 14,
			jenisSambungan: JenisSambunganPlgTmKurang5,
			want:           false,
		},
		{
			name:           "ULP-scoped role rejected on unit mismatch",
			role:           RoleTeknik,
			unit:           "ULP Taman",
			activityNumber: 2,
			ulpUnit:        "ULP Karang Pilang",
			want:           false,
		},
		{
			name:            "owner override narrows a co-owned stage to one role",
			role:            RoleNps,
			activityNumber:  3,
			jenisSambungan:  JenisSambunganJTR,
			ownerFnOverride: RoleNps,
			want:            true,
		},
		{
			name:            "owner override excludes roles not listed",
			role:            RolePerencanaan,
			activityNumber:  3,
			jenisSambungan:  JenisSambunganJTR,
			ownerFnOverride: RoleNps,
			want:            false,
		},
		{
			name:           "closing activity owned by pelayanan-pelanggan",
			role:           RolePelayananPelanggan,
			unit:           "ULP Taman",
			activityNumber: 17,
			ulpUnit:        "ULP Taman",
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OwnsActivity(tt.role, tt.unit, tt.activityNumber, tt.jenisSambungan, tt.ulpUnit, tt.ownerFnOverride)
			assert.Equal(t, tt.want, got)
		})
	}
}
