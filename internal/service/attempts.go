package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/attendance"
	"pc/internal/audit"
	"pc/internal/db"
	"pc/internal/face"
	"pc/internal/geo"
	"pc/internal/storage"
)

// Failure codes reported to the client (the frontend maps them to guidance).
const (
	FailLocation         = "location"
	FailLocationAccuracy = "location_accuracy"
	FailNoFace           = "no_face"
	FailMultipleFaces    = "multiple_faces"
	FailLiveness         = "liveness"
	FailFace             = "face"
)

var failureText = map[string]string{
	FailLocation:         "Siz amaliyot joyida emassiz",
	FailLocationAccuracy: "GPS aniqligi yetarli emas",
	FailNoFace:           "Rasmda yuz topilmadi",
	FailMultipleFaces:    "Rasmda bir nechta yuz bor",
	FailLiveness:         "Jonli yuz tasdiqlanmadi (ekran yoki rasm aniqlandi)",
	FailFace:             "Yuz saqlangan namunaga mos kelmadi",
}

// Rejections: no attempt is recorded for these.
var (
	errOutsideHours = apperr.New(http.StatusConflict, "outside_hours", "Hozir tasdiqlash vaqti emas (08:00–18:00)")
	errNoSchedule   = apperr.New(http.StatusConflict, "no_schedule", "Bugun uchun amaliyot jadvali yo'q")
	errNoLocation   = apperr.New(http.StatusConflict, "no_location", "Jadvalga amaliyot joyi biriktirilmagan, adminga murojaat qiling")
	errNoEnrolled   = apperr.New(http.StatusConflict, "no_face_enrolled", "Yuz namunasi mavjud emas, adminga murojaat qiling")
	errShiftLocked  = apperr.New(http.StatusConflict, "shift_locked", "Bugun boshqa smenada qatnashgansiz, smenani almashtirib bo'lmaydi")
	errStepDone     = apperr.New(http.StatusConflict, "step_done", "Bu bosqich allaqachon tasdiqlangan")
	errExhausted    = apperr.New(http.StatusConflict, "attempts_exhausted", "Bu bosqich uchun urinishlar tugadi")
	errRejected     = apperr.New(http.StatusConflict, "step_rejected", "Bu bosqich o'qituvchi tomonidan tasdiqlanmagan, qayta topshirib bo'lmaydi")
	errFaceDown     = apperr.New(http.StatusServiceUnavailable, "face_unavailable", face.UnavailableMessage)
)

// Review states of an attempt / step (see migration 00005).
const (
	ReviewPending  = "pending"
	ReviewApproved = "approved"
	ReviewRejected = "rejected"
)

type AttemptService struct {
	pool        *pgxpool.Pool
	store       storage.Storage
	face        face.Service
	loc         *time.Location
	clock       func() time.Time
	maxAttempts int
	gpsMaxAcc   float64
	// mismatchLimit: every Nth face mismatch within one attendance blocks the student.
	mismatchLimit int
}

func NewAttemptService(pool *pgxpool.Pool, store storage.Storage, f face.Service, loc *time.Location, maxAttempts int, gpsMaxAccuracyM float64) *AttemptService {
	if maxAttempts <= 0 {
		maxAttempts = 10
	}
	if gpsMaxAccuracyM <= 0 {
		gpsMaxAccuracyM = 100
	}
	return &AttemptService{pool: pool, store: store, face: f, loc: loc, clock: time.Now, maxAttempts: maxAttempts, gpsMaxAcc: gpsMaxAccuracyM, mismatchLimit: 5}
}

// WithMismatchLimit sets how many face mismatches in one attendance block the student (<=0 keeps 5).
func (s *AttemptService) WithMismatchLimit(n int) *AttemptService {
	if n > 0 {
		s.mismatchLimit = n
	}
	return s
}

// WithClock replaces the time source (tests).
func (s *AttemptService) WithClock(c func() time.Time) *AttemptService {
	s.clock = c
	return s
}

type AttemptInput struct {
	Image     []byte
	Filename  string
	Latitude  float64
	Longitude float64
	Accuracy  float64
}

