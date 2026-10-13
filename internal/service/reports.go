package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/attendance"
	"pc/internal/audit"
	"pc/internal/db"
)

const (
	maxReportDays    = 93
	maxReportDaysAll = 31
)

type ReportService struct {
	pool *pgxpool.Pool
	loc  *time.Location
}

func NewReportService(pool *pgxpool.Pool, loc *time.Location) *ReportService {
	return &ReportService{pool: pool, loc: loc}
}

// Today returns the current business date as a UTC-midnight time (calendar date semantics).
func (s *ReportService) Today() time.Time {
	n := time.Now().In(s.loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

type ReportCell struct {
	Status string `json:"status"`
	Shift  int    `json:"shift,omitempty"`
	Steps  int    `json:"steps"`
}

type ReportStudent struct {
	ID        uuid.UUID             `json:"id"`
	FullName  string                `json:"full_name"`
	GroupID   *uuid.UUID            `json:"group_id"`
	GroupName *string               `json:"group_name"`
	Cells     map[string]ReportCell `json:"cells"` // keyed by YYYY-MM-DD; missing = not a working day
	Totals    attendance.Totals     `json:"totals"`
}

type Report struct {
	From     string          `json:"from"`
	To       string          `json:"to"`
	Dates    []string        `json:"dates"`
	Students []ReportStudent `json:"students"`
}

// ParseRange parses the from/to query values (YYYY-MM-DD). Empty values default to today.
func (s *ReportService) ParseRange(fromStr, toStr string, maxDays int) (time.Time, time.Time, error) {
	fields := map[string]string{}
	parse := func(name, v string, def time.Time) time.Time {
		if v == "" {
			return def
		}
		t, err := time.Parse(DateLayout, v)
		if err != nil {
			fields[name] = "Sana noto'g'ri (YYYY-MM-DD)"
		}
		return t
	}
	today := s.Today()
	from := parse("from", fromStr, today)
	to := parse("to", toStr, from)
	if toStr == "" && fromStr == "" {
		to = today
	}
	if len(fields) == 0 {
		if to.Before(from) {
			fields["to"] = "Tugash sanasi boshlanishdan oldin bo'lishi mumkin emas"
		} else if int(to.Sub(from).Hours()/24)+1 > maxDays {
			fields["to"] = "Davr juda uzun"
		}
	}
	if len(fields) > 0 {
		return from, to, apperr.Validation("Davr noto'g'ri", fields)
	}
	return from, to, nil
}

// Build computes the attendance matrix for active students (of one group, or all) over [from, to].
func (s *ReportService) Build(ctx context.Context, groupID, studentID *uuid.UUID, from, to, today time.Time) (Report, error) {
	q := db.New(s.pool)
	students, err := q.ListReportStudents(ctx, db.ListReportStudentsParams{GroupID: nullUUID(groupID), StudentID: nullUUID(studentID)})
	if err != nil {
		return Report{}, err
	}
	fromD := pgtype.Date{Time: from, Valid: true}
	toD := pgtype.Date{Time: to, Valid: true}
	scheds, err := q.ListSchedulesInRange(ctx, db.ListSchedulesInRangeParams{FromDate: fromD, ToDate: toD})
	if err != nil {
		return Report{}, err
	}
	summary, err := q.ListAttendanceSummary(ctx, db.ListAttendanceSummaryParams{FromDate: fromD, ToDate: toD, GroupID: nullUUID(groupID), StudentID: nullUUID(studentID)})
	if err != nil {
		return Report{}, err
	}

	byUser := map[uuid.UUID][]attendance.Schedule{}
	byGroup := map[uuid.UUID][]attendance.Schedule{}
	for _, sc := range scheds {
		a := attendance.Schedule{Start: sc.StartDate.Time, End: sc.EndDate.Time,
			Days: attendance.Days{sc.Monday, sc.Tuesday, sc.Wednesday, sc.Thursday, sc.Friday, sc.Saturday}}
		if sc.UserID.Valid {
			byUser[sc.UserID.UUID] = append(byUser[sc.UserID.UUID], a)
		}
		if sc.GroupID.Valid {
			byGroup[sc.GroupID.UUID] = append(byGroup[sc.GroupID.UUID], a)
		}
	}
	type key struct {
		student uuid.UUID
		date    string
	}
	best := map[key]db.ListAttendanceSummaryRow{}
	for _, r := range summary {
		k := key{r.StudentID, r.Date.Time.Format(DateLayout)}
		if cur, ok := best[k]; !ok || r.StepsDone > cur.StepsDone {
			best[k] = r
		}
	}

	dates := attendance.DateRange(from, to)
	rep := Report{From: from.Format(DateLayout), To: to.Format(DateLayout), Dates: make([]string, len(dates)), Students: make([]ReportStudent, 0, len(students))}
	for i, d := range dates {
		rep.Dates[i] = d.Format(DateLayout)
	}
	for _, st := range students {
		rs := ReportStudent{ID: st.ID, FullName: st.FullName, GroupID: uuidPtr(st.GroupID), GroupName: textPtr(st.GroupName), Cells: map[string]ReportCell{}}
		var group []attendance.Schedule
		if st.GroupID.Valid {
			group = byGroup[st.GroupID.UUID]
		}
		for i, d := range dates {
			if d.After(today) {
				continue
			}
			ds := rep.Dates[i]
			status := attendance.StatusNone
			cell := ReportCell{}
			if rec, ok := best[key{st.ID, ds}]; ok {
				cell.Shift, cell.Steps = int(rec.Shift), int(rec.StepsDone)
				if rec.StepsDone > 0 || attendance.Expected(d, byUser[st.ID], group) {
					status = attendance.StatusFromSteps(int(rec.StepsDone))
				}
			} else if attendance.Expected(d, byUser[st.ID], group) {
				status = attendance.StatusAbsent
			}
			if status == attendance.StatusNone {
				continue
			}
			cell.Status = string(status)
			rs.Cells[ds] = cell
			rs.Totals.Add(status)
		}
		rs.Totals.Finish()
		rep.Students = append(rep.Students, rs)
	}
	return rep, nil
}

// ownedGroup returns the group, or 404 when teacherID is set and does not own it.
func (s *ReportService) ownedGroup(ctx context.Context, teacherID *uuid.UUID, groupID uuid.UUID) (db.Group, error) {
	q := db.New(s.pool)
	if teacherID == nil {
		g, err := q.GetGroup(ctx, groupID)
		return g, notFoundOr(err)
	}
	g, err := q.GetGroupForTeacher(ctx, db.GetGroupForTeacherParams{ID: groupID, TeacherID: uuid.NullUUID{UUID: *teacherID, Valid: true}})
	return g, notFoundOr(err)
}

// GroupReport is the report for one group; teacherID restricts access (nil = admin).
func (s *ReportService) GroupReport(ctx context.Context, teacherID *uuid.UUID, groupID uuid.UUID, fromStr, toStr string) (Report, error) {
	if _, err := s.ownedGroup(ctx, teacherID, groupID); err != nil {
		return Report{}, err
	}
	from, to, err := s.ParseRange(fromStr, toStr, maxReportDays)
	if err != nil {
		return Report{}, err
	}
	return s.Build(ctx, &groupID, nil, from, to, s.Today())
}

// AdminReport covers one group, or every active student when groupID is nil.
func (s *ReportService) AdminReport(ctx context.Context, groupID *uuid.UUID, fromStr, toStr string) (Report, error) {
	if groupID != nil {
		return s.GroupReport(ctx, nil, *groupID, fromStr, toStr)
	}
	from, to, err := s.ParseRange(fromStr, toStr, maxReportDaysAll)
	if err != nil {
		return Report{}, err
	}
	return s.Build(ctx, nil, nil, from, to, s.Today())
}

// ---- teacher group overview ----

type TodayStats struct {
	Expected int `json:"expected"`
	Full     int `json:"full"`
	Partial  int `json:"partial"`
	Absent   int `json:"absent"`
	// AttendedPercent is the share of expected students who confirmed at least one step today.
	AttendedPercent int `json:"attended_percent"`
}

type TeacherGroupDTO struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	FacultyName  *string    `json:"faculty_name"`
	StudentCount int64      `json:"student_count"`
	Today        TodayStats `json:"today"`
}

func todayStats(rep Report, today string) TodayStats {
	var t TodayStats
	for _, st := range rep.Students {
		switch st.Cells[today].Status {
		case "full":
			t.Full++
		case "partial":
			t.Partial++
		case "absent":
			t.Absent++
		default:
			continue
		}
		t.Expected++
	}
	if t.Expected > 0 {
		t.AttendedPercent = ((t.Full+t.Partial)*100 + t.Expected/2) / t.Expected
	}
	return t
}

func (s *ReportService) TeacherGroups(ctx context.Context, teacherID uuid.UUID) ([]TeacherGroupDTO, error) {
	q := db.New(s.pool)
	tid := uuid.NullUUID{UUID: teacherID, Valid: true}
	rows, err := q.ListGroupsAdmin(ctx, db.ListGroupsAdminParams{TeacherID: tid, Lim: 500})
	if err != nil {
		return nil, err
	}
	today := s.Today()
	out := []TeacherGroupDTO{}
	for _, g := range rows {
		gid := g.ID
		rep, err := s.Build(ctx, &gid, nil, today, today, today)
		if err != nil {
			return nil, err
		}
		out = append(out, TeacherGroupDTO{ID: g.ID, Name: g.Name, FacultyName: textPtr(g.FacultyName),
			StudentCount: g.StudentCount, Today: todayStats(rep, today.Format(DateLayout))})
	}
	return out, nil
}

type GroupStudentToday struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Status   string    `json:"status"` // full | partial | absent | "" (no schedule today)
	Shift    int       `json:"shift"`
	Steps    int       `json:"steps"`
}

