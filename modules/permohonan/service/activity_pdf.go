package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/phpdave11/gofpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

type activityPDFData struct {
	NoPermohonan   string
	JenisSambungan string
	WorkflowNode   string
	ActivityNumber *int16
	StageNumber    int16
	Status         string
	CompletedAt    string
	Payload        json.RawMessage
}

func generateActivityPDF(data activityPDFData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Export Jawaban Activity", false)
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("Go", "B", gobold.TTF)
	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("load PDF fonts: %w", err)
	}
	pdf.AddPage()

	pdf.SetFont("Go", "B", 16)
	pdf.MultiCell(0, 9, "Jawaban Activity", "", "L", false)
	pdf.Ln(2)

	writePDFField(pdf, "Nomor Permohonan", data.NoPermohonan, 0)
	writePDFField(pdf, "Jenis Sambungan", data.JenisSambungan, 0)
	writePDFField(pdf, "Workflow Node", data.WorkflowNode, 0)
	if data.ActivityNumber != nil {
		writePDFField(pdf, "Nomor Activity", fmt.Sprint(*data.ActivityNumber), 0)
	}
	writePDFField(pdf, "Tahap", fmt.Sprint(data.StageNumber), 0)
	writePDFField(pdf, "Status", data.Status, 0)
	if data.CompletedAt != "" {
		writePDFField(pdf, "Waktu Selesai", data.CompletedAt, 0)
	}

	pdf.Ln(3)
	pdf.SetFont("Go", "B", 12)
	pdf.MultiCell(0, 7, "Jawaban Form", "", "L", false)
	pdf.Ln(1)

	var payload any
	if len(data.Payload) != 0 {
		if err := json.Unmarshal(data.Payload, &payload); err != nil {
			return nil, fmt.Errorf("decode activity payload: %w", err)
		}
	}
	if payload == nil || isEmptyJSONObject(payload) {
		pdf.SetFont("Go", "", 10)
		pdf.MultiCell(0, 6, "Belum ada jawaban tersimpan.", "", "L", false)
	} else if err := writePDFValue(pdf, "", payload, 0); err != nil {
		return nil, err
	}

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf("generate activity PDF: %w", err)
	}
	return output.Bytes(), nil
}

func writePDFValue(pdf *gofpdf.Fpdf, label string, value any, depth int) error {
	if object, ok := value.(map[string]any); ok {
		if len(object) == 0 {
			writePDFField(pdf, label, "{}", depth)
			return nil
		}
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			childLabel := humanizePayloadKey(key)
			if label != "" {
				childLabel = label + " / " + childLabel
			}
			if err := writePDFValue(pdf, childLabel, object[key], depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if values, ok := value.([]any); ok {
		if len(values) == 0 {
			writePDFField(pdf, label, "[]", depth)
			return nil
		}
		for index, item := range values {
			itemLabel := fmt.Sprintf("%s [%d]", label, index+1)
			if label == "" {
				itemLabel = fmt.Sprintf("Item %d", index+1)
			}
			if err := writePDFValue(pdf, itemLabel, item, depth+1); err != nil {
				return err
			}
		}
		return nil
	}

	text, err := formatPayloadValue(value)
	if err != nil {
		return err
	}
	writePDFField(pdf, label, text, depth)
	return nil
}

func writePDFField(pdf *gofpdf.Fpdf, label, value string, depth int) {
	indent := float64(depth) * 4
	if pdf.GetY() > 270 {
		pdf.AddPage()
	}
	pdf.SetX(18 + indent)
	pdf.SetFont("Go", "B", 9)
	pdf.MultiCell(0, 5, sanitizePDFText(label), "", "L", false)
	pdf.SetX(18 + indent)
	pdf.SetFont("Go", "", 10)
	pdf.MultiCell(0, 5, sanitizePDFText(value), "", "L", false)
	pdf.Ln(1)
}

func formatPayloadValue(value any) (string, error) {
	if value == nil {
		return "-", nil
	}
	if text, ok := value.(string); ok {
		return text, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("format activity payload value: %w", err)
	}
	return string(encoded), nil
}

func isEmptyJSONObject(value any) bool {
	object, ok := value.(map[string]any)
	return ok && len(object) == 0
}

func humanizePayloadKey(key string) string {
	words := strings.FieldsFunc(key, func(r rune) bool { return r == '_' || r == '-' })
	for index, word := range words {
		if word == "" {
			continue
		}
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[index] = string(runes)
	}
	return strings.Join(words, " ")
}

func sanitizePDFText(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return ' '
		}
		return r
	}, value)
}

func safeFilenamePart(value string) string {
	var result strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			result.WriteRune(r)
		} else {
			result.WriteByte('_')
		}
	}
	return strings.Trim(result.String(), "_")
}
