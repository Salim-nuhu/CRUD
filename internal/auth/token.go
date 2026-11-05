package auth

import (
	// errors defines the stable error returned for rejected tokens.
	"errors"
	// time represents token lifetimes and their expiration timestamps.
	"time"

	// jwt creates, signs, and validates JSON Web Tokens.
	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken hides the specific reason a token failed validation.
var ErrInvalidToken = errors.New("invalid or expired token")

// TokenManager holds the signing key and default lifetime for issued tokens.
type TokenManager struct {
	// secret is the HMAC signing key stored as bytes for the JWT library.
	secret []byte
	// ttl is how long generated tokens remain valid after they are issued.
	ttl time.Duration
}

// NewTokenManager creates a manager with the provided secret and token lifetime.
func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	// Convert the secret into the byte form expected by the signing library.
	return &TokenManager{secret: []byte(secret), ttl: ttl}
}

// TTL returns the configured token lifetime.
func (m *TokenManager) TTL() time.Duration { return m.ttl }

// Generate creates a signed token whose subject is the user's id.
func (m *TokenManager) Generate(userID string) (string, error) {
	// Capture one timestamp so the issue and expiry times agree.
	now := time.Now()
	// Put the user's ID and the token's validity window in standard JWT claims.
	claims := jwt.RegisteredClaims{
		// Subject identifies the user represented by this token.
		Subject: userID,
		// IssuedAt records when this token was created.
		IssuedAt: jwt.NewNumericDate(now),
		// ExpiresAt prevents use after the configured lifetime.
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
	}
	// Sign the claims with HMAC-SHA256 and return the compact token string.
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse checks the signature and expiry, and returns the user id inside.
// We'll call this from the auth middleware in Step 8.
func (m *TokenManager) Parse(tokenString string) (string, error) {
	// Decode standard registered claims so the subject and expiry can be checked.
	var claims jwt.RegisteredClaims

	// Verify the signature and registered claims using this manager's secret.
	token, err := jwt.ParseWithClaims(tokenString, &claims,
		// Provide the HMAC secret only after checking the signing algorithm below.
		func(t *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}), // only accept the algorithm we use
		jwt.WithExpirationRequired(),            // a token with no expiry is rejected
	)
	// Reject parse errors, invalid signatures/claims, and tokens without a subject.
	if err != nil || !token.Valid || claims.Subject == "" {
		return "", ErrInvalidToken
	}
	// Return the validated user ID for the caller to use.
	return claims.Subject, nil
}
