package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/audit"
	"pc/internal/db"
	"pc/internal/hemis"
)

// GroupSyncService copies the group list from HEMIS into our groups.
type GroupSyncService struct {
	pool   *pgxpool.Pool
	source hemis.GroupSource // nil when no groups token is configured
}

func NewGroupSyncService(pool *pgxpool.Pool, source hemis.GroupSource) *GroupSyncService {
	return &GroupSyncService{pool: pool, source: source}
}

type SyncResult struct {
	Total     int `json:"total"`
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
}

var (
	errSyncNotConfigured = apperr.New(http.StatusServiceUnavailable, "hemis_not_configured", "HEMIS guruhlar tokeni sozlanmagan (HEMIS_GROUPS_TOKEN)")
	errSyncUnavailable   = apperr.New(http.StatusBadGateway, "hemis_unavailable", hemis.UnavailableMessage)
)

// advisoryKey serialises concurrent syncs (two admins pressing the button together).
const syncLockKey = 7_301_001

// Sync fetches every HEMIS group and, in one transaction:
//   - a group with the same hemis_id: renamed when the name changed;
//   - else a hand-made group (no hemis_id yet) with the same name: linked to the HEMIS id;
//   - else a new group is created.
//
// Each group's faculty comes from its HEMIS department: the faculty with that hemis_id is used, else a
// hand-made one with the same name is adopted, else it is created. A group without a department keeps its
// current faculty. Teachers and students are never touched. Groups that disappeared from
// HEMIS are kept. If HEMIS fails, nothing is written.
func (s *GroupSyncService) Sync(ctx context.Context, actor Actor) (SyncResult, error) {
	var res SyncResult
	if s.source == nil {
		return res, errSyncNotConfigured
	}
	remote, err := s.source.ListGroups(ctx) // network first, so no transaction is held open meanwhile
	switch {
	case err == nil:
	case errors.Is(err, hemis.ErrNotConfigured):
		return res, errSyncNotConfigured
	case errors.Is(err, hemis.ErrUnavailable):
		return res, errSyncUnavailable
	default:
		return res, err
	}
	res.Total = len(remote)

	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, syncLockKey); err != nil {
			return err
		}
		res = SyncResult{Total: len(remote)} // reset in case the transaction body is retried
		faculties := map[int64]uuid.UUID{}   // HEMIS department id -> our faculty
		for _, g := range remote {
			hid := pgtype.Int8{Int64: g.ID, Valid: true}
			var fac uuid.NullUUID
			if g.FacultyID != 0 {
				id, ok := faculties[g.FacultyID]
				if !ok {
					var err error
					if id, err = syncFaculty(ctx, tx, q, actor, g.FacultyID, g.FacultyName); err != nil {
						return err
					}
					faculties[g.FacultyID] = id
				}
				fac = uuid.NullUUID{UUID: id, Valid: true}
			}

			existing, err := q.GetGroupByHemisID(ctx, hid)
			if err == nil {
				renamed := existing.Name != g.Name
				moved := fac.Valid && existing.FacultyID != fac
				if !renamed && !moved {
					res.Unchanged++
					continue
				}
				before, after := map[string]any{}, map[string]any{"source": "hemis_sync"}
				if renamed {
					if err := q.RenameGroup(ctx, db.RenameGroupParams{ID: existing.ID, Name: g.Name}); err != nil {
						return err
					}
					before["name"], after["name"] = existing.Name, g.Name
				}
				if moved {
					if err := q.SetGroupFaculty(ctx, db.SetGroupFacultyParams{ID: existing.ID, FacultyID: fac}); err != nil {
						return err
					}
					before["faculty_id"], after["faculty_id"] = nullableUUID(existing.FacultyID), fac.UUID
				}
				if err := audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "group", &existing.ID, before, after)); err != nil {
					return err
				}
				res.Updated++
				continue
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}

			// A group typed in by hand before the sync existed: adopt it instead of duplicating it.
			if manual, err := q.FindUnlinkedGroupByName(ctx, g.Name); err == nil {
				if err := q.SetGroupHemisID(ctx, db.SetGroupHemisIDParams{ID: manual.ID, HemisID: hid}); err != nil {
					return err
				}
				before, after := map[string]any{"hemis_id": nil}, map[string]any{"hemis_id": g.ID, "source": "hemis_sync"}
				if fac.Valid && manual.FacultyID != fac {
					if err := q.SetGroupFaculty(ctx, db.SetGroupFacultyParams{ID: manual.ID, FacultyID: fac}); err != nil {
						return err
					}
					before["faculty_id"], after["faculty_id"] = nullableUUID(manual.FacultyID), fac.UUID
				}
				if err := audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "group", &manual.ID, before, after)); err != nil {
					return err
				}
				res.Updated++
				continue
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}

			created, err := q.CreateGroup(ctx, db.CreateGroupParams{Name: g.Name, HemisID: hid, FacultyID: fac})
			if err != nil {
				return err
			}
			if err := audit.Write(ctx, tx, actor.entry(audit.ActionCreate, "group", &created.ID, nil,
				map[string]any{"name": g.Name, "hemis_id": g.ID, "faculty_id": nullableUUID(fac), "source": "hemis_sync"})); err != nil {
				return err
			}
			res.Created++
		}
		return nil
	})
	if err != nil {
		return SyncResult{}, mapDBError(err)
	}
	return res, nil
}

func nullableUUID(u uuid.NullUUID) any {
	if !u.Valid {
		return nil
	}
	return u.UUID
}

// syncFaculty returns our faculty for a HEMIS department: matched by hemis_id, else a hand-made faculty with
// the same name is adopted, else a new one is created. An existing faculty is never renamed here.
func syncFaculty(ctx context.Context, tx pgx.Tx, q *db.Queries, actor Actor, hemisID int64, name string) (uuid.UUID, error) {
	hid := pgtype.Int8{Int64: hemisID, Valid: true}
	if f, err := q.GetFacultyByHemisID(ctx, hid); err == nil {
		return f.ID, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	if f, err := q.FindUnlinkedFacultyByName(ctx, name); err == nil {
		if err := q.SetFacultyHemisID(ctx, db.SetFacultyHemisIDParams{ID: f.ID, HemisID: hid}); err != nil {
			return uuid.Nil, err
		}
		return f.ID, audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "faculty", &f.ID,
			map[string]any{"hemis_id": nil}, map[string]any{"hemis_id": hemisID, "source": "hemis_sync"}))
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	f, err := q.CreateFaculty(ctx, db.CreateFacultyParams{Name: name, HemisID: hid})
	if err != nil {
		return uuid.Nil, err
	}
	return f.ID, audit.Write(ctx, tx, actor.entry(audit.ActionCreate, "faculty", &f.ID, nil,
		map[string]any{"name": name, "hemis_id": hemisID, "source": "hemis_sync"}))
}
