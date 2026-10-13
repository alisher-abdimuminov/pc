package http_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"pc/internal/hemis"
	apphttp "pc/internal/http"
)

type syncEnv struct {
	*adminEnv
	src *hemis.FakeGroups
}

func newSyncEnv(t *testing.T, configured bool) *syncEnv {
	src := hemis.NewFakeGroups()
	e := newEnvDeps(t, func(d *apphttp.Deps) {
		if configured {
			d.HemisGroups = src
		}
	})
	e.user("adm", "admin", false)
	return &syncEnv{adminEnv: &adminEnv{env: e, cookie: e.mustLogin("adm")}, src: src}
}

func (s *syncEnv) sync(want int) map[string]any {
	s.t.Helper()
	return s.call("POST", "/groups/sync", nil, want)
}

func (s *syncEnv) groups() map[int64]string { // hemis_id -> name (0 for hand-made)
	rows, err := s.pool.Query(context.Background(), `SELECT COALESCE(hemis_id, 0), name FROM groups ORDER BY name`)
	if err != nil {
		s.t.Fatal(err)
	}
	defer rows.Close()
	out := map[int64]string{}
	var manual []string
	for rows.Next() {
		var id int64
		var name string
		rows.Scan(&id, &name)
		if id == 0 {
			manual = append(manual, name)
			continue
		}
		out[id] = name
	}
	if len(manual) > 0 {
		out[0] = strings.Join(manual, "|")
	}
	return out
}

func counts(m map[string]any) [4]float64 {
	return [4]float64{m["total"].(float64), m["created"].(float64), m["updated"].(float64), m["unchanged"].(float64)}
}

func TestSyncCreatesUpdatesAndIsIdempotent(t *testing.T) {
	s := newSyncEnv(t, true)
	ctx := context.Background()
	s.src.Set(hemis.Group{ID: 1, Name: "101-guruh"}, hemis.Group{ID: 2, Name: "102-guruh"}, hemis.Group{ID: 3, Name: "103-guruh"})

	if got := counts(s.sync(200)); got != [4]float64{3, 3, 0, 0} {
		t.Fatalf("first sync: %v", got)
	}
	if g := s.groups(); len(g) != 3 || g[1] != "101-guruh" || g[3] != "103-guruh" {
		t.Fatalf("groups: %v", g)
	}
	if s.audits("create", "group") != 3 {
		t.Fatalf("creates audited: %d", s.audits("create", "group"))
	}

	// nothing changed: nothing written, nothing audited
	if got := counts(s.sync(200)); got != [4]float64{3, 0, 0, 3} {
		t.Fatalf("second sync: %v", got)
	}
	if s.audits("create", "group") != 3 || s.audits("update", "group") != 0 {
		t.Fatal("an idempotent sync must not audit anything")
	}

	// a rename at HEMIS is applied to the same row; teacher, faculty and students stay attached
	var gid, fid, tid string
	s.pool.QueryRow(ctx, `SELECT id FROM groups WHERE hemis_id=2`).Scan(&gid)
	s.pool.QueryRow(ctx, `INSERT INTO faculties (name) VALUES ('F') RETURNING id`).Scan(&fid)
	tid = s.user("tea", "teacher", false).ID.String()
	s.pool.Exec(ctx, `UPDATE groups SET faculty_id=$2, teacher_id=$3 WHERE id=$1`, gid, fid, tid)
	stu := s.user("stu", "student", false)
	s.pool.Exec(ctx, `UPDATE users SET group_id=$2 WHERE id=$1`, stu.ID, gid)

	s.src.Set(hemis.Group{ID: 1, Name: "101-guruh"}, hemis.Group{ID: 2, Name: "102-A guruh"}, hemis.Group{ID: 3, Name: "103-guruh"}, hemis.Group{ID: 4, Name: "104-guruh"})
	if got := counts(s.sync(200)); got != [4]float64{4, 1, 1, 2} {
		t.Fatalf("third sync: %v", got)
	}
	var name, gotFac, gotTea string
	s.pool.QueryRow(ctx, `SELECT name, faculty_id::text, teacher_id::text FROM groups WHERE id=$1`, gid).Scan(&name, &gotFac, &gotTea)
	if name != "102-A guruh" || gotFac != fid || gotTea != tid {
		t.Fatalf("rename lost data: %q %q %q", name, gotFac, gotTea)
	}
	var stuGroup string
	s.pool.QueryRow(ctx, `SELECT group_id::text FROM users WHERE id=$1`, stu.ID).Scan(&stuGroup)
	if stuGroup != gid {
		t.Fatal("students must stay in the renamed group")
	}
	var ch string
	s.pool.QueryRow(ctx, `SELECT changes::text FROM audit_logs WHERE action='update' AND entity_type='group' AND entity_id=$1`, gid).Scan(&ch)
	if !strings.Contains(ch, "102-guruh") || !strings.Contains(ch, "102-A guruh") {
		t.Fatalf("rename audit: %s", ch)
	}
}

