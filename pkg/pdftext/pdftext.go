// Package pdftext lets tests assert on text drawn by gofpdf without a full PDF
// parser. It inflates every content stream and looks for the string as gofpdf
// writes it: escaped, either single-byte or UTF-16BE (UTF-8 fonts).
package pdftext

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

// streamPattern locates each stream by its declared /Length so binary data
// that happens to contain line breaks is never truncated.
var streamPattern = regexp.MustCompile(`/Length (\d+)[^>]*>>\s*stream\r?\n`)

// Contains reports whether text appears, unsplit, in any content stream of pdf.
func Contains(pdf []byte, text string) bool {
	needles := [][]byte{[]byte(escape(text)), []byte(escape(utf16BE(text)))}
	for _, match := range streamPattern.FindAllSubmatchIndex(pdf, -1) {
		length, err := strconv.Atoi(string(pdf[match[2]:match[3]]))
		if err != nil || match[1]+length > len(pdf) {
			continue
		}
		content := pdf[match[1] : match[1]+length]
		if reader, err := zlib.NewReader(bytes.NewReader(content)); err == nil {
			if inflated, err := io.ReadAll(reader); err == nil {
				content = inflated
			}
		}
		for _, needle := range needles {
			if bytes.Contains(content, needle) {
				return true
			}
		}
	}
	return false
}

func utf16BE(text string) string {
	var b strings.Builder
	for _, unit := range utf16.Encode([]rune(text)) {
		b.WriteByte(byte(unit >> 8))
		b.WriteByte(byte(unit))
	}
	return b.String()
}

func escape(text string) string {
	return strings.NewReplacer(`\`, `\\`, "(", `\(`, ")", `\)`, "\r", `\r`).Replace(text)
}
