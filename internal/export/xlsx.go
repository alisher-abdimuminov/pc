// Package export builds Excel files with Uzbek headers.
package export

import (
	"bytes"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"

	"pc/internal/service"
)

const XLSXContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func cell(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func write(f *excelize.File) ([]byte, error) {
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// statusText is the cell text: the status and, on a second line, the shift.
func statusText(c service.ReportCell) string {
	var s string
	switch c.Status {
	case "full":
		s = "To'liq"
	case "partial":
		s = fmt.Sprintf("Qisman (%d/3)", c.Steps)
	case "absent":
		s = "Kelmagan"
	default:
		return ""
	}
	if c.Shift > 0 {
		s += fmt.Sprintf("\n%d-smena", c.Shift)
	}
	return s
}

var weekdayShort = [...]string{"Yak", "Dush", "Sesh", "Chor", "Pay", "Juma", "Shan"}

const (
	colGreenBG, colGreenFG = "DCFCE7", "166534"
	colAmberBG, colAmberFG = "FEF3C7", "92400E"
	colRedBG, colRedFG     = "FEE2E2", "991B1B"
	colOffBG               = "E5E7EB" // Sundays and days that have not come yet
	colHeadBG              = "1F2937"
)

type styles struct{ title, sub, head, headOff, name, plain, off, full, partial, absent, pctGreen, pctAmber, pctRed, pctNone, avg int }

func newStyles(f *excelize.File) styles {
	line := func(c string) excelize.Border { return excelize.Border{Type: "left", Color: c, Style: 1} }
	borders := func() []excelize.Border {
		return []excelize.Border{line("D1D5DB"), {Type: "right", Color: "D1D5DB", Style: 1}, {Type: "top", Color: "D1D5DB", Style: 1}, {Type: "bottom", Color: "D1D5DB", Style: 1}}
	}
	mk := func(st excelize.Style) int {
		id, _ := f.NewStyle(&st)
		return id
	}
	center := &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}
	fill := func(bg string) excelize.Fill { return excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{bg}} }
	status := func(bg, fg string) int {
		return mk(excelize.Style{Border: borders(), Fill: fill(bg), Font: &excelize.Font{Color: fg, Size: 10, Bold: true}, Alignment: center})
	}
	pct := func(bg, fg string) int {
		n := 9 // 0%
		return mk(excelize.Style{Border: borders(), Fill: fill(bg), NumFmt: n, Font: &excelize.Font{Color: fg, Bold: true}, Alignment: center})
	}
	return styles{
		title:    mk(excelize.Style{Font: &excelize.Font{Bold: true, Size: 16}, Alignment: &excelize.Alignment{Vertical: "center"}}),
		sub:      mk(excelize.Style{Font: &excelize.Font{Color: "6B7280", Size: 10}}),
		head:     mk(excelize.Style{Border: borders(), Fill: fill(colHeadBG), Font: &excelize.Font{Color: "FFFFFF", Bold: true}, Alignment: center}),
		headOff:  mk(excelize.Style{Border: borders(), Fill: fill("6B7280"), Font: &excelize.Font{Color: "FFFFFF", Bold: true}, Alignment: center}),
		name:     mk(excelize.Style{Border: borders(), Font: &excelize.Font{Bold: true}, Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}}),
		plain:    mk(excelize.Style{Border: borders(), Alignment: center}),
		off:      mk(excelize.Style{Border: borders(), Fill: fill(colOffBG), Alignment: center}),
		full:     status(colGreenBG, colGreenFG),
		partial:  status(colAmberBG, colAmberFG),
		absent:   status(colRedBG, colRedFG),
		pctGreen: pct(colGreenBG, colGreenFG),
		pctAmber: pct(colAmberBG, colAmberFG),
		pctRed:   pct(colRedBG, colRedFG),
		pctNone:  mk(excelize.Style{Border: borders(), Font: &excelize.Font{Color: "9CA3AF"}, Alignment: center}),
		avg:      mk(excelize.Style{Border: borders(), Font: &excelize.Font{Bold: true}, Fill: fill("F3F4F6"), Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}}),
	}
}

func bptr[T any](v T) *T { return &v }

