package jobs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pc/internal/db"
	"pc/internal/storage"
	"pc/internal/testutil"
)

func TestCleanupSelfiesRemovesOldFilesButKeepsRows(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	dir := t.TempDir()
	store := storage.NewLocal(dir)

	var student, group, loc, sched, att, step string
	pool.QueryRow(ctx, `INSERT INTO groups (name) VALUES ('g') RETURNING id`).Scan(&group)
	pool.QueryRow(ctx, `INSERT INTO users (username, role, group_id) VALUES ('s','student',$1) RETURNING id`, group).Scan(&student)
	pool.QueryRow(ctx, `INSERT INTO locations (name, polygon) VALUES ('l','[[1,1],[1,2],[2,2]]') RETURNING id`).Scan(&loc)
	pool.QueryRow(ctx, `INSERT INTO schedules (start_date,end_date,group_id,location_id) VALUES (current_date,current_date,$1,$2) RETURNING id`, group, loc).Scan(&sched)
	pool.QueryRow(ctx, `INSERT INTO attendances (student_id,schedule_id,shift,date) VALUES ($1,$2,1,current_date) RETURNING id`, student, sched).Scan(&att)
	pool.QueryRow(ctx, `INSERT INTO attendance_steps (attendance_id, step) VALUES ($1,1) RETURNING id`, att).Scan(&step)

	save := func() string {
		p, err := store.Save(ctx, storage.KindSelfies, "jpg", strings.NewReader("x"))
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	oldA, oldB, fresh := save(), save(), save()
	insert := func(path string, age string) string {
		var id string
		pool.QueryRow(ctx, `INSERT INTO attendance_attempts (step_id, image, success, attempted_at) VALUES ($1,$2,true, now() - $3::interval) RETURNING id`, step, path, age).Scan(&id)
		return id
	}
	idA, idB, idFresh := insert(oldA, "31 days"), insert(oldB, "90 days"), insert(fresh, "29 days")
	idNoImage := insert("", "200 days") // already cleaned: nothing to do

	n, err := CleanupSelfies(ctx, db.New(pool), store, time.Now().AddDate(0, 0, -30))
	if err != nil || n != 2 {
		t.Fatalf("removed %d, err %v; want 2", n, err)
	}
	for _, p := range []string{oldA, oldB} {
		if _, err := os.Stat(filepath.Join(dir, p)); !os.IsNotExist(err) {
			t.Errorf("%s should be deleted", p)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, fresh)); err != nil {
		t.Error("a recent selfie must be kept")
	}
	image := func(id string) string {
		var s string
		pool.QueryRow(ctx, `SELECT image FROM attendance_attempts WHERE id=$1`, id).Scan(&s)
		return s
	}
	if image(idA) != "" || image(idB) != "" || image(idFresh) != fresh || image(idNoImage) != "" {
		t.Fatal("image columns not updated as expected")
	}
	var rows int
	pool.QueryRow(ctx, `SELECT count(*) FROM attendance_attempts`).Scan(&rows)
	if rows != 4 {
		t.Fatalf("attempt rows must be kept, got %d", rows)
	}

	// idempotent, and tolerant of a file that is already gone
	n, err = CleanupSelfies(ctx, db.New(pool), store, time.Now().AddDate(0, 0, -30))
	if err != nil || n != 0 {
		t.Fatalf("second run: %d %v", n, err)
	}
	pool.Exec(ctx, `UPDATE attendance_attempts SET image=$2 WHERE id=$1`, idFresh, fresh)
	os.Remove(filepath.Join(dir, fresh))
	if n, err := CleanupSelfies(ctx, db.New(pool), store, time.Now()); err != nil || n != 1 {
		t.Fatalf("missing file: %d %v", n, err)
	}
	if image(idFresh) != "" {
		t.Fatal("column must be cleared even when the file was already gone")
	}
}
