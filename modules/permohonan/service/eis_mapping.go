package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
)

// mapEISRow reads only the approved EIS columns, plus NOTELP_HP_PEMOHON as the
// phone fallback when NOTELP_PEMOHON is blank. Connection type, tariff
// category and evidence must be completed by the operator before activation.
func mapEISRow(row map[string]string) (dto.PermohonanCreateRequest, error) {
	get := func(key string) string { return strings.TrimSpace(row[key]) }
	optional := func(key string) *string {
		value := get(key)
		if value == "" {
			return nil
		}
		return &value
	}
	phone := get("NOTELP_PEMOHON")
	if phone == "" {
		phone = get("NOTELP_HP_PEMOHON")
	}
	req := dto.PermohonanCreateRequest{
		PelangganNama: get("NAMA"), PelangganNoHp: phone,
		PelangganAlamat: get("ALAMAT"), UlpUnit: optional("NAMAUP"),
		GolonganTarif: optional("TARIF"), GolonganTarifLama: optional("TARIF_LAMA"),
	}
	switch strings.ToUpper(get("JENIS_TRANSAKSI")) {
	case "PASANG BARU", "PASANG BARU (PB)", "PB":
		req.JenisPermohonan = rbac.JenisPermohonanPasangBaru
	case "PERUBAHAN DAYA", "PERUBAHAN DAYA (PD)", "PD":
		req.JenisPermohonan = rbac.JenisPermohonanPerubahanDaya
	default:
		return req, fmt.Errorf("JENIS_TRANSAKSI is not supported")
	}
	jenisPembayaran, ok := normalizeJenisPembayaran(get("PAKET_SAR"))
	if !ok {
		return req, fmt.Errorf("PAKET_SAR must be Pascabayar or Prabayar")
	}
	req.JenisPembayaran = jenisPembayaran
	for _, field := range []struct {
		key string
		dst **int64
	}{
		{"TOTALBIAYA", &req.NominalPembayaran}, {"DAYA", &req.DayaBaru}, {"DAYA_LAMA", &req.DayaLama},
	} {
		value := get(field.key)
		if value == "" && field.key == "DAYA_LAMA" {
			continue
		}
		number, err := strconv.ParseInt(value, 10, 64)
		if err != nil || number < 0 || (number == 0 && field.key != "DAYA_LAMA") {
			return req, fmt.Errorf("%s must be a positive whole number", field.key)
		}
		if number > 0 {
			*field.dst = &number
		}
	}
	value := get("TGLBAYAR")
	if date, err := time.Parse("2006-01-02", value); err == nil {
		req.TanggalPembayaran = date.Format("2006-01-02")
	} else {
		serial, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(serial) || math.IsInf(serial, 0) || serial < 61 || serial > 2958465 {
			return req, fmt.Errorf("TGLBAYAR must be an Excel date or YYYY-MM-DD")
		}
		req.TanggalPembayaran = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(serial)).Format("2006-01-02")
	}
	return req, nil
}

// normalizeJenisPembayaran maps an EIS PAKET_SAR label such as
// "Prabayar (Prepaid) Satu Arah" to "prabayar" or "pascabayar". A label that
// names both payment kinds, or neither, is rejected rather than guessed.
func normalizeJenisPembayaran(raw string) (string, bool) {
	value := strings.ToLower(raw)
	prepaid := strings.Contains(value, "prabayar") || strings.Contains(value, "prepaid")
	postpaid := strings.Contains(value, "pascabayar") || strings.Contains(value, "postpaid")
	switch {
	case prepaid && !postpaid:
		return "prabayar", true
	case postpaid && !prepaid:
		return "pascabayar", true
	}
	return "", false
}
