package auth

// bcrypt provides a deliberately expensive password hash and verifier.
import "golang.org/x/crypto/bcrypt"

// bcryptCost controls how slow hashing is. 12 takes roughly a quarter
// of a second on a typical machine.
const bcryptCost = 12

// dummyHash is a real bcrypt hash of a throwaway password, made once at startup.
var dummyHash = mustHash("dummy-password-for-timing")

func mustHash(password string) []byte {
	// Hash the supplied password using the package's chosen work factor.
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		// This helper is used during package initialization, so hashing failure
		// prevents startup instead of leaving timing protection unavailable.
		panic(err)
	}
	// Return the bcrypt-encoded password bytes.
	return h
}

// HashPassword converts a plaintext password into a bcrypt hash for storage.
func HashPassword(password string) (string, error) {
	// bcrypt accepts bytes and applies the configured computational cost.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		// Pass hashing failures to the caller for proper handling.
		return "", err
	}
	// Store the encoded hash as a string, never the plaintext password.
	return string(hash), nil
}

// CheckPassword reports whether password matches the stored hash.
func CheckPassword(hash, password string) bool {
	// bcrypt safely compares a candidate password against its encoded hash.
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// BurnTime spends the same time a real password check would. We call it
// when the email doesn't exist, so an attacker can't tell "no such user"
// from "wrong password" by measuring how fast the server answers.
func BurnTime(password string) {
	// Compare against a fixed valid hash so nonexistent accounts take similar
	// time to process as accounts with an incorrect password.
	bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}