type AttemptResult struct {
	Success          bool     `json:"success"`
	Shift            int      `json:"shift"`
	Step             int      `json:"step"`
	FaceVerified     bool     `json:"face_verified"`
	LocationVerified bool     `json:"location_verified"`
	LivenessVerified bool     `json:"liveness_verified"`
	Failures         []string `json:"failures"`
	Message          string   `json:"message"`
	AttemptsUsed     int      `json:"attempts_used"`
	AttemptsLeft     int      `json:"attempts_left"`
	StepsDone        int      `json:"steps_done"`
	// Blocked: too many face mismatches; the account was deactivated and every session ended.
	Blocked bool `json:"blocked"`
}

func finite(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

func (in AttemptInput) validate() (string, error) {
	fields := map[string]string{}
	if !finite(in.Latitude, in.Longitude, in.Accuracy) || in.Latitude < -90 || in.Latitude > 90 || in.Longitude < -180 || in.Longitude > 180 {
		fields["latitude"] = "Joylashuv noto'g'ri"
	}
	if !finite(in.Accuracy) || in.Accuracy < 0 {
		fields["accuracy"] = "GPS aniqligi noto'g'ri"
	}
	var ext string
	switch {
	case len(in.Image) == 0:
		fields["image"] = "Rasmni yuboring"
	case int64(len(in.Image)) > storage.MaxSelfieSize:
		fields["image"] = "Rasm hajmi 5 MB dan oshmasligi kerak"
	default:
		head := in.Image
		if len(head) > 512 {
			head = head[:512]
		}
		e, err := storage.Validate(storage.KindSelfies, in.Filename, head)
		if err != nil {
			fields["image"] = "Rasm turi noto'g'ri (jpeg, png yoki webp)"
		}
		ext = e
	}
	if len(fields) > 0 {
		return "", apperr.Validation("Ma'lumotlar noto'g'ri", fields)
	}
	return ext, nil
}

// Attempt runs one confirmation attempt. See CLAUDE.md "Attendance rules".
func (s *AttemptService) Attempt(ctx context.Context, actor Actor, u db.User, in AttemptInput) (*AttemptResult, error) {
	now := s.clock()
	wall := now.In(s.loc)
	date := time.Date(wall.Year(), wall.Month(), wall.Day(), 0, 0, 0, 0, time.UTC)
	pgDate := pgtype.Date{Time: date, Valid: true}
	q := db.New(s.pool)

	// 1. server time decides shift and step
	win, ok := attendance.WindowAt(wall)
	if !ok {
		return nil, errOutsideHours
	}

	// 2. the active schedule for today
	scheds, err := q.ListSchedulesForStudentOnDate(ctx, db.ListSchedulesForStudentOnDateParams{OnDate: pgDate, StudentID: u.ID})
	if err != nil {
		return nil, err
	}
	if len(scheds) == 0 {
		return nil, errNoSchedule
	}
	sc := scheds[0]
	if !(attendance.Days{sc.Monday, sc.Tuesday, sc.Wednesday, sc.Thursday, sc.Friday, sc.Saturday}).Has(date.Weekday()) {
		return nil, errNoSchedule
	}
	if !sc.LocationID.Valid {
		return nil, errNoLocation
	}
	location, err := q.GetLocation(ctx, sc.LocationID.UUID)
	if err != nil {
		return nil, errNoLocation
	}
	var poly []geo.Point
	if err := json.Unmarshal(location.Polygon, &poly); err != nil || len(poly) < 3 {
		return nil, errNoLocation
	}
	if len(u.FaceEmbedding) == 0 {
		return nil, errNoEnrolled
	}
	ext, err := in.validate()
	if err != nil {
		return nil, err
	}

	// Cheap pre-checks before the slow face call (re-checked authoritatively in the transaction).
	if att, err := q.LockAttendanceNoLock(ctx, db.LockAttendanceNoLockParams{StudentID: u.ID, ScheduleID: sc.ID, Date: pgDate}); err == nil {
		if int(att.Shift) != win.Shift {
			return nil, errShiftLocked
		}
		if st, err := q.GetStep(ctx, db.GetStepParams{AttendanceID: att.ID, Step: int16(win.Step)}); err == nil {
			if st.Review == ReviewRejected {
				return nil, errRejected
			}
			if st.Success {
				return nil, errStepDone
			}
			if n, err := q.CountStepAttempts(ctx, st.ID); err == nil && int(n) >= s.maxAttempts {
				return nil, errExhausted
			}
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// 5a. location
	var failures []string
	geoRes := geo.Check(poly, geo.Point{in.Latitude, in.Longitude}, float64(location.ToleranceMeters))
	accuracyOK := in.Accuracy <= s.gpsMaxAcc
	if !accuracyOK {
		failures = append(failures, FailLocationAccuracy)
	}
	if !geoRes.OK {
		failures = append(failures, FailLocation)
	}
	locationOK := geoRes.OK && accuracyOK

	// 5b. face + liveness (before the transaction: it can take seconds).
	// When the face-service is busy or unreachable the face check is skipped and an attempt whose
	// location is fine is accepted for manual review (teacher/admin confirm it later).
	var faceOK, liveOK bool
	var dist, thr, spoof *float64
	deferFace := false
	if busy, berr := s.face.Busy(ctx); berr != nil || busy {
		deferFace = true
	} else {
		vr, ferr := s.face.Verify(ctx, in.Image, u.FaceEmbedding)
		switch {
		case ferr == nil:
			liveOK = vr.IsReal
			faceOK = vr.IsReal && vr.Verified
			dist, thr, spoof = vr.Distance, &vr.Threshold, vr.AntispoofScore
			if !vr.IsReal {
				failures = append(failures, FailLiveness)
			} else if !vr.Verified {
				failures = append(failures, FailFace)
			}
		case errors.Is(ferr, face.ErrNoFace):
			failures = append(failures, FailNoFace)
		case errors.Is(ferr, face.ErrMultipleFaces):
			failures = append(failures, FailMultipleFaces)
		case errors.Is(ferr, face.ErrInvalidImage):
			return nil, apperr.Validation("Ma'lumotlar noto'g'ri", map[string]string{"image": "Rasmni o'qib bo'lmadi"})
		case errors.Is(ferr, face.ErrUnavailable):
			if !locationOK {
				return nil, errFaceDown // not the student's fault: no attempt is consumed
			}
			deferFace = true
		default:
			return nil, ferr
		}
	}
	review := ""
	success := faceOK && liveOK && locationOK
	if deferFace && locationOK {
		success, review = true, ReviewPending
	}
	if failures == nil {
		failures = []string{}
	}

	// selfie
	path, err := s.store.Save(ctx, storage.KindSelfies, ext, bytes.NewReader(in.Image))
	if err != nil {
		return nil, err
	}

	// 3-4, 6, 7: one transaction, serialised per attendance row
	res := &AttemptResult{Success: success, Shift: win.Shift, Step: win.Step, FaceVerified: faceOK,
		LocationVerified: locationOK, LivenessVerified: liveOK, Failures: failures}
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		if err := q.EnsureAttendance(ctx, db.EnsureAttendanceParams{StudentID: u.ID, ScheduleID: sc.ID, Shift: int16(win.Shift), Date: pgDate}); err != nil {
			return err
		}
		att, err := q.LockAttendance(ctx, db.LockAttendanceParams{StudentID: u.ID, ScheduleID: sc.ID, Date: pgDate})
		if err != nil {
			return err
		}
		if int(att.Shift) != win.Shift {
			return errShiftLocked
		}
		if err := q.EnsureStep(ctx, db.EnsureStepParams{AttendanceID: att.ID, Step: int16(win.Step)}); err != nil {
			return err
		}
		st, err := q.GetStep(ctx, db.GetStepParams{AttendanceID: att.ID, Step: int16(win.Step)})
		if err != nil {
			return err
		}
		if st.Review == ReviewRejected {
			return errRejected
		}
		if st.Success {
			return errStepDone
		}
		used, err := q.CountStepAttempts(ctx, st.ID)
		if err != nil {
			return err
		}
		if int(used) >= s.maxAttempts {
			return errExhausted
		}

		p := db.CreateAttemptParams{
			StepID: st.ID, Image: path, UserAgent: actor.UserAgent, Latitude: in.Latitude, Longitude: in.Longitude,
			GpsAccuracy: in.Accuracy, LocationID: uuidNull(location.ID), FaceVerified: faceOK, LocationVerified: locationOK,
			LivenessVerified: liveOK, Success: success, Error: failureSummary(failures),
			FaceDistance: float8(dist), FaceThreshold: float8(thr), AntispoofScore: float8(spoof), Review: review,
		}
		if addr, perr := netip.ParseAddr(actor.IP); perr == nil {
			p.IpAddress = &addr
		}
		attempt, err := q.CreateAttempt(ctx, p)
		if err != nil {
			return err
		}
		switch {
		case review == ReviewPending:
			if err := q.MarkStepPending(ctx, st.ID); err != nil {
				return err
			}
		case success:
			if err := q.MarkStepSuccess(ctx, st.ID); err != nil {
				return err
			}
		}
		res.AttemptsUsed = int(used) + 1
		res.AttemptsLeft = s.maxAttempts - res.AttemptsUsed
		if success {
			res.AttemptsLeft = max(0, res.AttemptsLeft)
		}
		steps, err := q.ListStepsWithAttemptCounts(ctx, att.ID)
		if err != nil {
			return err
		}
		for _, r := range steps {
			if r.Success {
				res.StepsDone++
			}
		}
		if err := audit.Write(ctx, tx, actor.entry(audit.ActionAttendanceAttempt, "attendance_attempt", &attempt.ID, nil, map[string]any{
			"attendance_id": att.ID, "shift": win.Shift, "step": win.Step, "success": success, "face_verified": faceOK,
			"location_verified": locationOK, "liveness_verified": liveOK, "failures": failures, "review": review,
		})); err != nil {
			return err
		}
		return s.blockOnRepeatedMismatch(ctx, tx, q, actor, u, att.ID, failures, res)
	})
	if err != nil {
		_ = s.store.Delete(ctx, path) // the attempt was not recorded: do not keep its selfie
		return nil, mapDBError(err)
	}
	if res.Blocked {
		res.Message = AccountInactiveMessage
	} else if success {
		res.Message = "Tasdiqlandi"
	} else {
		res.Message = "Tasdiqlanmadi: " + failureSummary(failures)
	}
	return res, nil
}

// FailureSummary renders failure codes as the Uzbek text stored on an attempt.
func FailureSummary(codes []string) string { return failureSummary(codes) }

func failureSummary(codes []string) string {
	parts := make([]string, 0, len(codes))
	for _, c := range codes {
		parts = append(parts, failureText[c])
	}
	return strings.Join(parts, "; ")
}

func float8(f *float64) pgtype.Float8 {
	if f == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *f, Valid: true}
}