func TestSyncAdoptsHandMadeGroupsByName(t *testing.T) {
	s := newSyncEnv(t, true)
	ctx := context.Background()
	var manual, linkedElsewhere string
	s.pool.QueryRow(ctx, `INSERT INTO groups (name) VALUES ('201-guruh') RETURNING id`).Scan(&manual)
	s.pool.QueryRow(ctx, `INSERT INTO groups (name, hemis_id) VALUES ('202-guruh', 999) RETURNING id`).Scan(&linkedElsewhere)

	s.src.Set(hemis.Group{ID: 20, Name: "201-guruh"}, hemis.Group{ID: 21, Name: "202-guruh"})
	if got := counts(s.sync(200)); got != [4]float64{2, 1, 1, 0} {
		t.Fatalf("%v", got)
	}
	var hid string
	s.pool.QueryRow(ctx, `SELECT hemis_id::text FROM groups WHERE id=$1`, manual).Scan(&hid)
	if hid != "20" {
		t.Fatalf("hand-made group must be linked, got %q", hid)
	}
	// a group already tied to another HEMIS id is NOT hijacked just because the name matches
	s.pool.QueryRow(ctx, `SELECT hemis_id::text FROM groups WHERE id=$1`, linkedElsewhere).Scan(&hid)
	if hid != "999" {
		t.Fatalf("group with another hemis_id was overwritten: %q", hid)
	}
	var n int
	s.pool.QueryRow(ctx, `SELECT count(*) FROM groups WHERE name='202-guruh'`).Scan(&n)
	if n != 2 {
		t.Fatalf("a separate group must be created for HEMIS id 21, have %d named 202-guruh", n)
	}
	// two HEMIS groups with one hand-made namesake: only one adopts it
	s2 := newSyncEnv(t, true)
	s2.pool.Exec(ctx, `INSERT INTO groups (name) VALUES ('301-guruh')`)
	s2.src.Set(hemis.Group{ID: 30, Name: "301-guruh"}, hemis.Group{ID: 31, Name: "301-guruh"})
	if got := counts(s2.sync(200)); got != [4]float64{2, 1, 1, 0} {
		t.Fatalf("duplicate names: %v", got)
	}
}

func TestSyncFailuresWriteNothing(t *testing.T) {
	s := newSyncEnv(t, true)
	s.pool.Exec(context.Background(), `INSERT INTO groups (name) VALUES ('keep')`)

	s.src.SetErr(hemis.ErrUnavailable)
	out := s.call("POST", "/groups/sync", nil, 502)
	if e := out["error"].(map[string]any); e["code"] != "hemis_unavailable" || !strings.Contains(e["message"].(string), "HEMIS") {
		t.Fatalf("outage: %v", out)
	}
	s.src.SetErr(hemis.ErrNotConfigured)
	if e := s.call("POST", "/groups/sync", nil, 503)["error"].(map[string]any); e["code"] != "hemis_not_configured" {
		t.Fatalf("no token: %v", e)
	}
	if g := s.groups(); len(g) != 1 || g[0] != "keep" {
		t.Fatalf("failed syncs must not change groups: %v", g)
	}
	if s.audits("create", "group") != 0 {
		t.Fatal("nothing may be audited")
	}
}

func TestSyncWithoutAConfiguredSource(t *testing.T) {
	s := newSyncEnv(t, false)
	out := s.call("POST", "/groups/sync", nil, 503)
	if out["error"].(map[string]any)["code"] != "hemis_not_configured" {
		t.Fatalf("%v", out)
	}
}

func TestSyncIsAdminOnly(t *testing.T) {
	s := newSyncEnv(t, true)
	s.src.Set(hemis.Group{ID: 1, Name: "G"})
	s.user("tea", "teacher", false)
	tea := s.mustLogin("tea")
	for name, cookie := range map[string]string{"teacher": tea, "anonymous": ""} {
		want := 403
		if cookie == "" {
			want = 401
		}
		if r, _ := s.do("POST", "/api/admin/groups/sync", "", cookie); r.StatusCode != want {
			t.Errorf("%s: got %d want %d", name, r.StatusCode, want)
		}
	}
	if s.src.Calls() != 0 {
		t.Fatal("HEMIS must not be called for unauthorised requests")
	}
	if g := s.groups(); len(g) != 0 {
		t.Fatal("no group may be created")
	}
}

