package httpapi

import (
	"net/http/httptest"
	"testing"
)

func TestCSVMoney(t *testing.T) {
	for cents, want := range map[int64]string{0: "0.00", 5: "0.05", 123456: "1234.56", -2500: "-25.00"} {
		if got := csvMoney(cents); got != want {
			t.Errorf("csvMoney(%d) = %s, want %s", cents, got, want)
		}
	}
}

func TestWriteCSVEscapesFormulas(t *testing.T) {
	rec := httptest.NewRecorder()
	err := writeCSV(rec, "x.csv", []string{"a", "b", "c", "d"}, [][]string{{"=HYPERLINK(\"x\")", "-12.50", "Smith, J", "-1+1"}})
	if err != nil {
		t.Fatal(err)
	}
	want := "a,b,c,d\n\"'=HYPERLINK(\"\"x\"\")\",-12.50,\"Smith, J\",'-1+1\n"
	if rec.Body.String() != want {
		t.Errorf("got %q, want %q", rec.Body.String(), want)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="x.csv"` {
		t.Errorf("content disposition %q", cd)
	}
}
