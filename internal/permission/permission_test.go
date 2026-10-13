package permission

import (
	"slices"
	"testing"

	"pc/internal/db"
)

func TestNormalizeAddsDependenciesInCatalogOrder(t *testing.T) {
	got, err := Normalize([]string{"schedules:create", "schedules:create", "locations:create"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"faculties:view", "groups:view", "users:view", "locations:view", "locations:create",
		"schedules:view", "schedules:create",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

func TestNormalizeRejectsUnknownAndAcceptsEmpty(t *testing.T) {
	if _, err := Normalize([]string{"users:view", "users:delete"}); err == nil {
		t.Fatal("users:delete is not grantable")
	}
	if _, err := Normalize([]string{"*"}); err == nil {
		t.Fatal("wildcards are not grantable")
	}
	if got, err := Normalize(nil); err != nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
}

func TestEveryDependencyIsGrantable(t *testing.T) {
	for _, p := range Catalog {
		for _, r := range Requires(p) {
			if !Valid(r) {
				t.Errorf("%s requires %s, which is not in the catalog", p, r)
			}
		}
		if slices.Contains(Requires(p), p) {
			t.Errorf("%s requires itself", p)
		}
	}
}

func TestHas(t *testing.T) {
	little := db.User{Role: RoleLittle, Permissions: []string{"users:view"}}
	cases := []struct {
		u    db.User
		p    string
		want bool
	}{
		{db.User{Role: RoleAdmin}, "audit:view", true},
		{little, "users:view", true},
		{little, "users:update", false},
		{db.User{Role: "teacher", Permissions: []string{"users:view"}}, "users:view", false},
		{db.User{Role: "student"}, "users:view", false},
	}
	for _, c := range cases {
		if got := Has(c.u, c.p); got != c.want {
			t.Errorf("%s %s: %v", c.u.Role, c.p, got)
		}
	}
}

func TestGrantsDoNotLeakIntoOtherSections(t *testing.T) {
	for p, forbidden := range map[string][]string{
		"groups:view":    {"users:view", "audit:view"},
		"attempts:view":  {"users:view", "users:update"},
		"reports:view":   {"users:view"},
		"locations:view": {"groups:view"},
	} {
		got, _ := Normalize([]string{p})
		for _, f := range forbidden {
			if slices.Contains(got, f) {
				t.Errorf("%s must not grant %s: %v", p, f, got)
			}
		}
	}
}