// blockOnRepeatedMismatch deactivates the student and ends their sessions on every Nth face mismatch
// within one attendance (so an admin who reactivates the account gives them N more tries).
// It runs inside the attempt transaction, so the block and the attempt record commit together.
func (s *AttemptService) blockOnRepeatedMismatch(ctx context.Context, tx pgx.Tx, q *db.Queries, actor Actor, u db.User, attendanceID uuid.UUID, failures []string, res *AttemptResult) error {
	mismatch := false
	for _, f := range failures {
		mismatch = mismatch || f == FailFace
	}
	if !mismatch {
		return nil
	}
	n, err := q.CountFaceMismatches(ctx, attendanceID)
	if err != nil {
		return err
	}
	if n == 0 || int(n)%s.mismatchLimit != 0 {
		return nil
	}
	if _, err := q.SetUserActive(ctx, db.SetUserActiveParams{ID: u.ID, IsActive: false}); err != nil {
		return err
	}
	if err := q.DeleteUserSessions(ctx, u.ID); err != nil {
		return err
	}
	res.Blocked = true
	res.Message = AccountInactiveMessage
	return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "user", &u.ID,
		map[string]any{"is_active": true},
		map[string]any{"is_active": false, "reason": "face_mismatch_limit", "attendance_id": attendanceID, "mismatches": n}))
}
