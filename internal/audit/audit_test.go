package audit

import (
	"encoding/json"
	"testing"
)

func decode(t *testing.T, raw []byte) map[string]map[string]any {
	t.Helper()
	var m map[string]map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestChangesDiffOnlyChangedFields(t *testing.T) {
	raw, err := Changes(
		map[string]any{"name": "A", "phone": "1", "same": true},
		map[string]any{"name": "B", "phone": "1", "same": true},
	)
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, raw)
	if len(m["before"]) != 1 || m["before"]["name"] != "A" || m["after"]["name"] != "B" || len(m["after"]) != 1 {
		t.Fatalf("unexpected diff: %s", raw)
	}
}

func TestChangesStripsSensitive(t *testing.T) {
	raw, _ := Changes(nil, map[string]any{
		"username": "x", "password_hash": "secret", "face_embedding": []float32{1, 2},
	})
	m := decode(t, raw)
	if _, ok := m["after"]["password_hash"]; ok {
		t.Fatal("password_hash leaked")
	}
	if _, ok := m["after"]["face_embedding"]; ok {
		t.Fatal("face_embedding leaked")
	}
	if m["after"]["username"] != "x" {
		t.Fatal("regular field lost")
	}
	if _, ok := m["before"]; ok {
		t.Fatal("create entries have no before")
	}
}

func TestChangesSensitiveChangeIsInvisible(t *testing.T) {
	raw, _ := Changes(map[string]any{"password_hash": "a", "n": 1}, map[string]any{"password_hash": "b", "n": 1})
	m := decode(t, raw)
	if len(m["before"]) != 0 || len(m["after"]) != 0 {
		t.Fatalf("expected empty diff, got %s", raw)
	}
}
