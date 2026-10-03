package service

import (
	"testing"

	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestVendorRoleForWOAPPFollowsSRAPPOwner(t *testing.T) {
	tests := []struct {
		name       string
		connection string
		want       string
	}{
		{name: "JTR", connection: rbac.JenisSambunganJTR, want: rbac.RoleVendorSrApp},
		{name: "JTM", connection: rbac.JenisSambunganJTMGardu, want: rbac.RoleVendorSrApp},
		{name: "PLG TM below 5 GWNG", connection: rbac.JenisSambunganPlgTmKurang5, want: rbac.RoleVendorKonstruksi},
		{name: "PLG TM above 5 GWNG", connection: rbac.JenisSambunganPlgTmLebih5, want: rbac.RoleVendorKonstruksi},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			role, ok := vendorRoleForWO(workflow.WOAPP, test.connection)
			require.True(t, ok)
			require.Equal(t, test.want, role)
		})
	}

	_, ok := vendorRoleForWO(workflow.WOAPP, "invalid-connection")
	require.False(t, ok)
	_, ok = vendorRoleForWO(workflow.WOKonstruksi, rbac.JenisSambunganJTR)
	require.False(t, ok)
}

func TestGenerateVendorAPPWOPDF(t *testing.T) {
	content, err := generateVendorWOPDF(vendorWOPDFData{
		Title: "SURAT PERINTAH KERJA VENDOR APP",
		Scope: "Pekerjaan persiapan dan pemasangan APP sesuai dokumen teknis",
	})
	require.NoError(t, err)
	require.True(t, len(content) > 5)
	require.Equal(t, "%PDF", string(content[:4]))
}