// Attendance renders the attendance matrix: a title block, a frozen header with weekdays, coloured
// status cells, hatched-gray off days, totals with a coloured percentage, and a legend.
// withGroup adds a group column (admin "all" report).
func Attendance(rep service.Report, withGroup bool) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Davomat"
	f.SetSheetName("Sheet1", sheet)
	st := newStyles(f)

	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	today := now.Format(service.DateLayout)
	dmy := func(iso string) string {
		t, err := time.Parse(service.DateLayout, iso)
		if err != nil {
			return iso
		}
		return t.Format("02.01.2006")
	}

	const headRow = 4
	nameCols := 2 // T/r, F.I.Sh.
	if withGroup {
		nameCols = 3
	}
	nDays := len(rep.Dates)
	totalsAt := nameCols + nDays + 1 // first totals column
	lastCol := totalsAt + 4          // expected, full, partial, absent, percent

	// ---- title block ----
	f.SetCellValue(sheet, "A1", "Davomat hisoboti")
	f.SetCellStyle(sheet, "A1", "A1", st.title)
	f.SetRowHeight(sheet, 1, 28)
	f.SetCellValue(sheet, "A2", fmt.Sprintf("Davr: %s – %s", dmy(rep.From), dmy(rep.To)))
	f.SetCellValue(sheet, "A3", fmt.Sprintf("Tuzilgan: %s  ·  Talabalar soni: %d", now.Format("02.01.2006 15:04"), len(rep.Students)))
	f.SetCellStyle(sheet, "A2", "A3", st.sub)

	// ---- header ----
	heads := []string{"T/r", "F.I.Sh."}
	if withGroup {
		heads = append(heads, "Guruh")
	}
	for i, h := range heads {
		f.SetCellValue(sheet, cell(i+1, headRow), h)
		f.SetCellStyle(sheet, cell(i+1, headRow), cell(i+1, headRow), st.head)
	}
	off := func(d string) bool {
		t, err := time.Parse(service.DateLayout, d)
		return (err == nil && t.Weekday() == time.Sunday) || d > today
	}
	for i, d := range rep.Dates {
		t, _ := time.Parse(service.DateLayout, d)
		c := cell(nameCols+1+i, headRow)
		f.SetCellValue(sheet, c, t.Format("02.01")+"\n"+weekdayShort[t.Weekday()])
		if off(d) {
			f.SetCellStyle(sheet, c, c, st.headOff)
		} else {
			f.SetCellStyle(sheet, c, c, st.head)
		}
	}
	for i, h := range []string{"Kutilgan kunlar", "To'liq", "Qisman", "Kelmagan", "Foiz"} {
		c := cell(totalsAt+i, headRow)
		f.SetCellValue(sheet, c, h)
		f.SetCellStyle(sheet, c, c, st.head)
	}
	f.SetRowHeight(sheet, headRow, 34)

	// ---- students ----
	var pctSum float64
	var pctN int
	for i, s := range rep.Students {
		row := headRow + 1 + i
		f.SetRowHeight(sheet, row, 32)
		put := func(col int, v any, style int) {
			c := cell(col, row)
			f.SetCellValue(sheet, c, v)
			f.SetCellStyle(sheet, c, c, style)
		}
		put(1, i+1, st.plain)
		put(2, s.FullName, st.name)
		if withGroup {
			g := ""
			if s.GroupName != nil {
				g = *s.GroupName
			}
			put(3, g, st.plain)
		}
		for j, d := range rep.Dates {
			cc := s.Cells[d]
			style := st.plain
			switch {
			case cc.Status == "full":
				style = st.full
			case cc.Status == "partial":
				style = st.partial
			case cc.Status == "absent":
				style = st.absent
			case off(d):
				style = st.off
			}
			put(nameCols+1+j, statusText(cc), style)
		}
		put(totalsAt, s.Totals.Expected, st.plain)
		put(totalsAt+1, s.Totals.Full, st.plain)
		put(totalsAt+2, s.Totals.Partial, st.plain)
		put(totalsAt+3, s.Totals.Absent, st.plain)
		if s.Totals.Expected == 0 {
			put(totalsAt+4, "–", st.pctNone)
			continue
		}
		p := float64(s.Totals.Percent)
		pctSum += p
		pctN++
		style := st.pctRed
		switch {
		case p >= 80:
			style = st.pctGreen
		case p >= 50:
			style = st.pctAmber
		}
		put(totalsAt+4, p/100, style)
	}

	// ---- average + legend ----
	end := headRow + len(rep.Students)
	if pctN > 0 {
		r := end + 1
		f.MergeCell(sheet, cell(1, r), cell(totalsAt+3, r))
		f.SetCellValue(sheet, cell(1, r), "O'rtacha foiz")
		f.SetCellStyle(sheet, cell(1, r), cell(totalsAt+3, r), st.avg)
		f.SetCellValue(sheet, cell(totalsAt+4, r), pctSum/float64(pctN)/100)
		f.SetCellStyle(sheet, cell(totalsAt+4, r), cell(totalsAt+4, r), st.avg)
		avgPct, _ := f.NewStyle(&excelize.Style{NumFmt: 9, Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F3F4F6"}}, Alignment: &excelize.Alignment{Horizontal: "center"}})
		f.SetCellStyle(sheet, cell(totalsAt+4, r), cell(totalsAt+4, r), avgPct)
		end = r
	}
	legend := []struct {
		text  string
		style int
	}{
		{"To'liq — 3 ta bosqich tasdiqlangan", st.full},
		{"Qisman — 1–2 ta bosqich tasdiqlangan", st.partial},
		{"Kelmagan — ish kuni, davomat yo'q", st.absent},
		{"Kulrang — yakshanba yoki hali kelmagan kun", st.off},
	}
	for i, l := range legend {
		r := end + 2 + i
		c := cell(1, r)
		f.MergeCell(sheet, c, cell(max(nameCols, 2), r))
		f.SetCellValue(sheet, c, l.text)
		f.SetCellStyle(sheet, c, cell(max(nameCols, 2), r), l.style)
	}
	f.SetCellValue(sheet, cell(1, end+2+len(legend)), "Foiz — to'liq kelgan kunlarning kutilgan kunlarga nisbati")
	f.SetCellStyle(sheet, cell(1, end+2+len(legend)), cell(1, end+2+len(legend)), st.sub)

	// ---- layout ----
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 34)
	if withGroup {
		f.SetColWidth(sheet, "C", "C", 16)
	}
	dayFirst, _ := excelize.ColumnNumberToName(nameCols + 1)
	dayLast, _ := excelize.ColumnNumberToName(nameCols + max(nDays, 1))
	f.SetColWidth(sheet, dayFirst, dayLast, 12)
	tFirst, _ := excelize.ColumnNumberToName(totalsAt)
	tLast, _ := excelize.ColumnNumberToName(lastCol)
	f.SetColWidth(sheet, tFirst, tLast, 11)

	// the names and the header stay visible while scrolling a long period
	f.SetPanes(sheet, &excelize.Panes{
		Freeze: true, Split: false, XSplit: nameCols, YSplit: headRow,
		TopLeftCell: cell(nameCols+1, headRow+1), ActivePane: "bottomRight",
		Selection: []excelize.Selection{{SQRef: cell(nameCols+1, headRow+1), ActiveCell: cell(nameCols+1, headRow+1), Pane: "bottomRight"}},
	})
	if len(rep.Students) > 0 {
		f.AutoFilter(sheet, cell(1, headRow)+":"+cell(lastCol, headRow+len(rep.Students)), nil)
	}
	f.SetSheetView(sheet, 0, &excelize.ViewOptions{ShowGridLines: bptr(false)})
	f.SetPageLayout(sheet, &excelize.PageLayoutOptions{Orientation: bptr("landscape"), Size: bptr(9), FitToWidth: bptr(1), FitToHeight: bptr(0)})
	f.SetSheetProps(sheet, &excelize.SheetPropsOptions{FitToPage: bptr(true)})
	f.SetDefinedName(&excelize.DefinedName{Name: "_xlnm.Print_Titles", RefersTo: fmt.Sprintf("%s!$%d:$%d", sheet, headRow, headRow), Scope: sheet})
	return write(f)
}

