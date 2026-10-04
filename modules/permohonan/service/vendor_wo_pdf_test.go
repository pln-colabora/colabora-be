package service

import (
	"testing"
	"time"

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

func TestGenerateVendorWOPDFSupportsBlankTemplateValues(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		scope      string
		issuerRole string
		subject    string
	}{
		{
			name:       "tiang",
			title:      "SURAT PERINTAH KERJA VENDOR TIANG",
			scope:      "Pekerjaan pemasangan tiang sesuai dokumen teknis",
			issuerRole: "PERENCANAAN",
			subject:    "Perintah Kerja / Work Order Pemasangan Tiang",
		},
		{
			name:       "konstruksi",
			title:      "SURAT PERINTAH KERJA VENDOR KONSTRUKSI",
			scope:      "Pekerjaan konstruksi jaringan sesuai dokumen teknis",
			issuerRole: "KONSTRUKSI",
			subject:    "Perintah Kerja / Work Order Konstruksi Jaringan",
		},
		{
			name:       "app",
			title:      "SURAT PERINTAH KERJA VENDOR APP",
			scope:      "Pekerjaan APP sesuai dokumen teknis",
			issuerRole: "TRANSAKSI ENERGI",
			subject:    "Perintah Kerja / Work Order APP",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content, err := generateVendorWOPDF(vendorWOPDFData{
				Title: test.title, Scope: test.scope, IssuerRole: test.issuerRole,
			})
			require.NoError(t, err)
			require.Greater(t, len(content), 5)
			require.Equal(t, "%PDF", string(content[:4]))
			require.Equal(t, test.subject, woSubject(test.title))
		})
	}

	zero := time.Time{}
	require.Empty(t, formatWODate(&zero))
	issuedAt := time.Date(2026, time.October, 4, 0, 0, 0, 0, time.UTC)
	require.Equal(t, "04 Oktober 2026", formatWODate(&issuedAt))
}
