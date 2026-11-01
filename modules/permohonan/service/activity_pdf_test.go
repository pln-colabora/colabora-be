package service

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/pln-colabora/colabora-be/pkg/pdftext"

	"github.com/stretchr/testify/require"
)

func TestGenerateActivityPDFRendersDynamicPayload(t *testing.T) {
	number := int16(2)
	content, err := generateActivityPDF(activityPDFData{
		NoPermohonan:   "PB-2026-00001",
		JenisSambungan: "JTR",
		WorkflowNode:   "survei",
		ActivityNumber: &number,
		StageNumber:    2,
		Status:         "completed",
		Payload:        json.RawMessage(`{"notes":"survei – kondisi aman","alat_ukur":[{"nama":"meter-a","hasil":12.5}],"verified":true}`),
	})

	require.NoError(t, err)
	require.Greater(t, len(content), 100)
	require.Equal(t, "%PDF", string(content[:4]))
}

func TestGenerateActivityPDFHandlesEmptyPayload(t *testing.T) {
	content, err := generateActivityPDF(activityPDFData{
		WorkflowNode: "survei",
		Payload:      json.RawMessage(`{}`),
	})
	require.NoError(t, err)
	require.Equal(t, "%PDF", string(content[:4]))
}

func TestGenerateActivityPDFRejectsMalformedPayload(t *testing.T) {
	_, err := generateActivityPDF(activityPDFData{Payload: json.RawMessage(`{`)})
	require.Error(t, err)
}

func TestGenerateActivityPDFReportLayout(t *testing.T) {
	documentID := "550e8400-e29b-41d4-a716-446655440003"
	content, err := generateActivityPDF(activityPDFData{
		NoPermohonan: "PBPD-2026-0042", JenisPermohonan: "Pasang Baru (PB)", JenisSambungan: "JTR",
		UlpUnit: "ULP Taman", PelangganNama: "PT Sinar Karya", PelangganAlamat: "Jl. Raya Industri 18",
		WorkflowNode: "survei", StageNumber: 2, Status: "completed",
		CompletedAt: "2026-11-05T03:15:00Z", CompletedBy: "Budi (Teknik)",
		Attachments: []string{"foto-lokasi.jpg", "izin-pemilik-tanah.pdf"},
		GeneratedAt: time.Date(2026, 11, 1, 7, 32, 0, 0, time.UTC),
		Payload:     json.RawMessage(`{"surveyed_at":"2026-11-05","notes":"Kondisi aman","document_ids":["` + documentID + `"],"location_coordinates":{"latitude":-7.25,"longitude":112.75}}`),
	})
	require.NoError(t, err)
	for _, text := range []string{
		"LAPORAN AKTIVITAS", "Survei Perluasan Jaringan", "UID JAWA TIMUR", "UP3 SBY BARAT",
		"PBPD-2026-0042", "PT Sinar Karya", "2 - Survei", "Selesai", "5 November 2026 10:15 WIB", "Budi (Teknik)",
		"Tanggal survei", "5 November 2026", "Catatan", "Kondisi aman", "Koordinat lokasi", "-7.250000, 112.750000",
		"1. foto-lokasi.jpg", "2. izin-pemilik-tanah.pdf",
		"Dokumen ini dihasilkan otomatis oleh aplikasi Colabora pada 1 November 2026 14:32 WIB.",
		"Halaman 1 dari 1",
	} {
		require.True(t, pdftext.Contains(content, text), "PDF should contain %q", text)
	}
	for _, text := range []string{"PEJABAT BERWENANG", "Paraf", "Tidak memerlukan tanda tangan", "document_ids", documentID, "surveyed_at"} {
		require.False(t, pdftext.Contains(content, text), "PDF must not contain %q", text)
	}
}