// Grades renders the grades of one assignment.
func Grades(g service.GradesExport) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Baholar"
	f.SetSheetName("Sheet1", sheet)
	f.SetCellValue(sheet, "A1", "Topshiriq: "+g.Title)
	f.SetCellValue(sheet, "A2", "Guruh: "+g.Group)
	headers := []string{"T/r", "F.I.Sh.", "Holati", "Topshirilgan vaqt", "Baho", "Izoh"}
	for i, h := range headers {
		f.SetCellValue(sheet, cell(i+1, 4), h)
	}
	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	f.SetCellStyle(sheet, cell(1, 4), cell(len(headers), 4), bold)
	loc, err := time.LoadLocation("Asia/Tashkent")
	if err != nil {
		loc = time.UTC
	}
	for i, e := range g.Entries {
		row := i + 5
		status, when := "Topshirilmagan", ""
		if e.SubmissionID != nil {
			status = "Topshirilgan"
			if e.SubmittedAt != nil {
				when = e.SubmittedAt.In(loc).Format("02.01.2006 15:04")
			}
		}
		f.SetCellValue(sheet, cell(1, row), i+1)
		f.SetCellValue(sheet, cell(2, row), e.FullName)
		f.SetCellValue(sheet, cell(3, row), status)
		f.SetCellValue(sheet, cell(4, row), when)
		if e.Grade != nil {
			f.SetCellValue(sheet, cell(5, row), *e.Grade)
		}
		f.SetCellValue(sheet, cell(6, row), e.Feedback)
	}
	f.SetColWidth(sheet, "B", "B", 32)
	f.SetColWidth(sheet, "D", "D", 20)
	f.SetColWidth(sheet, "F", "F", 40)
	return write(f)
}
