package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Tokens issues and verifies signed JWT access tokens.
type Tokens struct {
	secret []byte
	ttl    time.Duration
	issuer string
	now    func() time.Time
}

// NewTokens returns a token issuer. The secret must be at least 32 bytes.
func NewTokens(secret string, ttl time.Duration) (*Tokens, error) {
	if len(secret) < 32 {
		return nil, errors.New("jwt secret must be at least 32 characters")
	}
	return &Tokens{secret: []byte(secret), ttl: ttl, issuer: "estates", now: time.Now}, nil
}

type claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// Issue returns a signed token for the user and its expiry time.
func (t *Tokens) Issue(userID, email string) (string, time.Time, error) {
	now := t.now()
	exp := now.Add(t.ttl)
	c := claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    t.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, exp, nil
}

// ErrInvalidToken is returned for tokens that are malformed, expired or
// signed with a different key.
var ErrInvalidToken = errors.New("invalid or expired token")

// Verify checks the token and returns the user ID it was issued for.
func (t *Tokens) Verify(token string) (string, error) {
	var c claims
	parsed, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) {
		return t.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(t.issuer),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil || !parsed.Valid || c.Subject == "" {
		return "", ErrInvalidToken
	}
	return c.Subject, nil
}
