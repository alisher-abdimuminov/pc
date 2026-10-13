// Package permission defines what a "little" admin can be granted. Admins have everything; every other
// role has nothing. A grant is "<resource>:<action>".
package permission

import (
	"fmt"
	"slices"
	"sort"

	"pc/internal/db"
)

const (
	RoleAdmin  = "admin"
	RoleLittle = "little"
)

// Catalog is every grantable permission, in display order.
var Catalog = []string{
	"dashboard:view",
	"faculties:view", "faculties:create", "faculties:update", "faculties:delete",
	"groups:view", "groups:create", "groups:update", "groups:delete",
	"users:view", "users:update",
	"locations:view", "locations:create", "locations:update", "locations:delete",
	"schedules:view", "schedules:create", "schedules:update", "schedules:delete",
	"attempts:view",
	"reports:view",
	"audit:view",
}

// requires lists what a permission needs in order to be usable: the pages behind it load lists from other
// sections (a schedule form picks groups, locations and students). Closure is applied by Normalize.
var requires = map[string][]string{
	"faculties:create": {"faculties:view"}, "faculties:update": {"faculties:view"}, "faculties:delete": {"faculties:view"},
	"groups:view":   {"faculties:view"},
	"groups:create": {"groups:view", "users:view"}, "groups:update": {"groups:view", "users:view"}, // the form picks a teacher
	"groups:delete":    {"groups:view"},
	"users:view":       {"groups:view", "faculties:view"}, // the list filters by group and faculty
	"users:update":     {"users:view"},
	"locations:create": {"locations:view"}, "locations:update": {"locations:view"}, "locations:delete": {"locations:view"},
	"schedules:view":   {"groups:view", "locations:view"},
	"schedules:create": {"schedules:view", "users:view"}, "schedules:update": {"schedules:view", "users:view"}, // the form picks a student
	"schedules:delete": {"schedules:view"},
	"attempts:view":    {"groups:view"},
	"reports:view":     {"groups:view"},
}

// Valid reports whether p is a grantable permission.
func Valid(p string) bool { return slices.Contains(Catalog, p) }

// Requires returns the full (transitive) set of permissions p depends on, without p itself.
func Requires(p string) []string {
	seen := map[string]bool{}
	var walk func(string)
	walk = func(x string) {
		for _, r := range requires[x] {
			if !seen[r] {
				seen[r] = true
				walk(r)
			}
		}
	}
	walk(p)
	out := make([]string, 0, len(seen))
	for r := range seen {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return slices.Index(Catalog, out[i]) < slices.Index(Catalog, out[j]) })
	return out
}

// Normalize validates a requested grant list, adds everything it depends on, removes duplicates and
// returns it in catalog order. An unknown permission is an error.
func Normalize(in []string) ([]string, error) {
	want := map[string]bool{}
	for _, p := range in {
		if !Valid(p) {
			return nil, fmt.Errorf("unknown permission %q", p)
		}
		want[p] = true
		for _, r := range Requires(p) {
			want[r] = true
		}
	}
	out := make([]string, 0, len(want))
	for _, p := range Catalog {
		if want[p] {
			out = append(out, p)
		}
	}
	return out, nil
}

// Has reports whether the user may do p: admins always, a little admin when it was granted.
func Has(u db.User, p string) bool {
	switch u.Role {
	case RoleAdmin:
		return true
	case RoleLittle:
		return slices.Contains(u.Permissions, p)
	}
	return false
}

// HasAny is true when the user holds at least one of ps.
func HasAny(u db.User, ps ...string) bool {
	return slices.ContainsFunc(ps, func(p string) bool { return Has(u, p) })
}
