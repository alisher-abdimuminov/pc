package export

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"

	"pc/internal/attendance"
	"pc/internal/service"
)

func sample() service.Report {
	g := "TEST-101"
	return service.Report{
		From: "2026-10-05", To: "2026-10-11",
		Dates: []string{"2026-10-05", "2026-10-06", "2026-10-07", "2026-10-11"}, // Mon, Tue, Wed, Sun
		Students: []service.ReportStudent{
			{FullName: "Alpha Bir", GroupName: &g, Cells: map[string]service.ReportCell{
				"2026-10-05": {Status: "full", Shift: 1, Steps: 3},
				"2026-10-06": {Status: "partial", Shift: 2, Steps: 1},
				"2026-10-07": {Status: "absent"},
			}, Totals: attendance.Totals{Expected: 3, Full: 1, Partial: 1, Absent: 1, Percent: 33}},
			{FullName: "Beta Ikki", Cells: map[string]service.ReportCell{}, Totals: attendance.Totals{}},
		},
	}
}

func open(t *testing.T, withGroup bool) (*excelize.File, [][]string) {
	t.Helper()
	b, err := Attendance(sample(), withGroup)
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	rows, _ := f.GetRows("Davomat")
	return f, rows
}

func TestAttendanceHasATitleBlockAndAHeaderWithWeekdays(t *testing.T) {
	_, rows := open(t, true)
	if rows[0][0] != "Davomat hisoboti" || rows[1][0] != "Davr: 05.10.2026 – 11.10.2026" {
		t.Fatalf("title block: %q %q", rows[0], rows[1])
	}
	if len(rows[2]) == 0 || rows[2][0][:9] != "Tuzilgan:" {
		t.Fatalf("generated line: %q", rows[2])
	}
	head := rows[3]
	want := []string{"T/r", "F.I.Sh.", "Guruh", "05.10\nDush", "06.10\nSesh", "07.10\nChor", "11.10\nYak", "Kutilgan kunlar", "To'liq", "Qisman", "Kelmagan", "Foiz"}
	for i, w := range want {
		if head[i] != w {
			t.Errorf("header[%d] = %q, want %q", i, head[i], w)
		}
	}
}

func TestAttendanceCellsAndTotals(t *testing.T) {
	f, rows := open(t, true)
	r := rows[4]
	if r[0] != "1" || r[1] != "Alpha Bir" || r[2] != "TEST-101" {
		t.Fatalf("person columns: %q", r)
	}
	if r[3] != "To'liq\n1-smena" || r[4] != "Qisman (1/3)\n2-smena" || r[5] != "Kelmagan" || r[6] != "" {
		t.Fatalf("day cells: %q", r[3:7])
	}
	if v, _ := f.GetCellValue("Davomat", "L5", excelize.Options{RawCellValue: true}); v != "0.33" { // percent is a real number
		t.Fatalf("percent raw value %q", v)
	}
	if s, _ := f.GetCellValue("Davomat", "L5"); s != "33%" {
		t.Fatalf("percent shown as %q", s)
	}
	if s, _ := f.GetCellValue("Davomat", "L6"); s != "–" { // nothing expected: no misleading 0%
		t.Fatalf("empty percent %q", s)
	}
	if avg, _ := f.GetCellValue("Davomat", "A7"); avg != "O'rtacha foiz" {
		t.Fatalf("average row %q", avg)
	}
}

func TestAttendanceStatusColoursAndGrayOffDays(t *testing.T) {
	f, _ := open(t, true)
	fill := func(c string) string {
		id, _ := f.GetCellStyle("Davomat", c)
		st, _ := f.GetStyle(id)
		if st == nil || len(st.Fill.Color) == 0 {
			return ""
		}
		return st.Fill.Color[0]
	}
	if fill("D5") != colGreenBG || fill("E5") != colAmberBG || fill("F5") != colRedBG {
		t.Fatalf("status fills: %s %s %s", fill("D5"), fill("E5"), fill("F5"))
	}
	if fill("G5") != colOffBG { // Sunday, nothing expected
		t.Fatalf("Sunday fill %q", fill("G5"))
	}
	if fill("L5") != colRedBG { // 33% is below 50
		t.Fatalf("percent fill %q", fill("L5"))
	}
}

func TestAttendanceFreezesHeaderAndNames(t *testing.T) {
	f, _ := open(t, true)
	p, err := f.GetPanes("Davomat")
	if err != nil || !p.Freeze || p.XSplit != 3 || p.YSplit != 4 {
		t.Fatalf("panes %+v %v", p, err)
	}
	f2, _ := open(t, false) // without the group column only two columns are frozen
	if p, _ := f2.GetPanes("Davomat"); p.XSplit != 2 {
		t.Fatalf("panes without group %+v", p)
	}
}

func TestAttendanceWithoutStudentsStillBuilds(t *testing.T) {
	rep := sample()
	rep.Students = nil
	b, err := Attendance(rep, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := excelize.OpenReader(bytes.NewReader(b)); err != nil {
		t.Fatal(err)
	}
}
