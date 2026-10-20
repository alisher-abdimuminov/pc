package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/attendance"
	"pc/internal/audit"
	"pc/internal/db"
	"pc/internal/storage"
)

type StudentService struct {
	pool        *pgxpool.Pool
	store       storage.Storage
	loc         *time.Location
	reports     *ReportService
	maxAttempts int
	clock       func() time.Time
	gpsMax      float64
}

// WithGPSLimit sets the maximum accepted GPS accuracy (meters) reported to the client.
func (s *StudentService) WithGPSLimit(m float64) *StudentService {
	s.gpsMax = m
	return s
}

// WithClock replaces the time source (tests).
func (s *StudentService) WithClock(c func() time.Time) *StudentService {
	s.clock = c
	return s
}

func NewStudentService(pool *pgxpool.Pool, store storage.Storage, loc *time.Location, reports *ReportService, maxAttemptsPerStep int) *StudentService {
	return &StudentService{pool: pool, store: store, loc: loc, reports: reports, maxAttempts: maxAttemptsPerStep, clock: time.Now}
}

// ---- 4.1 today ----

type TodaySchedule struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	LocationName string    `json:"location_name"`
	LocationURL  string    `json:"location_url"`
}

type TodayStep struct {
	Step        int    `json:"step"`
	Start       string `json:"start"`
	End         string `json:"end"`
	State       string `json:"state"` // done | pending | missed
	Current     bool   `json:"current"`
	Attempts    int    `json:"attempts"`
	MaxAttempts int    `json:"max_attempts"`
}

type TodayView struct {
	Date       string         `json:"date"`
	ServerTime time.Time      `json:"server_time"`
	State      string         `json:"state"` // no_schedule | before_start | active | finished
	Schedule   *TodaySchedule `json:"schedule"`
	Shift      int            `json:"shift"`
	// ShiftLocked is set when the day's attendance already belongs to a different shift than the
	// current time slot: the student cannot confirm in the other shift today.
	ShiftLocked bool        `json:"shift_locked"`
	Steps       []TodayStep `json:"steps"`
	// GPSMaxAccuracyM is the worst GPS accuracy the server accepts for a location check.
	GPSMaxAccuracyM float64 `json:"gps_max_accuracy_m"`
}

