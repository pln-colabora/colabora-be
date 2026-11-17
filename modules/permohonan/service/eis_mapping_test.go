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
	row["PAKET_SAR"] = "Prabayar (Prepaid) Satu Arah"
	fourth, err := mapEISRow(row)
	require.NoError(t, err)
	require.Equal(t, "prabayar", fourth.JenisPembayaran)
	require.Equal(t, "080000000000", fourth.PelangganNoHp)
	row["NOTELP_PEMOHON"] = ""
	row["NOTELP_HP_PEMOHON"] = "081111111111"
	fifth, err := mapEISRow(row)
	require.NoError(t, err)
	require.Equal(t, "081111111111", fifth.PelangganNoHp)
}

func TestNormalizeEISGolongan(t *testing.T) {
	tests := []struct {
		raw      string
		golongan string
		token    bool
	}{
		{"R1", "R1", false},
		{"R1T", "R1", true},
		{"R1M", "R1", false},
		{"R1MT", "R1", true},
		{"r1mt", "R1", true},
		{" B2T ", "B2", true},
		{"R-1", "R1", false},
		{"I4", "I4", false},
		{"S1", "S1", false},
		{"", "", false},
		{"X9", "X9", false},
		{"R1TM", "R1TM", false},
	}
	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			golongan, token := normalizeEISGolongan(test.raw)
			require.Equal(t, test.golongan, golongan)
			require.Equal(t, test.token, token)
		})
	}
}

func TestEISMappingNormalizesTokenTariff(t *testing.T) {
	row := map[string]string{"NAMA": "Synthetic Customer", "NOTELP_PEMOHON": "080000000000", "ALAMAT": "Synthetic Address", "NAMAUP": "ULP Taman", "TARIF": "R1MT", "DAYA": "900", "TOTALBIAYA": "100000", "TGLBAYAR": "2026-11-14", "JENIS_TRANSAKSI": "PASANG BARU", "PAKET_SAR": "Prabayar (Prepaid) Satu Arah"}
	req, err := mapEISRow(row)
	require.NoError(t, err)
	require.Equal(t, "R1", *req.GolonganTarif)
	require.Nil(t, req.GolonganTarifLama)

	row["PAKET_SAR"] = "Pascabayar"
	_, err = mapEISRow(row)
	require.EqualError(t, err, "TARIF R1MT is Token (prabayar) but PAKET_SAR is Pascabayar")

	// A change of power may move a Token customer to postpaid; TARIF_LAMA is not checked.
	row["JENIS_TRANSAKSI"], row["TARIF"], row["DAYA"], row["TARIF_LAMA"], row["DAYA_LAMA"] = "PERUBAHAN DAYA", "B2", "16500", "B2T", "10600"
	req, err = mapEISRow(row)
	require.NoError(t, err)
	require.Equal(t, "B2", *req.GolonganTarif)
	require.Equal(t, "B2", *req.GolonganTarifLama)
}

func TestNormalizeJenisPembayaran(t *testing.T) {
	tests := []struct {
		raw  string
		want string
		ok   bool
	}{
		{"Prabayar", "prabayar", true},
		{"PRABAYAR", "prabayar", true},
		{"Prabayar (Prepaid) Satu Arah", "prabayar", true},
		{"prepaid", "prabayar", true},
		{"Pascabayar", "pascabayar", true},
		{"Pascabayar (Postpaid)", "pascabayar", true},
		{"postpaid", "pascabayar", true},
		{"", "", false},
		{"Lainnya", "", false},
		{"Prabayar / Pascabayar", "", false},
		{"Prepaid / Postpaid", "", false},
	}
	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			got, ok := normalizeJenisPembayaran(test.raw)
			require.Equal(t, test.ok, ok)
			require.Equal(t, test.want, got)
		})
	}
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