type TeacherGroupDetail struct {
	TeacherGroupDTO
	Students []GroupStudentToday `json:"students"`
}

func (s *ReportService) TeacherGroup(ctx context.Context, teacherID uuid.UUID, groupID uuid.UUID) (TeacherGroupDetail, error) {
	g, err := s.ownedGroup(ctx, &teacherID, groupID)
	if err != nil {
		return TeacherGroupDetail{}, err
	}
	today := s.Today()
	rep, err := s.Build(ctx, &groupID, nil, today, today, today)
	if err != nil {
		return TeacherGroupDetail{}, err
	}
	ts := today.Format(DateLayout)
	d := TeacherGroupDetail{Students: []GroupStudentToday{}}
	d.ID, d.Name, d.StudentCount, d.Today = g.ID, g.Name, int64(len(rep.Students)), todayStats(rep, ts)
	if g.FacultyID.Valid {
		if f, err := db.New(s.pool).GetFaculty(ctx, g.FacultyID.UUID); err == nil {
			d.FacultyName = &f.Name
		}
	}
	for _, st := range rep.Students {
		c := st.Cells[ts]
		d.Students = append(d.Students, GroupStudentToday{ID: st.ID, FullName: st.FullName, Status: c.Status, Shift: c.Shift, Steps: c.Steps})
	}
	return d, nil
}

