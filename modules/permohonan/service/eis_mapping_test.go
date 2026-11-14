package service

import (
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEISMappingUsesApprovedColumns(t *testing.T) {
	row := map[string]string{"NAMA": "Synthetic Customer", "NOTELP_PEMOHON": "080000000000", "ALAMAT": "Synthetic Address", "NAMAUP": "ULP Taman", "TARIF": "R1", "DAYA": "2200", "TOTALBIAYA": "100000", "TGLBAYAR": "2026-11-14", "JENIS_TRANSAKSI": "PASANG BARU", "PAKET_SAR": "Pascabayar"}
	first, err := mapEISRow(row)
	require.NoError(t, err)
	require.Equal(t, "pascabayar", first.JenisPembayaran)
	require.Empty(t, first.JenisSambungan)
	require.Nil(t, first.Tarif)
	row["NOAGENDA"] = "ignored"
	row["NOTELP_HP_PEMOHON"] = "ignored"
	second, err := mapEISRow(row)
	require.NoError(t, err)
	require.Equal(t, first, second)
	row["TGLBAYAR"] = "46337.5"
	third, err := mapEISRow(row)
	require.NoError(t, err)
	require.Equal(t, "2026-11-11", third.TanggalPembayaran)
	row["PAKET_SAR"] = "unknown"
	_, err = mapEISRow(row)
	require.Error(t, err)
}

func TestDraftContinuationRequiresCompletedFields(t *testing.T) {
	p := entities.Permohonan{Status: "draft", JenisPermohonan: rbac.JenisPermohonanPasangBaru,
		PelangganNama: "Synthetic Customer", PelangganAlamat: "Synthetic Address",
		PelangganNoHp: "080000000000"}
	req := draftCreateRequest(p)
	require.Empty(t, req.TanggalPembayaran)
	require.Error(t, validateDraftForContinue(req))
	category, group := rbac.TarifRumahTangga, "R1"
	amount, power := int64(1000), int64(2200)
	req.Tarif, req.GolonganTarif = &category, &group
	req.NominalPembayaran, req.DayaBaru = &amount, &power
	req.TanggalPembayaran = "2026-11-14"
	req.JenisSambungan = rbac.JenisSambunganJTR
	req.JenisPembayaran = "pascabayar"
	require.NoError(t, validateDraftForContinue(req))
}
