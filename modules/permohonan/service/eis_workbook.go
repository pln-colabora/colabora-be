package service

import (
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
)

const maxEISDataRows = 5000

// readEISRows returns the first sheet with its header at index zero. Empty
// positions are retained so validation errors keep their Excel row numbers.
func readEISRows(reader io.ReadSeeker, format string) ([][]string, error) {
	switch format {
	case ".xls":
		book, err := xls.OpenReader(reader, "utf-8")
		if err != nil || book.NumSheets() == 0 {
			return nil, fmt.Errorf("invalid EIS XLS workbook")
		}
		sheet := book.GetSheet(0)
		if sheet == nil || sheet.MaxRow < 1 {
			return nil, fmt.Errorf("EIS sheet has no data")
		}
		if sheet.MaxRow > maxEISDataRows {
			return nil, fmt.Errorf("EIS import exceeds 5000 rows")
		}
		rows := make([][]string, int(sheet.MaxRow)+1)
		for index := range rows {
			row := sheet.Row(index)
			if row == nil {
				continue
			}
			cells := make([]string, row.LastCol())
			for col := range cells {
				cells[col] = cleanEISCell(row.Col(col))
			}
			rows[index] = cells
		}
		return rows, nil
	case ".xlsx":
		book, err := excelize.OpenReader(reader)
		if err != nil {
			return nil, fmt.Errorf("invalid EIS XLSX workbook")
		}
		defer book.Close()
		sheets := book.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("invalid EIS XLSX workbook")
		}
		iterator, err := book.Rows(sheets[0])
		if err != nil {
			return nil, fmt.Errorf("invalid EIS XLSX workbook: %w", err)
		}
		defer iterator.Close()
		rows := make([][]string, 0)
		for iterator.Next() {
			if len(rows) > maxEISDataRows {
				return nil, fmt.Errorf("EIS import exceeds 5000 rows")
			}
			// Raw values keep Excel date serials parseable by mapEISRow.
			cells, err := iterator.Columns(excelize.Options{RawCellValue: true})
			if err != nil {
				return nil, fmt.Errorf("invalid EIS XLSX workbook: %w", err)
			}
			for col := range cells {
				cells[col] = cleanEISCell(cells[col])
			}
			rows = append(rows, cells)
		}
		if err := iterator.Error(); err != nil {
			return nil, fmt.Errorf("invalid EIS XLSX workbook: %w", err)
		}
		return rows, nil
	default:
		return nil, fmt.Errorf("unsupported EIS workbook format")
	}
}

// cleanEISCell drops NUL bytes, which EIS exports can leave inside text cells
// and PostgreSQL rejects in text columns, and turns other control characters
// into spaces.
func cleanEISCell(value string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == 0:
			return -1
		case unicode.IsControl(r):
			return ' '
		}
		return r
	}, value)
}