// ---- per-student attempts ----

type AttemptDTO struct {
	ID               uuid.UUID `json:"id"`
	AttemptedAt      time.Time `json:"attempted_at"`
	FaceVerified     bool      `json:"face_verified"`
	LocationVerified bool      `json:"location_verified"`
	LivenessVerified bool      `json:"liveness_verified"`
	Success          bool      `json:"success"`
	Error            string    `json:"error"`
	FaceDistance     *float64  `json:"face_distance"`
	FaceThreshold    *float64  `json:"face_threshold"`
	AntispoofScore   *float64  `json:"antispoof_score"`
	GPSAccuracy      float64   `json:"gps_accuracy"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	HasSelfie        bool      `json:"has_selfie"`
}

type StepDTO struct {
	Step     int          `json:"step"`
	Success  bool         `json:"success"`
	Attempts []AttemptDTO `json:"attempts"`
}

type AttemptDayDTO struct {
	Date  string    `json:"date"`
	Shift int       `json:"shift"`
	Steps []StepDTO `json:"steps"`
}

type StudentAttempts struct {
	Student struct {
		ID        uuid.UUID `json:"id"`
		FullName  string    `json:"full_name"`
		GroupName *string   `json:"group_name"`
	} `json:"student"`
	From string          `json:"from"`
	To   string          `json:"to"`
	Days []AttemptDayDTO `json:"days"`
}

// StudentAttempts lists steps and attempts by day (admin only).
func (s *ReportService) StudentAttempts(ctx context.Context, studentID uuid.UUID, fromStr, toStr string) (StudentAttempts, error) {
	q := db.New(s.pool)
	var out StudentAttempts
	stu, err := q.GetUser(ctx, studentID)
	if err != nil || stu.Role != "student" {
		return out, apperr.NotFound()
	}
	var groupName *string
	if stu.GroupID.Valid {
		if g, err := q.GetGroup(ctx, stu.GroupID.UUID); err == nil {
			groupName = &g.Name
		}
	}
	from, to, err := s.ParseRange(fromStr, toStr, maxReportDays)
	if err != nil {
		return out, err
	}
	rows, err := q.ListStudentAttempts(ctx, db.ListStudentAttemptsParams{
		StudentID: studentID, FromDate: pgtype.Date{Time: from, Valid: true}, ToDate: pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return out, err
	}
	out.Student.ID, out.Student.FullName, out.Student.GroupName = stu.ID, stu.FullName, groupName
	out.From, out.To = from.Format(DateLayout), to.Format(DateLayout)
	out.Days = []AttemptDayDTO{}
	for _, r := range rows {
		ds := r.Date.Time.Format(DateLayout)
		if n := len(out.Days); n == 0 || out.Days[n-1].Date != ds {
			out.Days = append(out.Days, AttemptDayDTO{Date: ds, Shift: int(r.Shift), Steps: []StepDTO{}})
		}
		day := &out.Days[len(out.Days)-1]
		if n := len(day.Steps); n == 0 || day.Steps[n-1].Step != int(r.Step) {
			day.Steps = append(day.Steps, StepDTO{Step: int(r.Step), Success: r.StepSuccess, Attempts: []AttemptDTO{}})
		}
		step := &day.Steps[len(day.Steps)-1]
		step.Attempts = append(step.Attempts, AttemptDTO{
			ID: r.AttemptID, AttemptedAt: r.AttemptedAt.Time, FaceVerified: r.FaceVerified, LocationVerified: r.LocationVerified,
			LivenessVerified: r.LivenessVerified, Success: r.Success, Error: r.Error, FaceDistance: floatPtr(r.FaceDistance),
			FaceThreshold: floatPtr(r.FaceThreshold), AntispoofScore: floatPtr(r.AntispoofScore), GPSAccuracy: r.GpsAccuracy,
			Latitude: r.Latitude, Longitude: r.Longitude, HasSelfie: r.Image != "",
		})
	}
	return out, nil
}

func floatPtr(f pgtype.Float8) *float64 {
	if !f.Valid {
		return nil
	}
	return &f.Float64
}

// AuditExport records that an attendance report was exported.
func (s *ReportService) AuditExport(ctx context.Context, actor Actor, groupID *uuid.UUID, rep Report) error {
	return audit.Write(ctx, s.pool, actor.entry(audit.ActionExport, "attendance_report", groupID, nil,
		map[string]any{"from": rep.From, "to": rep.To, "students": len(rep.Students), "format": "xlsx"}))
}
