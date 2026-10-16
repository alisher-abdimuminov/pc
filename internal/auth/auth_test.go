package auth

import (
	"strings"
	"testing"
)

func TestGeneratePassword(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		p, err := GeneratePassword()
		if err != nil {
			t.Fatal(err)
		}
		if len(p) != 12 {
			t.Fatalf("len=%d", len(p))
		}
		if strings.ContainsAny(p, "0OolI1") {
			t.Fatalf("ambiguous char in %q", p)
		}
		seen[p] = true
	}
	if len(seen) < 199 {
		t.Fatal("passwords are not random")
	}
}

func TestHashAndCheck(t *testing.T) {
	h, err := HashPassword("s3cret-pass")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(h, "s3cret-pass") || CheckPassword(h, "wrong") || CheckPassword("", "x") {
		t.Fatal("password check misbehaves")
	}
}

func TestTokenHash(t *testing.T) {
	raw, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 43 || hash != HashToken(raw) || hash == raw || len(hash) != 64 {
		t.Fatalf("bad token: %q %q", raw, hash)
	}
}
