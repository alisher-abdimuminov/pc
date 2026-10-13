package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/audit"
	"pc/internal/config"
	"pc/internal/db"
	"pc/internal/demo"
	"pc/internal/service"
	"pc/internal/storage"
)

var systemActor = service.Actor{Role: "system"}

// createAdmin creates an admin with a random password and prints it once.
func createAdmin(ctx context.Context, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("createadmin", flag.ContinueOnError)
	username := fs.String("username", "", "admin username")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *username == "" {
		return errors.New("--username is required")
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	u, password, err := service.NewUserService(pool).CreateUser(ctx, systemActor, service.CreateUserInput{
		Username: *username, Role: "admin", FullName: *username, MustChangePassword: true,
	})
	if err != nil {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			return errors.New(ae.Message)
		}
		return err
	}
	fmt.Printf("Admin created.\n  username: %s\n  password: %s\n(shown only once; it must be changed at first login)\n", u.Username.String, password)
	return nil
}

// seed creates a test faculty, group, teacher and students (idempotent).
func seed(ctx context.Context, cfg *config.Config) error {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	q := db.New(pool)
	users := service.NewUserService(pool)

	var faculty db.Faculty
	var group db.Group
	err = service.InTx(ctx, pool, func(tx pgx.Tx, q *db.Queries) error {
		faculties, err := q.ListFaculties(ctx, db.ListFacultiesParams{Search: "Test fakultet", Lim: 1})
		if err != nil {
			return err
		}
		if len(faculties) > 0 {
			faculty = faculties[0]
		} else {
			if faculty, err = q.CreateFaculty(ctx, db.CreateFacultyParams{Name: "Test fakultet"}); err != nil {
				return err
			}
			if err := audit.Write(ctx, tx, audit.Entry{ActorRole: "system", Action: audit.ActionCreate, EntityType: "faculty", EntityID: &faculty.ID, After: faculty}); err != nil {
				return err
			}
		}
		groups, err := q.ListGroups(ctx, db.ListGroupsParams{Search: "TEST-101", Lim: 1})
		if err != nil {
			return err
		}
		if len(groups) > 0 {
			group = groups[0]
			return nil
		}
		group, err = q.CreateGroup(ctx, db.CreateGroupParams{Name: "TEST-101", FacultyID: uuid.NullUUID{UUID: faculty.ID, Valid: true}})
		if err != nil {
			return err
		}
		return audit.Write(ctx, tx, audit.Entry{ActorRole: "system", Action: audit.ActionCreate, EntityType: "group", EntityID: &group.ID, After: group})
	})
	if err != nil {
		return err
	}

	teacher, err := ensureUser(ctx, q, users, service.CreateUserInput{
		Username: "teacher1", Role: "teacher", FullName: "Test O'qituvchi", MustChangePassword: true,
	})
	if err != nil {
		return err
	}
	if !group.TeacherID.Valid {
		g, err := q.UpdateGroup(ctx, db.UpdateGroupParams{
			ID: group.ID, Name: group.Name, FacultyID: group.FacultyID,
			TeacherID: uuid.NullUUID{UUID: teacher.ID, Valid: true},
		})
		if err != nil {
			return err
		}
		group = g
	}
	for i := 1; i <= 3; i++ {
		if _, err := ensureUser(ctx, q, users, service.CreateUserInput{
			Username: fmt.Sprintf("student%d", i), Role: "student", FullName: fmt.Sprintf("Test Talaba %d", i),
			GroupID: &group.ID, FacultyID: &faculty.ID,
		}); err != nil {
			return err
		}
	}
	fmt.Println("Seed done. (Student password login works only with DEV_STUDENT_PASSWORD_LOGIN=true.)")
	return nil
}

func ensureUser(ctx context.Context, q *db.Queries, users *service.UserService, in service.CreateUserInput) (db.User, error) {
	existing, err := q.GetUserByUsername(ctx, pgtype.Text{String: in.Username, Valid: true})
	if err == nil {
		fmt.Printf("  %-9s %-9s exists\n", in.Role, in.Username)
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, err
	}
	u, password, err := users.CreateUser(ctx, systemActor, in)
	if err != nil {
		return db.User{}, err
	}
	fmt.Printf("  %-9s %-9s password: %s\n", in.Role, in.Username, password)
	return u, nil
}

// resetPassword generates a new random password for an existing user and prints it once.
// By default the user must change it at the next login; --keep lets a test account keep it.
func resetPassword(ctx context.Context, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("resetpassword", flag.ContinueOnError)
	username := fs.String("username", "", "username of the account")
	keep := fs.Bool("keep", false, "do not force a password change at the next login (for test accounts)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *username == "" {
		return errors.New("--username is required")
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	u, err := db.New(pool).GetUserByUsername(ctx, pgtype.Text{String: strings.ToLower(strings.TrimSpace(*username)), Valid: true})
	if err != nil {
		return fmt.Errorf("user %q not found", *username)
	}
	password, err := service.NewUserService(pool).SetGeneratedPassword(ctx, systemActor, u.ID, !*keep)
	if err != nil {
		return err
	}
	note := "must be changed at first login"
	if *keep {
		note = "no forced change"
	}
	fmt.Printf("Password reset.\n  username: %s\n  role:     %s\n  password: %s\n(shown only once; %s)\n", u.Username.String, u.Role, password, note)
	if u.Role == "student" && !cfg.DevStudentPasswordLogin {
		fmt.Println("note: students can log in with a local password only when DEV_STUDENT_PASSWORD_LOGIN=true")
	}
	return nil
}

// seedDemo fills the database with sample attendance attempts for the seeded students.
func seedDemo(ctx context.Context, cfg *config.Config, args []string) error {
	fs := flag.NewFlagSet("seeddemo", flag.ContinueOnError)
	days := fs.Int("days", 10, "how many past days to generate")
	reset := fs.Bool("reset", false, "remove earlier demo data first")
	if err := fs.Parse(args); err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	sum, err := demo.Run(ctx, pool, storage.NewLocal(cfg.UploadDir), demo.Options{Days: *days, Reset: *reset, Loc: cfg.Location})
	if err != nil {
		return err
	}
	if sum.Skipped {
		fmt.Println("Demo data already exists (use --reset to regenerate).")
		return nil
	}
	fmt.Printf("Demo data created: %d attendances, %d attempts for student1..student3.\n", sum.Attendances, sum.Attempts)
	return nil
}
