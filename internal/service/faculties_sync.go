package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/audit"
	"pc/internal/db"
	"pc/internal/hemis"
)

// FacultySyncService copies the faculty list from HEMIS into our faculties.
type FacultySyncService struct {
	pool   *pgxpool.Pool
	source hemis.FacultySource // nil when no HEMIS token is configured
}

func NewFacultySyncService(pool *pgxpool.Pool, source hemis.FacultySource) *FacultySyncService {
	return &FacultySyncService{pool: pool, source: source}
}

const facultySyncLockKey = 7_301_002

// Sync fetches every HEMIS faculty and, in one transaction:
//   - a faculty with the same hemis_id: renamed when the name changed;
//   - else a hand-made faculty (no hemis_id yet) with the same name: linked to the HEMIS id;
//   - else a new faculty is created.
//
// Groups and students are never touched. Faculties that disappeared from HEMIS are kept.
// If HEMIS fails, nothing is written.
func (s *FacultySyncService) Sync(ctx context.Context, actor Actor) (SyncResult, error) {
	var res SyncResult
	if s.source == nil {
		return res, errSyncNotConfigured
	}
	remote, err := s.source.ListFaculties(ctx)
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
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, facultySyncLockKey); err != nil {
			return err
		}
		res = SyncResult{Total: len(remote)}
		for _, f := range remote {
			hid := pgtype.Int8{Int64: f.ID, Valid: true}

			existing, err := q.GetFacultyByHemisID(ctx, hid)
			if err == nil {
				if existing.Name == f.Name {
					res.Unchanged++
					continue
				}
				if err := q.RenameFaculty(ctx, db.RenameFacultyParams{ID: existing.ID, Name: f.Name}); err != nil {
					return err
				}
				if err := audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "faculty", &existing.ID,
					map[string]any{"name": existing.Name}, map[string]any{"name": f.Name, "source": "hemis_sync"})); err != nil {
					return err
				}
				res.Updated++
				continue
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}

			if manual, err := q.FindUnlinkedFacultyByName(ctx, f.Name); err == nil {
				if err := q.SetFacultyHemisID(ctx, db.SetFacultyHemisIDParams{ID: manual.ID, HemisID: hid}); err != nil {
					return err
				}
				if err := audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "faculty", &manual.ID,
					map[string]any{"hemis_id": nil}, map[string]any{"hemis_id": f.ID, "source": "hemis_sync"})); err != nil {
					return err
				}
				res.Updated++
				continue
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}

			created, err := q.CreateFaculty(ctx, db.CreateFacultyParams{Name: f.Name, HemisID: hid})
			if err != nil {
				return err
			}
			if err := audit.Write(ctx, tx, actor.entry(audit.ActionCreate, "faculty", &created.ID, nil,
				map[string]any{"name": f.Name, "hemis_id": f.ID, "source": "hemis_sync"})); err != nil {
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
