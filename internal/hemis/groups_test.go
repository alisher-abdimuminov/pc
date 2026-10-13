package hemis

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// pagedServer serves `total` groups (ids 1..total) in pages of the requested limit.
func pagedServer(t *testing.T, total int, token string, hits *atomic.Int32) *GroupsHTTP {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			hits.Add(1)
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(401)
			w.Write([]byte(`{"success":false}`))
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if limit != 200 || page < 1 {
			t.Errorf("bad paging params: limit=%d page=%d", limit, page)
		}
		pages := (total + limit - 1) / limit
		var items []string
		for id := (page-1)*limit + 1; id <= total && id <= page*limit; id++ {
			items = append(items, fmt.Sprintf(`{"id":%d,"name":"  Guruh-%d "}`, id, id))
		}
		fmt.Fprintf(w, `{"success":true,"data":{"items":[%s],"pagination":{"pageCount":%d}}}`, strings.Join(items, ","), pages)
	}))
	t.Cleanup(srv.Close)
	return NewGroups(srv.URL+"/rest/v1/data/group-list", token, 3*time.Second)
}

func TestListGroupsWalksEveryPage(t *testing.T) {
	var hits atomic.Int32
	c := pagedServer(t, 450, "tok", &hits) // 3 pages of 200
	groups, err := c.ListGroups(context.Background())
	if err != nil || len(groups) != 450 || hits.Load() != 3 {
		t.Fatalf("%d groups, %d requests, %v", len(groups), hits.Load(), err)
	}
	if groups[0] != (Group{ID: 1, Name: "Guruh-1"}) || groups[449].ID != 450 { // names are trimmed
		t.Fatalf("%+v ... %+v", groups[0], groups[449])
	}
}

func TestListGroupsSinglePageAndEmpty(t *testing.T) {
	var hits atomic.Int32
	if g, err := pagedServer(t, 5, "tok", &hits).ListGroups(context.Background()); err != nil || len(g) != 5 || hits.Load() != 1 {
		t.Fatalf("%d %d %v", len(g), hits.Load(), err)
	}
	if g, err := pagedServer(t, 0, "tok", nil).ListGroups(context.Background()); err != nil || len(g) != 0 {
		t.Fatalf("empty list: %d %v", len(g), err)
	}
}

func TestListGroupsSkipsBadEntriesAndDeduplicates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"data":{"items":[
			{"id":1,"name":"A"},{"id":0,"name":"no id"},{"id":2,"name":"   "},{"name":"no id at all"},
			{"id":"3","name":"C (string id)"},{"id":1,"name":"A renamed"},{"id":4}
		],"pagination":{"pageCount":1}}}`))
	}))
	defer srv.Close()
	groups, err := NewGroups(srv.URL, "tok", time.Second).ListGroups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0] != (Group{ID: 1, Name: "A renamed"}) || groups[1] != (Group{ID: 3, Name: "C (string id)"}) {
		t.Fatalf("%+v", groups)
	}
}

func TestListGroupsErrors(t *testing.T) {
	if _, err := NewGroups("http://x", "  ", time.Second).ListGroups(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("no token: %v", err)
	}
	reply := func(status int, body string) *GroupsHTTP {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); w.Write([]byte(body)) }))
		t.Cleanup(srv.Close)
		return NewGroups(srv.URL, "secret-token", time.Second)
	}
	for name, c := range map[string]*GroupsHTTP{
		"401":           reply(401, `{"success":false,"error":"token secret-token rejected"}`),
		"500":           reply(500, `internal secret-token details`),
		"success=false": reply(200, `{"success":false,"error":"nope"}`),
		"garbage":       reply(200, `<html>`),
	} {
		_, err := c.ListGroups(context.Background())
		if !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: %v", name, err)
		} else if strings.Contains(err.Error(), "secret-token") || strings.Contains(err.Error(), "details") {
			t.Errorf("%s: error leaks upstream text or the token: %v", name, err)
		}
	}
	// an error on a later page fails the whole listing: no partial results
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 2 {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`{"success":true,"data":{"items":[{"id":1,"name":"A"}],"pagination":{"pageCount":3}}}`))
	}))
	defer srv.Close()
	if g, err := NewGroups(srv.URL, "t", time.Second).ListGroups(context.Background()); err == nil || g != nil {
		t.Fatalf("partial result: %v %v", g, err)
	}
}

func TestListGroupsStopsAtThePageCap(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		w.Write([]byte(`{"success":true,"data":{"items":[{"id":1,"name":"A"}],"pagination":{"pageCount":999999}}}`)) // never ends
	}))
	defer srv.Close()
	if _, err := NewGroups(srv.URL, "t", time.Second).ListGroups(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n.Load() != groupsMaxPages {
		t.Fatalf("%d requests, want the cap of %d", n.Load(), groupsMaxPages)
	}
}

func TestListGroupsTimeoutAndOutage(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(300 * time.Millisecond) }))
	defer slow.Close()
	if _, err := NewGroups(slow.URL, "t", 50*time.Millisecond).ListGroups(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("timeout: %v", err)
	}
	dead := httptest.NewServer(http.NotFoundHandler())
	url := dead.URL
	dead.Close()
	if _, err := NewGroups(url, "t", time.Second).ListGroups(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("down: %v", err)
	}
}

func TestFakeGroups(t *testing.T) {
	f := NewFakeGroups(Group{ID: 1, Name: "A"})
	g, err := f.ListGroups(context.Background())
	if err != nil || len(g) != 1 || f.Calls() != 1 {
		t.Fatal("fake")
	}
	f.SetErr(ErrUnavailable)
	if _, err := f.ListGroups(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatal("fake error")
	}
}

func TestListFacultiesKeepsOnlyFaculties(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(401)
			return
		}
		page := r.URL.Query().Get("page")
		if page == "1" {
			w.Write([]byte(`{"success":true,"data":{"items":[
				{"id":59,"name":" Urgut fakulteti ","structureType":{"code":"11","name":"Fakultet"}},
				{"id":64,"name":"Kafedra","structureType":{"code":"12","name":"Kafedra"}},
				{"id":900,"name":"A group, not a department"}],"pagination":{"pageCount":2}}}`))
			return
		}
		w.Write([]byte(`{"success":true,"data":{"items":[{"id":2,"name":"Tillar","structureType":{"code":"11"}}],"pagination":{"pageCount":2}}}`))
	}))
	defer srv.Close()
	got, err := NewFaculties(srv.URL, "tok", 3*time.Second).ListFaculties(context.Background())
	if err != nil || len(got) != 2 || got[0] != (Faculty{ID: 59, Name: "Urgut fakulteti"}) || got[1].ID != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := NewFaculties(srv.URL, "", time.Second).ListFaculties(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("no token: %v", err)
	}
	if _, err := NewFaculties(srv.URL, "bad", time.Second).ListFaculties(context.Background()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("bad token: %v", err)
	}
}

func TestListGroupsReadsTheDepartment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success":true,"data":{"items":[
			{"id":1,"name":"A","department":{"id":59,"name":" Urgut fakulteti "}},
			{"id":2,"name":"B","department":null},
			{"id":3,"name":"C"}],"pagination":{"pageCount":1}}}`))
	}))
	defer srv.Close()
	got, err := NewGroups(srv.URL, "tok", time.Second).ListGroups(context.Background())
	if err != nil || len(got) != 3 {
		t.Fatalf("%+v %v", got, err)
	}
	if got[0].FacultyID != 59 || got[0].FacultyName != "Urgut fakulteti" || got[1].FacultyID != 0 || got[2].FacultyID != 0 {
		t.Fatalf("%+v", got)
	}
}
