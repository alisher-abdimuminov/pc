// Package audit writes audit_logs entries inside the caller's transaction.
package audit

import (
	"context"
	"encoding/json"
	"net/netip"

	"github.com/google/uuid"

	"pc/internal/db"
)

const (
	ActionCreate            = "create"
	ActionUpdate            = "update"
	ActionDelete            = "delete"
	ActionLogin             = "login"
	ActionLoginFailed       = "login_failed"
	ActionLogout            = "logout"
	ActionViewSensitive     = "view_sensitive"
	ActionExport            = "export"
	ActionGrade             = "grade"
	ActionAttendanceAttempt = "attendance_attempt"
)

// sensitiveKeys never reach the audit log.
var sensitiveKeys = map[string]bool{
	"password_hash":   true,
	"face_embedding":  true,
	"passport_pin":    true,
	"passport_number": true,
}

type Entry struct {
	ActorID    *uuid.UUID
	ActorRole  string
	Action     string
	EntityType string
	EntityID   *uuid.UUID
	// Before/After are structs or maps; only changed, non-sensitive fields are stored.
	Before    any
	After     any
	IP        string
	UserAgent string
}

// Write inserts the entry using tx (a pgx.Tx, pool or *db.Queries-compatible DBTX).
func Write(ctx context.Context, tx db.DBTX, e Entry) error {
	changes, err := Changes(e.Before, e.After)
	if err != nil {
		return err
	}
	p := db.CreateAuditLogParams{
		ActorRole:  e.ActorRole,
		Action:     e.Action,
		EntityType: e.EntityType,
		Changes:    changes,
		UserAgent:  e.UserAgent,
	}
	if e.ActorID != nil {
		p.ActorID = uuid.NullUUID{UUID: *e.ActorID, Valid: true}
	}
	if e.EntityID != nil {
		p.EntityID = uuid.NullUUID{UUID: *e.EntityID, Valid: true}
	}
	if addr, err := netip.ParseAddr(e.IP); err == nil {
		p.Ip = &addr
	}
	_, err = db.New(tx).CreateAuditLog(ctx, p)
	return err
}

// Changes builds the JSONB payload {"before": {...}, "after": {...}}.
// When both sides exist only fields whose value differs are kept.
func Changes(before, after any) ([]byte, error) {
	b, err := toMap(before)
	if err != nil {
		return nil, err
	}
	a, err := toMap(after)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if b != nil && a != nil {
		bd, ad := map[string]any{}, map[string]any{}
		for k, av := range a {
			if bv, ok := b[k]; !ok || !equal(bv, av) {
				ad[k] = av
				if ok {
					bd[k] = bv
				}
			}
		}
		for k, bv := range b {
			if _, ok := a[k]; !ok {
				bd[k] = bv
			}
		}
		out["before"], out["after"] = bd, ad
	} else {
		if b != nil {
			out["before"] = b
		}
		if a != nil {
			out["after"] = a
		}
	}
	return json.Marshal(out)
}

func toMap(v any) (map[string]any, error) {
	if v == nil {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	for k := range m {
		if sensitiveKeys[k] {
			delete(m, k)
		}
	}
	return m, nil
}

func equal(a, b any) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return string(ja) == string(jb)
}

// HasChanges reports whether before and after differ in any field that would be recorded
// (sensitive fields and timestamps that always move are ignored).
func HasChanges(before, after any) bool {
	b, err1 := toMap(before)
	a, err2 := toMap(after)
	if err1 != nil || err2 != nil {
		return true
	}
	for k, av := range a {
		if k == "updated_at" || k == "last_login_at" || k == "hemis_synced_at" {
			continue
		}
		if bv, ok := b[k]; !ok || !equal(bv, av) {
			return true
		}
	}
	return false
}
