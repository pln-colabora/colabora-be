package rbac

import (
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestWorkflowOwnershipMatrix(t *testing.T) {
	roles := []string{RolePelayananPelanggan, RoleTeknik, RolePerencanaan, RoleNps, RoleKonstruksi, RoleTransaksiEnergi, RoleJaringan, RolePdkb, RoleVendorTiang, RoleVendorKonstruksi, RoleVendorSrApp, RoleSuperUser, "unknown"}
	owners := map[workflow.Code][2]string{
		workflow.Permohonan: {RolePelayananPelanggan, RoleNps}, workflow.Survei: {RoleTeknik, RolePerencanaan},
		workflow.RAB: {RoleTeknik, RolePerencanaan}, workflow.KebutuhanTiang: {RoleTeknik, RolePerencanaan},
		workflow.Perluasan: {RoleNps, RoleNps}, workflow.NPS: {RoleNps, RoleNps},
		workflow.WOTiang: {RolePerencanaan, RolePerencanaan}, workflow.WOKonstruksi: {RoleKonstruksi, RoleKonstruksi},
		workflow.WOAPP: {RoleTransaksiEnergi, RoleTransaksiEnergi}, workflow.Reservasi: {RoleTransaksiEnergi, RoleTransaksiEnergi}, workflow.Tera: {RoleTransaksiEnergi, RoleTransaksiEnergi},
		workflow.PKVendor: {RoleKonstruksi, RoleKonstruksi}, workflow.WOPDKB: {RoleKonstruksi, RoleKonstruksi},
		workflow.PemasanganTiang: {RoleVendorTiang, RoleVendorTiang}, workflow.Konstruksi: {RoleVendorKonstruksi, RoleVendorKonstruksi},
		workflow.DokumentasiPDKB: {RolePdkb, RolePdkb}, workflow.Energize: {RoleTeknik, RoleJaringan}, workflow.SRAPP: {RoleVendorSrApp, RoleVendorKonstruksi},
		workflow.PDL: {RolePelayananPelanggan, RolePelayananPelanggan}, workflow.AIL: {RolePelayananPelanggan, RolePelayananPelanggan}, workflow.Selesai: {RolePelayananPelanggan, RolePelayananPelanggan},
	}
	require.Len(t, owners, len(workflow.Definitions()))
	for _, connection := range AllJenisSambungan {
		family := 0
		if connection == JenisSambunganPlgTmKurang5 || connection == JenisSambunganPlgTmLebih5 {
			family = 1
		}
		for code, pair := range owners {
			for _, role := range roles {
				require.Equal(t, role == pair[family], OwnsWorkflowNode(role, "ULP Taman", connection, "ULP Taman", code), "%s/%s/%s", connection, code, role)
				if role == RoleTeknik || role == RolePelayananPelanggan {
					require.False(t, OwnsWorkflowNode(role, "other", connection, "ULP Taman", code))
					require.False(t, OwnsWorkflowNode(role, "", connection, "", code))
				}
			}
		}
	}
	require.False(t, OwnsWorkflowNode(RoleNps, "UP3", "bad", "ULP Taman", workflow.NPS))
	require.False(t, OwnsWorkflowNode(RoleNps, "UP3", "JTR", "ULP Taman", "unknown"))
}

func TestWorkflowAuthorizationAndActions(t *testing.T) {
	no := false
	s := workflow.Snapshot{Connection: "JTR", Decisions: workflow.Decisions{KebutuhanTiang: &no, PerluPDKB: &no, NPS: workflow.Delegated}}
	require.NoError(t, AuthorizeWorkflowNode(RolePelayananPelanggan, "ULP Taman", "ULP Taman", s, workflow.Permohonan))
	require.ErrorIs(t, AuthorizeWorkflowNode(RoleNps, "UP3", "ULP Taman", s, workflow.Permohonan), ErrWorkflowForbidden)
	require.ErrorIs(t, AuthorizeWorkflowNode(RoleTeknik, "ULP Taman", "ULP Taman", s, workflow.Survei), workflow.ErrNotActionable)
	for _, code := range []workflow.Code{workflow.Permohonan, workflow.Survei, workflow.RAB, workflow.KebutuhanTiang, workflow.Perluasan, workflow.NPS, workflow.WOKonstruksi} {
		next, _, err := workflow.Transition(s, code, workflow.Completed)
		require.NoError(t, err)
		s = next
	}
	require.ErrorIs(t, AuthorizeWorkflowNode(RolePerencanaan, "UP3", "ULP Taman", s, workflow.WOTiang), workflow.ErrNotActionable)
	require.ErrorIs(t, AuthorizeWorkflowNode(RoleKonstruksi, "UP3", "ULP Taman", s, workflow.WOAPP), ErrWorkflowForbidden)
	actions, err := AvailableActions(RoleKonstruksi, "UP3", "ULP Taman", s)
	require.NoError(t, err)
	require.Len(t, actions, 1)
	require.Equal(t, workflow.PKVendor, actions[0].Code)
	actions, err = AvailableActions(RoleSuperUser, "UP3", "ULP Taman", s)
	require.NoError(t, err)
	require.Empty(t, actions)
	require.ErrorIs(t, AuthorizeWorkflowNode(RoleSuperUser, "UP3", "ULP Taman", s, workflow.PKVendor), ErrWorkflowForbidden)
}
