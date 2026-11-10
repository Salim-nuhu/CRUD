package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid or expired token")

// claims is the data inside our tokens: the standard fields (subject,
// expiry, issued-at) plus our own "role" field.
type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl}
}

func (m *TokenManager) TTL() time.Duration { return m.ttl }

// Generate creates a signed token carrying the user's id and role.
func (m *TokenManager) Generate(userID, role string) (string, error) {
	if userID == "" || role == "" {
		return "", errors.New("token needs both a user id and a role")
	}

	now := time.Now()
	c := claims{
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

// Parse checks the signature and expiry, and returns who the token is for.
func (m *TokenManager) Parse(tokenString string) (Identity, error) {
	var c claims

	token, err := jwt.ParseWithClaims(tokenString, &c,
		func(t *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
	)
	if err != nil || !token.Valid || c.Subject == "" || c.Role == "" {
		return Identity{}, ErrInvalidToken
	}
	return Identity{UserID: c.Subject, Role: c.Role}, nil
}
