package httpapi

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// writeCSV sends rows as a CSV download. Spreadsheet apps treat cells that
// start with = + - or @ as formulas, so those are prefixed with a quote.
func writeCSV(w http.ResponseWriter, filename string, header []string, rows [][]string) error {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("Cache-Control", "no-store")
	cw := csv.NewWriter(w)
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, row := range rows {
		for i, cell := range row {
			row[i] = safeCell(cell)
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func safeCell(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		// A plain negative number is fine and stays readable as a number.
		if _, err := strconv.ParseFloat(s, 64); err == nil && s[0] == '-' {
			return s
		}
		return "'" + s
	}
	return s
}

// csvMoney formats cents as a plain decimal, e.g. 123456 -> "1234.56".
func csvMoney(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}
