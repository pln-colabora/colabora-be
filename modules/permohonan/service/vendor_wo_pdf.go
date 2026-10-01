package service

import (
	"bytes"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/phpdave11/gofpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

type vendorWOPDFData struct {
	Title           string
	NoPermohonan    string
	JenisPermohonan string
	JenisSambungan  string
	UlpUnit         string
	PelangganNama   string
	PelangganAlamat string
	PelangganNoHp   string
	Tarif           string
	DayaLama        string
	DayaBaru        string
	VendorNama      string
	VendorEmail     string
	VendorTelepon   string
	IssuedAt        time.Time
	Scope           string
	Coordinates     string
	EstimasiSelesai string
	PerluPDKB       string
	Notes           string
}

// generateVendorWOPDF creates a private unsigned WO artefact. Missing source
// values are deliberately rendered as empty cells so the existing WO request
// payload does not need to change.
func generateVendorWOPDF(data vendorWOPDFData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(data.Title, false)
	pdf.SetMargins(17, 15, 17)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("Go", "B", gobold.TTF)
	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("load WO PDF fonts: %w", err)
	}
	pdf.AddPage()

	pdf.SetFont("Go", "B", 10)
	pdf.CellFormat(0, 6, "PT PLN (PERSERO)", "", 1, "C", false, 0, "")
	pdf.SetFont("Go", "", 9)
	pdf.CellFormat(0, 5, safePDFText(data.UlpUnit), "", 1, "C", false, 0, "")
	pdf.Ln(3)
	pdf.SetDrawColor(24, 62, 112)
	pdf.SetLineWidth(.7)
	pdf.Line(17, pdf.GetY(), 193, pdf.GetY())
	pdf.Ln(4)
	pdf.SetFont("Go", "B", 15)
	pdf.CellFormat(0, 8, data.Title, "", 1, "C", false, 0, "")
	pdf.SetFont("Go", "", 9)
	pdf.CellFormat(0, 5, "DRAFT - BELUM DITANDATANGANI", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	writeWOSection(pdf, "IDENTITAS PERINTAH KERJA")
	writeWOFields(pdf, [][2]string{
		{"Nomor WO", ""},
		{"Nomor Permohonan", data.NoPermohonan},
		{"Jenis Permohonan", data.JenisPermohonan},
		{"Jenis Sambungan", data.JenisSambungan},
		{"Tanggal Terbit", formatWODate(&data.IssuedAt)},
	})

	writeWOSection(pdf, "DATA PELANGGAN DAN LOKASI PEKERJAAN")
	writeWOFields(pdf, [][2]string{
		{"Nama Pelanggan", data.PelangganNama},
		{"Alamat Pekerjaan", data.PelangganAlamat},
		{"Nomor Telepon", data.PelangganNoHp},
		{"ULP", data.UlpUnit},
		{"Tarif", data.Tarif},
		{"Daya Lama (VA)", data.DayaLama},
		{"Daya Baru (VA)", data.DayaBaru},
	})

	writeWOSection(pdf, "PENUGASAN VENDOR")
	writeWOFields(pdf, [][2]string{
		{"Nama Vendor", data.VendorNama},
		{"Email Vendor", data.VendorEmail},
		{"Telepon Vendor", data.VendorTelepon},
		{"Referensi Kontrak", ""},
		{"Pengawas / Kontak Lapangan", ""},
	})

	writeWOSection(pdf, "URAIAN PEKERJAAN")
	writeWOFields(pdf, [][2]string{
		{"Ruang Lingkup", data.Scope},
		{"Koordinat Lokasi", data.Coordinates},
		{"Volume Pekerjaan", ""},
		{"Nilai / Harga Pekerjaan", ""},
		{"Referensi Gambar Teknik", ""},
		{"SLA Vendor", ""},
		{"Estimasi Selesai", data.EstimasiSelesai},
		{"Memerlukan PDKB", data.PerluPDKB},
		{"Catatan", data.Notes},
	})

	pdf.Ln(2)
	pdf.SetFont("Go", "", 8)
	pdf.MultiCell(0, 4, "Kolom yang belum memiliki data pada sistem sengaja dikosongkan. Lampiran pendukung tidak digabungkan ke PDF ini.", "", "L", false)
	pdf.Ln(5)
	writeWOSignatureBlocks(pdf)

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("generate WO PDF: %w", err)
	}
	return output.Bytes(), nil
}

func writeWOSection(pdf *gofpdf.Fpdf, title string) {
	if pdf.GetY() > 250 {
		pdf.AddPage()
	}
	pdf.SetFillColor(231, 239, 248)
	pdf.SetFont("Go", "B", 9)
	pdf.CellFormat(0, 7, title, "", 1, "L", true, 0, "")
	pdf.Ln(1)
}

func writeWOFields(pdf *gofpdf.Fpdf, fields [][2]string) {
	labelWidth := 57.0
	for _, field := range fields {
		if pdf.GetY() > 266 {
			pdf.AddPage()
		}
		startY := pdf.GetY()
		pdf.SetFont("Go", "B", 8)
		pdf.MultiCell(labelWidth, 5, safePDFText(field[0]), "", "L", false)
		labelHeight := pdf.GetY() - startY
		pdf.SetXY(17+labelWidth, startY)
		pdf.SetFont("Go", "", 8)
		pdf.MultiCell(176-labelWidth, 5, safePDFText(field[1]), "", "L", false)
		valueHeight := pdf.GetY() - startY
		if labelHeight > valueHeight {
			pdf.SetY(startY + labelHeight)
		}
		pdf.SetDrawColor(220, 226, 232)
		pdf.Line(17, pdf.GetY(), 193, pdf.GetY())
		pdf.Ln(1)
	}
}

func writeWOSignatureBlocks(pdf *gofpdf.Fpdf) {
	if pdf.GetY() > 224 {
		pdf.AddPage()
	}
	leftX, rightX := 24.0, 112.0
	y := pdf.GetY()
	pdf.SetFont("Go", "", 8)
	pdf.SetXY(leftX, y)
	pdf.CellFormat(68, 5, "Dibuat oleh,", "", 0, "C", false, 0, "")
	pdf.SetXY(rightX, y)
	pdf.CellFormat(68, 5, "Diterima oleh,", "", 1, "C", false, 0, "")
	pdf.Ln(22)
	pdf.SetDrawColor(80, 80, 80)
	pdf.Line(leftX+8, pdf.GetY(), leftX+60, pdf.GetY())
	pdf.Line(rightX+8, pdf.GetY(), rightX+60, pdf.GetY())
	pdf.SetY(pdf.GetY() + 2)
	pdf.SetXY(leftX, pdf.GetY())
	pdf.CellFormat(68, 5, "Nama / Jabatan", "", 0, "C", false, 0, "")
	pdf.SetXY(rightX, pdf.GetY())
	pdf.CellFormat(68, 5, "Nama / Jabatan", "", 1, "C", false, 0, "")
}

func formatWODate(value *time.Time) string {
	if value == nil {
		return ""
	}
	months := [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return fmt.Sprintf("%02d %s %d", value.Day(), months[int(value.Month())-1], value.Year())
}

func safePDFText(value string) string {
	return fmtMapControls(value)
}

func fmtMapControls(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return ' '
		}
		return r
	}, value)
}