func TestConcurrentSyncsDoNotDuplicateGroups(t *testing.T) {
	s := newSyncEnv(t, true)
	var groups []hemis.Group
	for i := 1; i <= 40; i++ {
		groups = append(groups, hemis.Group{ID: int64(i), Name: "G-" + string(rune('A'+i%26)) + string(rune('a'+i/26))})
	}
	s.src.Set(groups...)
	var wg sync.WaitGroup
	codes := make([]int, 4)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, _ := s.do("POST", "/api/admin/groups/sync", "", s.cookie)
			codes[i] = r.StatusCode
		}(i)
	}
	wg.Wait()
	for _, c := range codes {
		if c != 200 {
			t.Fatalf("statuses %v", codes)
		}
	}
	var n, distinct int
	s.pool.QueryRow(context.Background(), `SELECT count(*), count(DISTINCT hemis_id) FROM groups`).Scan(&n, &distinct)
	if n != 40 || distinct != 40 {
		t.Fatalf("%d groups, %d distinct hemis ids, want 40/40", n, distinct)
	}
	if s.audits("create", "group") != 40 {
		t.Fatalf("each group created once: %d", s.audits("create", "group"))
	}
}

func TestSyncSetsFacultyFromDepartment(t *testing.T) {
	s := newSyncEnv(t, true)
	ctx := context.Background()
	var known, manualFac, oldFac, manualGroup string
	s.pool.QueryRow(ctx, `INSERT INTO faculties (name, hemis_id) VALUES ('Tillar', 1) RETURNING id`).Scan(&known)
	s.pool.QueryRow(ctx, `INSERT INTO faculties (name) VALUES ('Fizika') RETURNING id`).Scan(&manualFac)
	s.pool.QueryRow(ctx, `INSERT INTO faculties (name) VALUES ('Eski') RETURNING id`).Scan(&oldFac)
	s.pool.QueryRow(ctx, `INSERT INTO groups (name, hemis_id, faculty_id) VALUES ('G-old', 10, $1) RETURNING id`, oldFac).Scan(new(string))
	s.pool.QueryRow(ctx, `INSERT INTO groups (name) VALUES ('G-manual') RETURNING id`).Scan(&manualGroup)
	s.pool.QueryRow(ctx, `INSERT INTO groups (name, hemis_id, faculty_id) VALUES ('G-nodept', 14, $1) RETURNING id`, oldFac).Scan(new(string))

	s.src.Set(
		hemis.Group{ID: 10, Name: "G-old", FacultyID: 1, FacultyName: "Tillar"},            // existing group moves to a known faculty
		hemis.Group{ID: 11, Name: "G-new", FacultyID: 1, FacultyName: "Tillar"},            // new group, known faculty
		hemis.Group{ID: 12, Name: "G-adopt", FacultyID: 2, FacultyName: "Fizika"},          // hand-made faculty is adopted
		hemis.Group{ID: 13, Name: "G-create", FacultyID: 3, FacultyName: "Yangi fakultet"}, // faculty is created
		hemis.Group{ID: 15, Name: "G-manual", FacultyID: 3, FacultyName: "Yangi fakultet"}, // hand-made group adopted + faculty
		hemis.Group{ID: 14, Name: "G-nodept"},                                              // no department: faculty kept
	)
	if got := counts(s.sync(200)); got != [4]float64{6, 3, 2, 1} {
		t.Fatalf("sync: %v", got)
	}
	fac := func(group string) string {
		var n string
		s.pool.QueryRow(ctx, `SELECT COALESCE(f.name,'') FROM groups g LEFT JOIN faculties f ON f.id=g.faculty_id WHERE g.name=$1`, group).Scan(&n)
		return n
	}
	for g, want := range map[string]string{"G-old": "Tillar", "G-new": "Tillar", "G-adopt": "Fizika", "G-create": "Yangi fakultet", "G-manual": "Yangi fakultet", "G-nodept": "Eski"} {
		if got := fac(g); got != want {
			t.Errorf("%s: faculty %q, want %q", g, got, want)
		}
	}
	var n int
	s.pool.QueryRow(ctx, `SELECT count(*) FROM faculties`).Scan(&n)
	if n != 4 { // Tillar, Fizika, Eski, Yangi fakultet: nothing duplicated
		t.Fatalf("faculties: %d", n)
	}
	var hid string
	s.pool.QueryRow(ctx, `SELECT hemis_id::text FROM faculties WHERE id=$1`, manualFac).Scan(&hid)
	if hid != "2" {
		t.Fatalf("hand-made faculty not linked: %q", hid)
	}

	if got := counts(s.sync(200)); got != [4]float64{6, 0, 0, 6} {
		t.Fatalf("second sync must change nothing: %v", got)
	}
}