func (s *StudentService) dateOf(now time.Time) time.Time {
	n := now.In(s.loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// wall returns now as a wall-clock value in the business timezone.
func (s *StudentService) wall(now time.Time) time.Time { return now.In(s.loc) }

func (s *StudentService) Today(ctx context.Context, u db.User) (TodayView, error) {
	return s.TodayAt(ctx, u, s.clock())
}

func (s *StudentService) TodayAt(ctx context.Context, u db.User, now time.Time) (TodayView, error) {
	q := db.New(s.pool)
	date := s.dateOf(now)
	wall := s.wall(now)
	v := TodayView{Date: date.Format(DateLayout), ServerTime: now, State: "no_schedule", Steps: []TodayStep{}, GPSMaxAccuracyM: s.gpsMax}

	scheds, err := q.ListSchedulesForStudentOnDate(ctx, db.ListSchedulesForStudentOnDateParams{
		OnDate: pgtype.Date{Time: date, Valid: true}, StudentID: u.ID,
	})
	if err != nil {
		return v, err
	}
	if len(scheds) == 0 {
		return v, nil
	}
	// A personal schedule that covers the date decides, otherwise the group's (already ordered).
	sc := scheds[0]
	days := attendance.Days{sc.Monday, sc.Tuesday, sc.Wednesday, sc.Thursday, sc.Friday, sc.Saturday}
	if !days.Has(date.Weekday()) {
		return v, nil
	}
	ts := &TodaySchedule{ID: sc.ID, Name: sc.Name}
	if sc.LocationID.Valid {
		if l, err := q.GetLocation(ctx, sc.LocationID.UUID); err == nil {
			ts.LocationName, ts.LocationURL = l.Name, l.LocationUrl
		}
	}
	v.Schedule = ts

	m := wall.Hour()*60 + wall.Minute()
	switch {
	case m < attendance.DayStart:
		v.State = "before_start"
	case m >= attendance.DayEnd:
		v.State = "finished"
	default:
		v.State = "active"
	}

	// The day's shift: the attendance's if one exists, otherwise the one the current time falls in.
	att, attErr := q.GetTodayAttendance(ctx, db.GetTodayAttendanceParams{StudentID: u.ID, OnDate: pgtype.Date{Time: date, Valid: true}})
	hasAtt := attErr == nil
	if attErr != nil && !errors.Is(attErr, pgx.ErrNoRows) {
		return v, attErr
	}
	cur, inWindow := attendance.WindowAt(wall)
	switch {
	case hasAtt:
		v.Shift = int(att.Shift)
		v.ShiftLocked = inWindow && cur.Shift != v.Shift
	case inWindow:
		v.Shift = cur.Shift
	case m < attendance.DayStart:
		v.Shift = 1
	default:
		return v, nil // the day is over and nothing was confirmed: nothing left to show
	}

	done := map[int]db.ListStepsWithAttemptCountsRow{}
	if hasAtt {
		rows, err := q.ListStepsWithAttemptCounts(ctx, att.ID)
		if err != nil {
			return v, err
		}
		for _, r := range rows {
			done[int(r.Step)] = r
		}
	}
	for _, w := range attendance.ShiftWindows(v.Shift) {
		r := done[w.Step]
		if r.Review == ReviewRejected {
			r.Attempts = int32(s.maxAttempts) // a rejected step is final: no more tries
		}
		v.Steps = append(v.Steps, TodayStep{
			Step: w.Step, Start: attendance.Clock(w.Start), End: attendance.Clock(w.End),
			State:    string(attendance.StateOf(w, r.Success, wall)),
			Current:  inWindow && cur.Shift == w.Shift && cur.Step == w.Step && !v.ShiftLocked,
			Attempts: int(r.Attempts), MaxAttempts: s.maxAttempts,
		})
	}
	return v, nil
}

// ---- 4.2 monthly calendar ----

type MonthView struct {
	Month  string                `json:"month"`
	From   string                `json:"from"`
	To     string                `json:"to"`
	Dates  []string              `json:"dates"`
	Cells  map[string]ReportCell `json:"cells"`
	Totals attendance.Totals     `json:"totals"`
}

func (s *StudentService) Month(ctx context.Context, u db.User, month string) (MonthView, error) {
	if month == "" {
		month = s.dateOf(time.Now()).Format("2006-01")
	}
	first, err := time.Parse("2006-01", month)
	if err != nil {
		return MonthView{}, fieldErr("month", "Oy noto'g'ri (YYYY-MM)")
	}
	last := first.AddDate(0, 1, -1)
	rep, err := s.reports.Build(ctx, nil, &u.ID, first, last, s.reports.Today())
	if err != nil {
		return MonthView{}, err
	}
	mv := MonthView{Month: month, From: rep.From, To: rep.To, Dates: rep.Dates, Cells: map[string]ReportCell{}}
	if len(rep.Students) == 1 {
		mv.Cells, mv.Totals = rep.Students[0].Cells, rep.Students[0].Totals
	}
	return mv, nil
}

// ---- 4.4 profile ----

type ProfileDTO struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	FullName    string    `json:"full_name"`
	ShortName   string    `json:"short_name"`
	FirstName   string    `json:"first_name"`
	SecondName  string    `json:"second_name"`
	ThirdName   string    `json:"third_name"`
	BirthDate   string    `json:"birth_date"`
	Gender      string    `json:"gender"`
	Phone       string    `json:"phone"`
	Level       string    `json:"level"`
	Semester    string    `json:"semester"`
	Address     string    `json:"address"`
	Country     string    `json:"country"`
	Province    string    `json:"province"`
	District    string    `json:"district"`
	GroupName   *string   `json:"group_name"`
	FacultyName *string   `json:"faculty_name"`
	PhotoURL    string    `json:"photo_url"`
}

// Profile never includes passport data.
func (s *StudentService) Profile(ctx context.Context, u db.User) (ProfileDTO, error) {
	q := db.New(s.pool)
	p := ProfileDTO{ID: u.ID, Username: u.Username.String, FullName: u.FullName, ShortName: u.ShortName,
		FirstName: u.FirstName, SecondName: u.SecondName, ThirdName: u.ThirdName, BirthDate: u.BirthDate,
		Gender: u.Gender, Phone: u.Phone, Level: u.Level, Semester: u.Semester, Address: u.Address,
		Country: u.Country, Province: u.Province, District: u.District}
	if u.GroupID.Valid {
		if g, err := q.GetGroup(ctx, u.GroupID.UUID); err == nil {
			p.GroupName = &g.Name
		}
	}
	if u.FacultyID.Valid {
		if f, err := q.GetFaculty(ctx, u.FacultyID.UUID); err == nil {
			p.FacultyName = &f.Name
		}
	}
	if u.Image != "" {
		p.PhotoURL = "/api/files/avatars/" + u.ID.String()
	}
	return p, nil
}

// ---- 4.3 assignments ----

type StudentSubmission struct {
	ID          uuid.UUID  `json:"id"`
	SubmittedAt time.Time  `json:"submitted_at"`
	HasFile     bool       `json:"has_file"`
	Grade       *float64   `json:"grade"`
	Feedback    string     `json:"feedback"`
	GradedAt    *time.Time `json:"graded_at"`
}

type StudentAssignment struct {
	ID          uuid.UUID          `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	HasFile     bool               `json:"has_file"`
	Deadline    time.Time          `json:"deadline"`
	Overdue     bool               `json:"overdue"`
	CanSubmit   bool               `json:"can_submit"`
	Submission  *StudentSubmission `json:"submission"`
}

type StudentAssignments struct {
	ServerTime time.Time           `json:"server_time"`
	Items      []StudentAssignment `json:"items"`
}

type assignmentRow struct {
	ID                       uuid.UUID
	Title, Description, File string
	Deadline                 time.Time
	SubID                    uuid.NullUUID
	SubFile                  pgtype.Text
	SubmittedAt              pgtype.Timestamptz
	Grade                    pgtype.Numeric
	Feedback                 pgtype.Text
	GradedAt                 pgtype.Timestamptz
}

func buildStudentAssignment(r assignmentRow, now time.Time) StudentAssignment {
	a := StudentAssignment{ID: r.ID, Title: r.Title, Description: r.Description, HasFile: r.File != "",
		Deadline: r.Deadline, Overdue: !now.Before(r.Deadline)}
	if r.SubID.Valid {
		a.Submission = &StudentSubmission{ID: r.SubID.UUID, SubmittedAt: r.SubmittedAt.Time, HasFile: r.SubFile.String != "",
			Grade: numericFloat(r.Grade), Feedback: r.Feedback.String, GradedAt: tsPtr(r.GradedAt)}
	}
	graded := a.Submission != nil && a.Submission.GradedAt != nil
	a.CanSubmit = !a.Overdue && !graded
	return a
}

func (s *StudentService) Assignments(ctx context.Context, u db.User) (StudentAssignments, error) {
	out := StudentAssignments{ServerTime: time.Now(), Items: []StudentAssignment{}}
	if !u.GroupID.Valid {
		return out, nil
	}
	rows, err := db.New(s.pool).ListStudentAssignments(ctx, db.ListStudentAssignmentsParams{StudentID: u.ID, GroupID: u.GroupID.UUID})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.Items = append(out.Items, buildStudentAssignment(assignmentRow{
			ID: r.ID, Title: r.Title, Description: r.Description, File: r.File, Deadline: r.Deadline.Time,
			SubID: r.SubmissionID, SubFile: r.SubmissionFile, SubmittedAt: r.SubmittedAt, Grade: r.Grade,
			Feedback: r.Feedback, GradedAt: r.GradedAt,
		}, out.ServerTime))
	}
	return out, nil
}

// assignment loads one assignment of the student's own group; anything else is a 404.
func (s *StudentService) assignment(ctx context.Context, q *db.Queries, u db.User, id uuid.UUID, now time.Time) (StudentAssignment, error) {
	if !u.GroupID.Valid {
		return StudentAssignment{}, apperr.NotFound()
	}
	r, err := q.GetStudentAssignment(ctx, db.GetStudentAssignmentParams{AssignmentID: id, GroupID: u.GroupID.UUID, StudentID: u.ID})
	if err != nil {
		return StudentAssignment{}, notFoundOr(err)
	}
	return buildStudentAssignment(assignmentRow{
		ID: r.ID, Title: r.Title, Description: r.Description, File: r.File, Deadline: r.Deadline.Time,
		SubID: r.SubmissionID, SubFile: r.SubmissionFile, SubmittedAt: r.SubmittedAt, Grade: r.Grade,
		Feedback: r.Feedback, GradedAt: r.GradedAt,
	}, now), nil
}

func (s *StudentService) Assignment(ctx context.Context, u db.User, id uuid.UUID) (StudentAssignment, error) {
	return s.assignment(ctx, db.New(s.pool), u, id, time.Now())
}

var (
	errDeadlinePassed = apperr.New(http.StatusConflict, "deadline_passed", "Topshirish muddati tugagan")
	errAlreadyGraded  = apperr.New(http.StatusConflict, "already_graded", "Ish baholangan, qayta yuklab bo'lmaydi")
)

// Submit creates or replaces the student's submission. Server time decides the deadline.
func (s *StudentService) Submit(ctx context.Context, actor Actor, u db.User, assignmentID uuid.UUID, up *Upload) (StudentAssignment, error) {
	return s.submitAt(ctx, actor, u, assignmentID, up, time.Now)
}

func (s *StudentService) submitAt(ctx context.Context, actor Actor, u db.User, assignmentID uuid.UUID, up *Upload, clock func() time.Time) (StudentAssignment, error) {
	var out StudentAssignment
	if up == nil {
		return out, fieldErr("file", "Faylni tanlang")
	}
	// Cheap pre-check before touching the disk; the transaction below is authoritative.
	pre, err := s.assignment(ctx, db.New(s.pool), u, assignmentID, clock())
	if err != nil {
		return out, err
	}
	if pre.Overdue {
		return out, errDeadlinePassed
	}
	if pre.Submission != nil && pre.Submission.GradedAt != nil {
		return out, errAlreadyGraded
	}
	// reuse the teacher-side upload validation (content sniffing, size limit)
	path, err := (&AssignmentService{store: s.store}).saveUpload(ctx, storage.KindSubmissions, up)
	if err != nil {
		return out, err
	}

	var oldPath string
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		now := clock()
		cur, err := s.assignment(ctx, q, u, assignmentID, now)
		if err != nil {
			return err
		}
		if cur.Overdue {
			return errDeadlinePassed
		}
		existing, err := q.GetSubmissionForUpdate(ctx, db.GetSubmissionForUpdateParams{AssignmentID: assignmentID, StudentID: u.ID})
		replaced := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if replaced && existing.GradedAt.Valid {
			return errAlreadyGraded
		}
		var sub db.Submission
		if replaced {
			oldPath = existing.File
			sub, err = q.ReplaceSubmissionFile(ctx, db.ReplaceSubmissionFileParams{ID: existing.ID, File: path})
		} else {
			sub, err = q.CreateSubmission(ctx, db.CreateSubmissionParams{AssignmentID: assignmentID, StudentID: u.ID, File: path})
		}
		if err != nil {
			return err
		}
		action := audit.ActionCreate
		if replaced {
			action = audit.ActionUpdate
		}
		if err := audit.Write(ctx, tx, actor.entry(action, "submission", &sub.ID, nil,
			map[string]any{"assignment_id": assignmentID, "replaced": replaced, "submitted_at": sub.SubmittedAt.Time})); err != nil {
			return err
		}
		out, err = s.assignment(ctx, q, u, assignmentID, now)
		return err
	})
	if err != nil {
		_ = s.store.Delete(ctx, path)
		return StudentAssignment{}, mapDBError(err)
	}
	if oldPath != "" && oldPath != path {
		_ = s.store.Delete(ctx, oldPath)
	}
	return out, nil
}
