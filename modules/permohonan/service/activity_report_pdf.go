package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/phpdave11/gofpdf"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// woActivityNodes keep their own WO letters; the activity report layout below
// applies to every other workflow node, including wo_pdkb.
var woActivityNodes = map[workflow.Code]bool{
	workflow.WOTiang: true, workflow.WOKonstruksi: true, workflow.WOAPP: true,
}

var activityNames = map[workflow.Code]string{
	workflow.Permohonan:      "Permohonan PB/PD",
	workflow.Survei:          "Survei Perluasan Jaringan",
	workflow.RAB:             "RAB, KKO & KKF",
	workflow.KebutuhanTiang:  "Kebutuhan Tiang",
	workflow.Prioritas:       "Verifikasi Prioritas Pekerjaan",
	workflow.Perluasan:       "Permohonan Perluasan",
	workflow.NPS:             "Delegasi PK NPS",
	workflow.WOTiang:         "WO Vendor Tiang",
	workflow.WOKonstruksi:    "WO Vendor Konstruksi",
	workflow.WOAPP:           "WO Vendor APP",
	workflow.Reservasi:       "Reservasi Material",
	workflow.Tera:            "Perakitan dan Tera APP",
	workflow.WOPDKB:          "WO PDKB",
	workflow.PKVendor:        "PK Vendor",
	workflow.PemasanganTiang: "Pelaksanaan Pemasangan Tiang",
	workflow.Konstruksi:      "Pelaksanaan Konstruksi",
	workflow.DokumentasiPDKB: "Dokumentasi PDKB",
	workflow.Energize:        "Pengoperasian Jaringan Listrik",
	workflow.SRAPP:           "Pemasangan SR/APP dan Penyalaan",
	workflow.PDL:             "Entri dan Mutasi PDL",
	workflow.AIL:             "Arsip AIL-Updating DIJ",
	workflow.Selesai:         "Selesai",
}

var stageNames = map[int16]string{
	1: "Permohonan PB/PD", 2: "Survei", 3: "Perencanaan Perluasan", 4: "Pra Pelaksanaan Konstruksi",
	5: "Pelaksanaan Konstruksi", 6: "Energize Jaringan", 7: "Penutupan",
}

var activityStatusNames = map[string]string{
	string(workflow.Completed): "Selesai", string(workflow.InProgress): "Sedang berjalan",
	string(workflow.Available): "Perlu tindakan", string(workflow.Locked): "Belum dimulai",
	string(workflow.Skipped): "Tidak berlaku",
}

var tarifNames = map[string]string{
	rbac.TarifRumahTangga: "Rumah Tangga", rbac.TarifSosial: "Sosial", rbac.TarifBisnis: "Bisnis",
	rbac.TarifIndustri: "Industri", rbac.TarifPemerintah: "Pemerintah",
}

// payloadFields lists known payload keys in display order with their labels.
var payloadFields = []struct{ key, label string }{
	{"nomor_surat", "Nomor surat"},
	{"surveyed_at", "Tanggal survei"},
	{"kebutuhan_tiang", "Kebutuhan tiang"},
	{"priority", "Prioritas"},
	{"nps_delegation_status", "Status delegasi NPS"},
	{"tanggal_pdkb", "Tanggal PDKB"},
	{"estimasi_tanggal_selesai", "Estimasi tanggal selesai"},
	{"perlu_pdkb", "Perlu PDKB"},
	{"workflow_node", "Jenis pekerjaan"},
	{"energize_status", "Status penyalaan"},
	{"operation_result", "Hasil pengoperasian"},
	{"location_coordinates", "Koordinat lokasi"},
	{"notes", "Catatan"},
	{"migration_reason", "Keterangan"},
}

var payloadValueNames = map[string]map[string]string{
	"nps_delegation_status": {"delegated": "Didelegasikan", "returned": "Dikembalikan"},
	"energize_status":       {"temporary": "Sementara", "permanent": "Permanen"},
	"migration_reason":      {"nps_already_completed": "Dilewati otomatis karena delegasi NPS sudah selesai"},
}

var isoDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// generateActivityPDF renders an unsigned activity report in the WO letter
// style: PLN letterhead, label-colon-value rows, and a generated-by footer.
func generateActivityPDF(data activityPDFData) ([]byte, error) {
	rows, err := activityPayloadRows(data.Payload)
	if err != nil {
		return nil, err
	}
	generatedAt := data.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now()
	}
	name := activityName(data.WorkflowNode)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Laporan Aktivitas - "+name, true)
	pdf.SetMargins(17, 12, 17)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("Go", "B", gobold.TTF)
	pdf.RegisterImageOptionsReader("pln-wordmark", gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(plnWordmark))
	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("load activity PDF assets: %w", err)
	}
	pdf.AliasNbPages("{nb}")
	pdf.SetFooterFunc(func() { writeActivityFooter(pdf, generatedAt) })
	pdf.AddPage()

	writeWOLetterhead(pdf, vendorWOPDFData{})
	pdf.SetXY(17, 44)
	pdf.SetFont("Go", "B", 12)
	pdf.CellFormat(0, 6, "LAPORAN AKTIVITAS", "", 1, "L", false, 0, "")
	pdf.SetFont("Go", "B", 10)
	pdf.MultiCell(0, 5, safePDFText(name), "", "L", false)
	pdf.Ln(5)

	writeActivitySection(pdf, "Identitas Permohonan")
	for _, row := range activityIdentityRows(data) {
		writeActivityRow(pdf, row[0], row[1])
	}
	pdf.Ln(4)

	writeActivitySection(pdf, "Data Aktivitas")
	if len(rows) == 0 {
		writeActivityText(pdf, "Belum ada data aktivitas tersimpan.")
	}
	for _, row := range rows {
		writeActivityRow(pdf, row[0], row[1])
	}
	pdf.Ln(4)

	writeActivitySection(pdf, "Lampiran")
	if len(data.Attachments) == 0 {
		writeActivityText(pdf, "Tidak ada lampiran.")
	}
	for index, filename := range data.Attachments {
		writeActivityText(pdf, fmt.Sprintf("%d. %s", index+1, filename))
	}

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("generate activity PDF: %w", err)
	}
	return output.Bytes(), nil
}

func activityName(code string) string {
	if name, ok := activityNames[workflow.Code(code)]; ok {
		return name
	}
	return humanizePayloadKey(code)
}

func activityIdentityRows(data activityPDFData) [][2]string {
	rows := [][2]string{
		{"Nomor Permohonan", data.NoPermohonan},
		{"Nama Pelanggan", data.PelangganNama},
		{"Alamat", data.PelangganAlamat},
		{"Jenis Permohonan", data.JenisPermohonan},
		{"Jenis Sambungan", data.JenisSambungan},
		{"ULP/Unit", data.UlpUnit},
	}
	if value := tarifDaya(data.Tarif, data.GolonganTarif, data.DayaBaru); value != "" {
		rows = append(rows, [2]string{"Tarif/Daya", value})
	}
	if data.DayaLama != nil || data.GolonganTarifLama != nil {
		rows = append(rows, [2]string{"Tarif/Daya lama", tarifDaya(nil, data.GolonganTarifLama, data.DayaLama)})
	}
	stage := fmt.Sprint(data.StageNumber)
	if name, ok := stageNames[data.StageNumber]; ok {
		stage += " - " + name
	}
	rows = append(rows, [2]string{"Tahap", stage})
	status := data.Status
	if name, ok := activityStatusNames[status]; ok {
		status = name
	}
	rows = append(rows, [2]string{"Status aktivitas", status})
	if completedAt, err := time.Parse(time.RFC3339, data.CompletedAt); err == nil {
		rows = append(rows, [2]string{"Waktu selesai", formatIndonesianDateTime(completedAt)})
	}
	if data.CompletedBy != "" {
		rows = append(rows, [2]string{"Dikerjakan oleh", data.CompletedBy})
	}
	return rows
}

func tarifDaya(tarif, golongan *string, daya *int64) string {
	parts := make([]string, 0, 3)
	if tarif != nil && *tarif != "" {
		if name, ok := tarifNames[*tarif]; ok {
			parts = append(parts, name)
		} else {
			parts = append(parts, *tarif)
		}
	}
	if golongan != nil && *golongan != "" {
		parts = append(parts, *golongan)
	}
	if daya != nil {
		parts = append(parts, formatVA(*daya))
	}
	return strings.Join(parts, " / ")
}

