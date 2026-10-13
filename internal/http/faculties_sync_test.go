package http_test

import (
	"context"
	"strings"
	"testing"

	"pc/internal/hemis"
	apphttp "pc/internal/http"
)

func newFacultySyncEnv(t *testing.T, src *hemis.FakeFaculties) *adminEnv {
	e := newEnvDeps(t, func(d *apphttp.Deps) {
		if src != nil {
			d.HemisFaculties = src
		}
	})
	e.user("adm", "admin", false)
	return &adminEnv{env: e, cookie: e.mustLogin("adm")}
}

func TestFacultySyncCreatesRenamesAdoptsAndKeepsGroups(t *testing.T) {
	src := hemis.NewFakeFaculties(hemis.Faculty{ID: 1, Name: "Tillar"}, hemis.Faculty{ID: 2, Name: "Fizika"})
	s := newFacultySyncEnv(t, src)
	ctx := context.Background()
	var manual string
	s.pool.QueryRow(ctx, `INSERT INTO faculties (name) VALUES ('Fizika') RETURNING id`).Scan(&manual)

	if got := counts(s.call("POST", "/faculties/sync", nil, 200)); got != [4]float64{2, 1, 1, 0} {
		t.Fatalf("first sync: %v", got)
	}
	var hid string
	s.pool.QueryRow(ctx, `SELECT hemis_id::text FROM faculties WHERE id=$1`, manual).Scan(&hid)
	if hid != "2" {
		t.Fatalf("hand-made faculty must be linked, got %q", hid)
	}
	if got := counts(s.call("POST", "/faculties/sync", nil, 200)); got != [4]float64{2, 0, 0, 2} {
		t.Fatalf("idempotent sync: %v", got)
	}

	var fid string
	s.pool.QueryRow(ctx, `SELECT id FROM faculties WHERE hemis_id=1`).Scan(&fid)
	s.pool.Exec(ctx, `INSERT INTO groups (name, faculty_id) VALUES ('G', $1)`, fid)
	src.Set(hemis.Faculty{ID: 1, Name: "Tillar fakulteti"}, hemis.Faculty{ID: 2, Name: "Fizika"})
	if got := counts(s.call("POST", "/faculties/sync", nil, 200)); got != [4]float64{2, 0, 1, 1} {
		t.Fatalf("rename sync: %v", got)
	}
	var name, gf string
	s.pool.QueryRow(ctx, `SELECT f.name, g.faculty_id::text FROM faculties f JOIN groups g ON g.faculty_id=f.id WHERE f.id=$1`, fid).Scan(&name, &gf)
	if name != "Tillar fakulteti" || gf != fid {
		t.Fatalf("rename lost the group link: %q %q", name, gf)
	}
	var n int
	s.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE entity_type='faculty' AND changes::text LIKE '%hemis_sync%'`).Scan(&n)
	if n != 3 { // create + adopt + rename
		t.Fatalf("audit entries: %d", n)
	}
}

func TestFacultySyncFailuresAndAccess(t *testing.T) {
	src := hemis.NewFakeFaculties(hemis.Faculty{ID: 1, Name: "X"})
	s := newFacultySyncEnv(t, src)
	src.SetErr(hemis.ErrUnavailable)
	if e := s.call("POST", "/faculties/sync", nil, 502)["error"].(map[string]any); e["code"] != "hemis_unavailable" || !strings.Contains(e["message"].(string), "HEMIS") {
		t.Fatalf("outage: %v", e)
	}
	src.SetErr(hemis.ErrNotConfigured)
	if e := s.call("POST", "/faculties/sync", nil, 503)["error"].(map[string]any); e["code"] != "hemis_not_configured" {
		t.Fatalf("no token: %v", e)
	}
	var n int
	s.pool.QueryRow(context.Background(), `SELECT count(*) FROM faculties`).Scan(&n)
	if n != 0 {
		t.Fatal("failed syncs must not write")
	}

	s.user("tea", "teacher", false)
	if r, _ := s.do("POST", "/api/admin/faculties/sync", "", s.mustLogin("tea")); r.StatusCode != 403 {
		t.Fatalf("teacher got %d", r.StatusCode)
	}
	if r, _ := s.do("POST", "/api/admin/faculties/sync", "", ""); r.StatusCode != 401 {
		t.Fatalf("anonymous got %d", r.StatusCode)
	}

	none := newFacultySyncEnv(t, nil)
	if e := none.call("POST", "/faculties/sync", nil, 503)["error"].(map[string]any); e["code"] != "hemis_not_configured" {
		t.Fatalf("unconfigured: %v", e)
	}
}
