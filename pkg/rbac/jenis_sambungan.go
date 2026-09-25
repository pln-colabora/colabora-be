package rbac

const (
	JenisSambunganJTR          = "JTR"
	JenisSambunganJTMGardu     = "JTM/Gardu"
	JenisSambunganPlgTmKurang5 = "PLG TM <5 GWNG"
	JenisSambunganPlgTmLebih5  = "PLG TM >5 GWNG"
)

var AllJenisSambungan = []string{
	JenisSambunganJTR,
	JenisSambunganJTMGardu,
	JenisSambunganPlgTmKurang5,
	JenisSambunganPlgTmLebih5,
}

const (
	JenisPermohonanPasangBaru    = "Pasang Baru (PB)"
	JenisPermohonanPerubahanDaya = "Perubahan Daya (PD)"
)

var AllJenisPermohonan = []string{
	JenisPermohonanPasangBaru,
	JenisPermohonanPerubahanDaya,
}

const (
	TarifRumahTangga = "rumah_tangga"
	TarifSosial      = "sosial"
	TarifBisnis      = "bisnis"
	TarifIndustri    = "industri"
	TarifPemerintah  = "pemerintah"
)

var AllTarif = []string{
	TarifRumahTangga,
	TarifSosial,
	TarifBisnis,
	TarifIndustri,
	TarifPemerintah,
}