func TestActivityPayloadRowsUseHumanLabels(t *testing.T) {
	rows, err := activityPayloadRows(json.RawMessage(`{"kebutuhan_tiang":false,"priority":"Medium","energize_status":"permanent","operation_result":"Normal","nps_delegation_status":"delegated","workflow_node":"pelaksanaan_konstruksi","perlu_pdkb":true,"tanggal_pdkb":"2026-12-01","vendor_id":"x","custom_field":"nilai"}`))
	require.NoError(t, err)
	require.Equal(t, [][2]string{
		{"Kebutuhan tiang", "Tidak"},
		{"Prioritas", "Medium"},
		{"Status delegasi NPS", "Didelegasikan"},
		{"Tanggal PDKB", "1 Desember 2026"},
		{"Perlu PDKB", "Ya"},
		{"Jenis pekerjaan", "Pelaksanaan Konstruksi"},
		{"Status penyalaan", "Permanen"},
		{"Hasil pengoperasian", "Normal"},
		{"Custom field", "nilai"},
	}, rows)
}

func TestGenerateActivityPDFNumbersEveryPage(t *testing.T) {
	attachments := make([]string, 0, 50)
	for index := range 50 {
		attachments = append(attachments, fmt.Sprintf("lampiran-%02d.pdf", index+1))
	}
	content, err := generateActivityPDF(activityPDFData{NoPermohonan: "PBPD-2026-0043", WorkflowNode: "arsip_ail", Attachments: attachments, Payload: json.RawMessage(`{}`)})
	require.NoError(t, err)
	for _, text := range []string{"Arsip AIL-Updating DIJ", "Belum ada data aktivitas tersimpan.", "Halaman 1 dari 2", "Halaman 2 dari 2", "50. lampiran-50.pdf"} {
		require.True(t, pdftext.Contains(content, text), "PDF should contain %q", text)
	}
}

func TestWOLettersKeepSignatureAndParaf(t *testing.T) {
	content, err := generateVendorWOPDF(vendorWOPDFData{Title: "SURAT PERINTAH KERJA VENDOR TIANG", Scope: "Pekerjaan uji", NomorSurat: "123/WO/2026"})
	require.NoError(t, err)
	for _, text := range []string{"MANAGER PLN UP3 SURABAYA BARAT", "CYNTHIA DEWI ARIYANI", "123/WO/2026", "Paraf", "UP3 SBY BARAT"} {
		require.True(t, pdftext.Contains(content, text), "WO PDF should still contain %q", text)
	}
	legacy, err := generateLegacyActivityPDF(activityPDFData{WorkflowNode: "wo_tiang", Payload: json.RawMessage(`{"notes":"WO"}`)})
	require.NoError(t, err)
	require.True(t, pdftext.Contains(legacy, "Jawaban Activity"))
	require.True(t, woActivityNodes["wo_tiang"] && woActivityNodes["wo_konstruksi"] && woActivityNodes["wo_app"])
}

func TestWOPDKBUsesActivityReportLayout(t *testing.T) {
	require.False(t, woActivityNodes["wo_pdkb"])
	content, err := generateActivityPDF(activityPDFData{
		NoPermohonan: "PBPD-2026-0044", WorkflowNode: "wo_pdkb", StageNumber: 4, Status: "completed",
		GeneratedAt: time.Date(2026, 11, 1, 7, 32, 0, 0, time.UTC),
		Payload:     json.RawMessage(`{"nomor_surat":"045/WO-PDKB/2026","notes":"Jadwal PDKB terkonfirmasi","tanggal_pdkb":"2026-11-12"}`),
	})
	require.NoError(t, err)
	for _, text := range []string{
		"LAPORAN AKTIVITAS", "WO PDKB", "PBPD-2026-0044", "4 - Pra Pelaksanaan Konstruksi",
		"Nomor surat", "045/WO-PDKB/2026", "Tanggal PDKB", "12 November 2026", "Catatan", "Jadwal PDKB terkonfirmasi",
		"Dokumen ini dihasilkan otomatis oleh aplikasi Colabora pada 1 November 2026 14:32 WIB.",
	} {
		require.True(t, pdftext.Contains(content, text), "PDF should contain %q", text)
	}
	for _, text := range []string{"Jawaban Activity", "PEJABAT BERWENANG", "Paraf", "nomor_surat"} {
		require.False(t, pdftext.Contains(content, text), "PDF must not contain %q", text)
	}
}
