package http_test

import (
	"os"
	"testing"

	"pc/internal/auth"
)

func TestMain(m *testing.M) {
	auth.UseFastHashingForTests()
	os.Exit(m.Run())
}
