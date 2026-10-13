package auth

import (
	"crypto/rand"
	"errors"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

var bcryptCost = 12

// UseFastHashingForTests lowers the bcrypt cost so test suites do not spend minutes hashing.
// Never call it from non-test code.
func UseFastHashingForTests() { bcryptCost = bcrypt.MinCost }

// passwordAlphabet omits ambiguous characters: 0 O o l 1 I.
const passwordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"

const GeneratedPasswordLength = 12

func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	return string(b), err
}

func CheckPassword(hash, plain string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// GeneratePassword returns a random 12-character password from crypto/rand.
func GeneratePassword() (string, error) {
	out := make([]byte, GeneratedPasswordLength)
	max := big.NewInt(int64(len(passwordAlphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = passwordAlphabet[n.Int64()]
	}
	return string(out), nil
}

var ErrWeakPassword = errors.New("password too short")

// ValidateNewPassword enforces the minimal policy for user-chosen passwords.
func ValidateNewPassword(p string) error {
	if len(p) < 8 {
		return ErrWeakPassword
	}
	return nil
}
