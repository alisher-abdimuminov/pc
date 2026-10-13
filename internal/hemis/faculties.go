package hemis

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Faculty is one faculty of the HEMIS department list.
type Faculty struct {
	ID   int64
	Name string
}

// FacultySource lists every faculty known to HEMIS (mocked in tests).
type FacultySource interface {
	ListFaculties(ctx context.Context) ([]Faculty, error)
}

// facultyStructureCode is HEMIS' structureType.code for "Fakultet" (departments, chairs etc. have other codes).
const facultyStructureCode = "11"

// FacultiesHTTP reads the paginated department list (data/department-list) and keeps the faculties:
//
//	{"data": {"items": [{"id": 59, "name": "...", "structureType": {"code": "11", "name": "Fakultet"}}], ...}}
//
// It shares the groups token. Entries without a structureType are skipped, so pointing the URL at the
// group list by mistake yields nothing instead of turning every group into a faculty.
type FacultiesHTTP struct {
	groups *GroupsHTTP
}

func NewFaculties(listURL, token string, timeout time.Duration) *FacultiesHTTP {
	return &FacultiesHTTP{groups: NewGroups(listURL, token, timeout)}
}

type facultiesPage struct {
	Success bool `json:"success"`
	Data    struct {
		Items []struct {
			ID            flexID `json:"id"`
			Name          flex   `json:"name"`
			StructureType struct {
				Code flex `json:"code"`
			} `json:"structureType"`
		} `json:"items"`
		Pagination struct {
			PageCount flexID `json:"pageCount"`
		} `json:"pagination"`
	} `json:"data"`
}

// ListFaculties walks every page and returns the faculties, de-duplicated by id (the last entry wins).
func (c *FacultiesHTTP) ListFaculties(ctx context.Context) ([]Faculty, error) {
	g := c.groups
	if strings.TrimSpace(g.token) == "" {
		return nil, ErrNotConfigured
	}
	byID := map[int64]Faculty{}
	var order []int64
	for page := 1; page <= groupsMaxPages; page++ {
		var p facultiesPage
		if err := getPage(ctx, g.http, g.url, g.token, page, &p); err != nil {
			return nil, err
		}
		if !p.Success {
			return nil, fmt.Errorf("%w: success=false", ErrUnavailable)
		}
		for _, it := range p.Data.Items {
			name := strings.TrimSpace(string(it.Name))
			if it.ID == 0 || name == "" || string(it.StructureType.Code) != facultyStructureCode {
				continue
			}
			if _, seen := byID[int64(it.ID)]; !seen {
				order = append(order, int64(it.ID))
			}
			byID[int64(it.ID)] = Faculty{ID: int64(it.ID), Name: name}
		}
		pages := int(p.Data.Pagination.PageCount)
		if pages <= 1 || page >= pages || len(p.Data.Items) == 0 {
			break
		}
	}
	out := make([]Faculty, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out, nil
}

// FakeFaculties is an in-memory FacultySource for tests and local development.
type FakeFaculties struct {
	mu        sync.Mutex
	faculties []Faculty
	err       error
}

func NewFakeFaculties(f ...Faculty) *FakeFaculties { return &FakeFaculties{faculties: f} }

func (f *FakeFaculties) Set(v ...Faculty) { f.mu.Lock(); f.faculties = v; f.mu.Unlock() }
func (f *FakeFaculties) SetErr(err error) { f.mu.Lock(); f.err = err; f.mu.Unlock() }

func (f *FakeFaculties) ListFaculties(context.Context) ([]Faculty, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	return append([]Faculty(nil), f.faculties...), nil
}
