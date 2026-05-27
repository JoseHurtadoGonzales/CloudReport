package recipe

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/xuri/excelize/v2"
)

// Xlsx recipe accepts the engine output as JSON describing a workbook, or as
// a {"sheets":[...rows]} shape, and produces an .xlsx.
//
// Two modes:
//
//   1. The engine produced JSON. We marshal it into the workbook structure.
//   2. The engine produced HTML with <table> elements (fallback). We don't
//      parse HTML in-process; for that use the html-to-xlsx recipe (worker).
type Xlsx struct{}

type xlsxWorkbook struct {
	Sheets []xlsxSheet `json:"sheets"`
}

type xlsxSheet struct {
	Name string          `json:"name"`
	Rows [][]interface{} `json:"rows"`
}

func (Xlsx) Execute(c *Context) (*Result, error) {
	var wb xlsxWorkbook
	if err := json.Unmarshal([]byte(c.HTML), &wb); err != nil || len(wb.Sheets) == 0 {
		// fallback single-sheet with one cell of HTML
		wb = xlsxWorkbook{Sheets: []xlsxSheet{{Name: "Sheet1", Rows: [][]interface{}{{c.HTML}}}}}
	}
	f := excelize.NewFile()
	defer f.Close()

	// Replace default sheet with first sheet from workbook.
	first := wb.Sheets[0]
	if first.Name == "" {
		first.Name = "Sheet1"
	}
	f.SetSheetName("Sheet1", first.Name)
	if err := writeRows(f, first.Name, first.Rows); err != nil {
		return nil, err
	}
	for _, s := range wb.Sheets[1:] {
		if s.Name == "" {
			s.Name = "Sheet"
		}
		if _, err := f.NewSheet(s.Name); err != nil {
			return nil, err
		}
		if err := writeRows(f, s.Name, s.Rows); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return &Result{
		Content:  buf.Bytes(),
		MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		FileName: "report.xlsx",
	}, nil
}

func writeRows(f *excelize.File, sheet string, rows [][]interface{}) error {
	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return fmt.Errorf("coords: %w", err)
		}
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			return err
		}
	}
	return nil
}
