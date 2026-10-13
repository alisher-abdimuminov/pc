package hemis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Group is one entry of the HEMIS group list.
type Group struct {
	ID   int64
	Name string
	// Faculty is the group's HEMIS department; zero when HEMIS did not send one.
	FacultyID   int64
	FacultyName string
}

// GroupSource lists every group known to HEMIS (mocked in tests).
type GroupSource interface {
	ListGroups(ctx context.Context) ([]Group, error)
}

const (
	groupsPageSize = 200
	// groupsMaxPages guards against a server that never reports the last page.
	groupsMaxPages = 500
)

// ErrNotConfigured is returned when no groups token is set.
var ErrNotConfigured = errors.New("hemis: groups token is not configured")

// GroupsHTTP reads the paginated group list:
//
//	GET {url}?limit=200&page=N   Authorization: Bearer <token>
//	-> {"success": true, "data": {"items": [{"id": 1, "name": "..."}], "pagination": {"pageCount": 3}}}
type GroupsHTTP struct {
	url   string
	token string
	http  *http.Client
}

func NewGroups(listURL, token string, timeout time.Duration) *GroupsHTTP {
	return &GroupsHTTP{url: listURL, token: token, http: &http.Client{Timeout: timeout}}
}

type groupsPage struct {
	Success bool `json:"success"`
	Error   any  `json:"error"`
	Data    struct {
		Items []struct {
			ID   flexID `json:"id"`
			Name flex   `json:"name"`
			Dept struct {
				ID   flexID `json:"id"`
				Name flex   `json:"name"`
			} `json:"department"`
		} `json:"items"`
		Pagination struct {
			PageCount flexID `json:"pageCount"`
		} `json:"pagination"`
	} `json:"data"`
}

func (c *GroupsHTTP) fetch(ctx context.Context, page int) (groupsPage, error) {
	var out groupsPage
	err := getPage(ctx, c.http, c.url, c.token, page, &out)
	if err == nil && !out.Success {
		err = fmt.Errorf("%w: success=false", ErrUnavailable)
	}
	return out, err
}

// getPage GETs one page of a paginated HEMIS list into out. Neither the URL nor the token
// ever reach an error message.
func getPage(ctx context.Context, client *http.Client, rawURL, token string, page int, out any) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("%w: bad list url", ErrUnavailable)
	}
	q := u.Query()
	q.Set("limit", strconv.Itoa(groupsPageSize))
	q.Set("page", strconv.Itoa(page))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return fmt.Errorf("%w: request failed", ErrUnavailable)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrUnavailable, resp.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%w: unreadable response", ErrUnavailable)
	}
	return nil
}

// ListGroups walks every page and returns the groups, de-duplicated by id (the last entry wins).
// Entries without an id or a name are skipped.
func (c *GroupsHTTP) ListGroups(ctx context.Context) ([]Group, error) {
	if strings.TrimSpace(c.token) == "" {
		return nil, ErrNotConfigured
	}
	byID := map[int64]Group{}
	var order []int64
	for page := 1; page <= groupsMaxPages; page++ {
		p, err := c.fetch(ctx, page)
		if err != nil {
			return nil, err
		}
		for _, it := range p.Data.Items {
			name := strings.TrimSpace(string(it.Name))
			if it.ID == 0 || name == "" {
				continue
			}
			if _, seen := byID[int64(it.ID)]; !seen {
				order = append(order, int64(it.ID))
			}
			g := Group{ID: int64(it.ID), Name: name}
			if fn := strings.TrimSpace(string(it.Dept.Name)); it.Dept.ID != 0 && fn != "" {
				g.FacultyID, g.FacultyName = int64(it.Dept.ID), fn
			}
			byID[int64(it.ID)] = g
		}
		pages := int(p.Data.Pagination.PageCount)
		if pages <= 1 || page >= pages || len(p.Data.Items) == 0 {
			break
		}
	}
	out := make([]Group, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out, nil
}

// FakeGroups is an in-memory GroupSource for tests and local development.
type FakeGroups struct {
	mu     sync.Mutex
	groups []Group
	err    error
	calls  int
}

func NewFakeGroups(groups ...Group) *FakeGroups { return &FakeGroups{groups: groups} }

func (f *FakeGroups) Set(groups ...Group) { f.mu.Lock(); f.groups = groups; f.mu.Unlock() }
func (f *FakeGroups) SetErr(err error)    { f.mu.Lock(); f.err = err; f.mu.Unlock() }
func (f *FakeGroups) Calls() int          { f.mu.Lock(); defer f.mu.Unlock(); return f.calls }

func (f *FakeGroups) ListGroups(context.Context) ([]Group, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return append([]Group(nil), f.groups...), nil
}
