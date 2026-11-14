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

// mapEISRow reads only the approved EIS columns. Connection type, tariff
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
	req := dto.PermohonanCreateRequest{
		PelangganNama: get("NAMA"), PelangganNoHp: get("NOTELP_PEMOHON"),
		PelangganAlamat: get("ALAMAT"), UlpUnit: optional("NAMAUP"),
		GolonganTarif: optional("TARIF"), GolonganTarifLama: optional("TARIF_LAMA"),
		JenisPembayaran: strings.ToLower(get("PAKET_SAR")),
	}
	switch strings.ToUpper(get("JENIS_TRANSAKSI")) {
	case "PASANG BARU", "PASANG BARU (PB)", "PB":
		req.JenisPermohonan = rbac.JenisPermohonanPasangBaru
	case "PERUBAHAN DAYA", "PERUBAHAN DAYA (PD)", "PD":
		req.JenisPermohonan = rbac.JenisPermohonanPerubahanDaya
	default:
		return req, fmt.Errorf("JENIS_TRANSAKSI is not supported")
	}
	if req.JenisPembayaran != "pascabayar" && req.JenisPembayaran != "prabayar" {
		return req, fmt.Errorf("PAKET_SAR must be Pascabayar or Prabayar")
	}
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