// activityPayloadRows turns a node payload into labelled rows, omitting
// technical identifiers such as document_ids and UUID values.
func activityPayloadRows(payload json.RawMessage) ([][2]string, error) {
	if len(payload) == 0 {
		return nil, nil
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, fmt.Errorf("decode activity payload: %w", err)
	}
	object, ok := decoded.(map[string]any)
	if !ok {
		var rows [][2]string
		appendPayloadRows(&rows, "Data", "", decoded)
		return rows, nil
	}
	var rows [][2]string
	known := make(map[string]bool, len(payloadFields))
	for _, field := range payloadFields {
		known[field.key] = true
		if value, ok := object[field.key]; ok {
			appendPayloadRows(&rows, field.label, field.key, value)
		}
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		if !known[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		appendPayloadRows(&rows, sentenceCaseKey(key), key, object[key])
	}
	return rows, nil
}

func appendPayloadRows(rows *[][2]string, label, key string, value any) {
	if technicalPayloadKey(key) {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		if key == "location_coordinates" {
			latitude, latOK := typed["latitude"].(float64)
			longitude, lonOK := typed["longitude"].(float64)
			if latOK && lonOK {
				*rows = append(*rows, [2]string{label, fmt.Sprintf("%.6f, %.6f", latitude, longitude)})
			}
			return
		}
		keys := make([]string, 0, len(typed))
		for child := range typed {
			keys = append(keys, child)
		}
		sort.Strings(keys)
		for _, child := range keys {
			appendPayloadRows(rows, label+" - "+sentenceCaseKey(child), child, typed[child])
		}
	case []any:
		scalars := make([]string, 0, len(typed))
		for index, item := range typed {
			if _, nested := item.(map[string]any); nested {
				appendPayloadRows(rows, fmt.Sprintf("%s %d", label, index+1), "", item)
				continue
			}
			if text, ok := payloadScalar(key, item); ok {
				scalars = append(scalars, text)
			}
		}
		if len(scalars) > 0 {
			*rows = append(*rows, [2]string{label, strings.Join(scalars, ", ")})
		}
	default:
		if text, ok := payloadScalar(key, value); ok {
			*rows = append(*rows, [2]string{label, text})
		}
	}
}

func payloadScalar(key string, value any) (string, bool) {
	switch typed := value.(type) {
	case nil:
		return "-", true
	case bool:
		if typed {
			return "Ya", true
		}
		return "Tidak", true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case string:
		if _, err := uuid.Parse(typed); err == nil {
			return "", false
		}
		if names, ok := payloadValueNames[key]; ok {
			if name, ok := names[typed]; ok {
				return name, true
			}
		}
		if key == "workflow_node" {
			return activityName(typed), true
		}
		if isoDatePattern.MatchString(typed) {
			if date, err := time.Parse("2006-01-02", typed); err == nil {
				return formatIndonesianDate(date), true
			}
		}
		if date, err := time.Parse(time.RFC3339, typed); err == nil {
			return formatIndonesianDateTime(date), true
		}
		if strings.TrimSpace(typed) == "" {
			return "-", true
		}
		return typed, true
	default:
		return fmt.Sprint(typed), true
	}
}

func technicalPayloadKey(key string) bool {
	return key == "id" || strings.HasSuffix(key, "_id") || strings.HasSuffix(key, "_ids")
}

func sentenceCaseKey(key string) string {
	words := strings.Fields(strings.ToLower(humanizePayloadKey(key)))
	if len(words) == 0 {
		return key
	}
	text := strings.Join(words, " ")
	return strings.ToUpper(text[:1]) + text[1:]
}

func formatVA(value int64) string {
	digits := strconv.FormatInt(value, 10)
	var b strings.Builder
	for index, digit := range digits {
		if index > 0 && (len(digits)-index)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(digit)
	}
	return b.String() + " VA"
}

var indonesianMonths = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

// formatIndonesianDate renders e.g. "5 November 2026" (no zero padding,
// unlike formatWODate which the WO letters keep).
func formatIndonesianDate(value time.Time) string {
	return fmt.Sprintf("%d %s %d", value.Day(), indonesianMonths[int(value.Month())-1], value.Year())
}

// formatIndonesianDateTime renders a timestamp in WIB, e.g. "1 November 2026 14:32 WIB".
func formatIndonesianDateTime(value time.Time) string {
	local := value.In(wib)
	return fmt.Sprintf("%s %02d:%02d WIB", formatIndonesianDate(local), local.Hour(), local.Minute())
}

func writeActivitySection(pdf *gofpdf.Fpdf, title string) {
	ensureActivitySpace(pdf, 14)
	pdf.SetX(17)
	pdf.SetFont("Go", "B", 9.5)
	pdf.CellFormat(0, 5.5, title, "", 1, "L", false, 0, "")
	pdf.Ln(1)
}

func writeActivityRow(pdf *gofpdf.Fpdf, label, value string) {
	ensureActivitySpace(pdf, 5)
	if strings.TrimSpace(value) == "" {
		value = "-"
	}
	y := pdf.GetY()
	pdf.SetXY(17, y)
	pdf.SetFont("Go", "", 9)
	pdf.CellFormat(42, 4.5, label, "", 0, "L", false, 0, "")
	pdf.CellFormat(4, 4.5, ":", "", 0, "C", false, 0, "")
	pdf.MultiCell(130, 4.5, safePDFText(value), "", "L", false)
	if pdf.GetY() < y+4.5 {
		pdf.SetY(y + 4.5)
	}
	pdf.Ln(0.8)
}

func writeActivityText(pdf *gofpdf.Fpdf, text string) {
	ensureActivitySpace(pdf, 5)
	pdf.SetX(17)
	pdf.SetFont("Go", "", 9)
	pdf.MultiCell(176, 4.5, safePDFText(text), "", "L", false)
	pdf.Ln(0.8)
}

// ensureActivitySpace starts a new page before a row that would otherwise be
// split from its label at the auto page-break line.
func ensureActivitySpace(pdf *gofpdf.Fpdf, height float64) {
	_, pageHeight := pdf.GetPageSize()
	_, _, _, bottom := pdf.GetMargins()
	if pdf.GetY()+height > pageHeight-bottom {
		pdf.AddPage()
	}
}

// writeActivityFooter replaces the WO address and paraf footer: reports are
// generated documents and carry no signature or initials.
func writeActivityFooter(pdf *gofpdf.Fpdf, generatedAt time.Time) {
	pdf.SetXY(17, -13.5)
	pdf.SetFont("Go", "", 7)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(146, 3.6, "Dokumen ini dihasilkan otomatis oleh aplikasi Colabora pada "+formatIndonesianDateTime(generatedAt)+".", "", 0, "L", false, 0, "")
	pdf.CellFormat(30, 3.6, fmt.Sprintf("Halaman %d dari {nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}
