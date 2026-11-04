package auth

import "golang.org/x/crypto/bcrypt"

// bcryptCost controls how slow hashing is. 12 takes roughly a quarter
// of a second on a typical machine: invisible to one user logging in,
// painful for someone trying millions of guesses.
const bcryptCost = 12

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches the stored hash.
// We'll use it in Step 7 for login.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
