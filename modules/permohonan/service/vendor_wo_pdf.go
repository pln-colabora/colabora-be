package service

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/phpdave11/gofpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

//go:embed assets/pln_wordmark.png
var plnWordmark []byte

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
	IssuerRole      string
}

// generateVendorWOPDF creates a private, unsigned work-order letter. The
// letter follows the operational WO correspondence layout while keeping fields
// that are unavailable in COLABORA blank.
func generateVendorWOPDF(data vendorWOPDFData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(data.Title, false)
	pdf.SetMargins(17, 12, 17)
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("Go", "B", gobold.TTF)
	pdf.RegisterImageOptionsReader("pln-wordmark", gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(plnWordmark))
	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("load WO PDF assets: %w", err)
	}
	pdf.AddPage()

	writeWOLetterhead(pdf, data)
	writeWOMetadata(pdf, data)
	writeWORecipient(pdf, data)
	writeWOBody(pdf, data)
	writeWOSignature(pdf, data)
	writeWOFooter(pdf)

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("generate WO PDF: %w", err)
	}
	return output.Bytes(), nil
}

func writeWOLetterhead(pdf *gofpdf.Fpdf, data vendorWOPDFData) {
	pdf.ImageOptions("pln-wordmark", 154, 11, 38, 0, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	pdf.SetXY(154, 26)
	pdf.SetFont("Go", "", 9.5)
	pdf.CellFormat(38, 4.8, "UID JAWA TIMUR", "", 1, "R", false, 0, "")
	pdf.SetX(154)
	pdf.SetFont("Go", "B", 11)
	pdf.CellFormat(38, 4.8, "UP3 SBY BARAT", "", 1, "R", false, 0, "")
	pdf.SetY(48)
}

func writeWOMetadata(pdf *gofpdf.Fpdf, data vendorWOPDFData) {
	leftX, rightX := 28.0, 123.0
	rows := [][2]string{
		{"Nomor", ""},
		{"Lampiran", ""},
		{"Sifat", ""},
		{"Hal", woSubject(data.Title)},
	}

	pdf.SetFont("Go", "", 9)
	for _, row := range rows {
		y := pdf.GetY()
		pdf.SetXY(leftX, y)
		pdf.CellFormat(22, 4.5, row[0], "", 0, "L", false, 0, "")
		pdf.CellFormat(4, 4.5, ":", "", 0, "C", false, 0, "")
		pdf.MultiCell(61, 4.5, safePDFText(row[1]), "", "L", false)
		if pdf.GetY() < y+4.5 {
			pdf.SetY(y + 4.5)
		}
	}

	pdf.SetXY(rightX, 43)
	pdf.SetFont("Go", "", 9)
	pdf.CellFormat(64, 5, formatWODate(&data.IssuedAt), "", 1, "R", false, 0, "")
}

func writeWORecipient(pdf *gofpdf.Fpdf, data vendorWOPDFData) {
	const recipientX = 123.0
	pdf.SetXY(recipientX, 57)
	pdf.SetFont("Go", "", 9)
	pdf.CellFormat(64, 4.5, "Kepada", "", 0, "L", false, 0, "")
	pdf.SetXY(recipientX, 61.5)
	pdf.CellFormat(64, 4.5, "Yth.", "", 0, "L", false, 0, "")
	pdf.SetXY(recipientX, 66)
	pdf.SetFont("Go", "B", 9)
	pdf.MultiCell(64, 4.5, safePDFText(data.VendorNama), "", "L", false)
	pdf.SetFont("Go", "", 8)
	if data.VendorEmail != "" {
		pdf.MultiCell(64, 4, safePDFText(data.VendorEmail), "", "L", false)
	}
	if data.VendorTelepon != "" {
		pdf.MultiCell(64, 4, safePDFText(data.VendorTelepon), "", "L", false)
	}
}

func writeWOBody(pdf *gofpdf.Fpdf, data vendorWOPDFData) {
	pdf.SetY(83)
	pdf.SetX(49)
	pdf.SetFont("Go", "", 9)
	pdf.CellFormat(0, 5, "u.p. Yth. Direktur", "", 1, "L", false, 0, "")
	pdf.Ln(5)

	pdf.SetX(49)
	pdf.MultiCell(140, 4.6, "Sehubungan dengan pelaksanaan pekerjaan, kami mohon agar Saudara melaksanakan pekerjaan dengan ketentuan sebagai berikut:", "", "L", false)
	pdf.Ln(4)

	scope := safePDFText(data.Scope)
	if scope == "" {
		scope = "____________________________"
	}
	writeWONumberedInstruction(pdf, "1.", "Melaksanakan "+scope+" sesuai data dan dokumen teknis terlampir.")
	writeWONumberedInstruction(pdf, "2.", "Sebelum pelaksanaan pekerjaan, menyampaikan jadwal kepada PIC pengawas pekerjaan.")
	writeWONumberedInstruction(pdf, "3.", "Mematuhi standar keselamatan dan ketentuan kerja yang berlaku.")
	pdf.Ln(6)

	pdf.SetX(49)
	pdf.MultiCell(140, 4.6, "Demikian disampaikan, atas perhatian dan kerja samanya kami ucapkan terima kasih.", "", "L", false)
}

func writeWONumberedInstruction(pdf *gofpdf.Fpdf, number, text string) {
	startY := pdf.GetY()
	pdf.SetX(49)
	pdf.SetFont("Go", "", 9)
	pdf.CellFormat(8, 4.6, number, "", 0, "L", false, 0, "")
	pdf.SetXY(57, startY)
	pdf.MultiCell(132, 4.6, text, "", "L", false)
	pdf.Ln(1)
}

func writeWOSignature(pdf *gofpdf.Fpdf, data vendorWOPDFData) {
	if pdf.GetY() < 190 {
		pdf.SetY(190)
	}
	pdf.SetX(108)
	pdf.SetFont("Go", "B", 9)
	pdf.MultiCell(79, 4.5, "PEJABAT BERWENANG", "", "C", false)
	pdf.SetX(108)
	pdf.SetFont("Go", "", 8)
	pdf.CellFormat(79, 4.5, safePDFText(data.IssuerRole), "", 1, "C", false, 0, "")
	pdf.Ln(18)
	pdf.SetDrawColor(50, 50, 50)
	pdf.SetLineWidth(.3)
	pdf.Line(120, pdf.GetY(), 175, pdf.GetY())
	pdf.SetX(108)
	pdf.SetFont("Go", "", 8)
	pdf.CellFormat(79, 4.5, "Nama dan Jabatan", "", 1, "C", false, 0, "")
}

func writeWOFooter(pdf *gofpdf.Fpdf) {
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetXY(17, 273)
	pdf.SetFont("Go", "", 7.5)
	pdf.CellFormat(120, 4, "Jl. Raya Taman No 48 D, Sepanjang, Surabaya Barat 61257", "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 126, 181)
	pdf.CellFormat(3, 4, "T", "", 0, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(15, 4, " (031) 123", "", 0, "L", false, 0, "")
	pdf.SetTextColor(0, 126, 181)
	pdf.CellFormat(3, 4, "W", "", 0, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(30, 4, " www.pln.co.id", "", 0, "L", false, 0, "")
	pdf.SetXY(164, 277)
	pdf.CellFormat(12, 4, "Paraf", "", 0, "L", false, 0, "")
	pdf.SetDrawColor(50, 50, 50)
	pdf.SetLineWidth(.3)
	pdf.Line(176, 281, 193, 281)
}

func woSubject(title string) string {
	switch title {
	case "SURAT PERINTAH KERJA VENDOR TIANG":
		return "Perintah Kerja / Work Order Pemasangan Tiang"
	case "SURAT PERINTAH KERJA VENDOR KONSTRUKSI":
		return "Perintah Kerja / Work Order Konstruksi Jaringan"
	case "SURAT PERINTAH KERJA VENDOR APP":
		return "Perintah Kerja / Work Order APP"
	default:
		return "Perintah Kerja / Work Order"
	}
}

func formatWODate(value *time.Time) string {
	if value == nil || value.IsZero() {
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
