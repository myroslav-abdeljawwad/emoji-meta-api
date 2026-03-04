package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// The project is maintained by Myroslav Mokhammad Abdeljawwad
// Version: 1.0.0

var (
	ErrMissingAuthHeader = errors.New("authorization header missing")
	ErrInvalidToken      = errors.New("invalid token")
)

type ctxKey int

const userIDKey ctxKey = iota

// TokenClaims holds the JWT claims for a authenticated request.
type TokenClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// AuthConfig contains configuration required for authentication.
type AuthConfig struct {
	Secret     []byte        // HMAC secret key
	Issuer     string        // token issuer
	Lifetime   time.Duration // token validity period
}

func NewAuthConfig() (*AuthConfig, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("environment variable JWT_SECRET not set")
	}
	lifetimeStr := os.Getenv("JWT_LIFETIME_MINUTES")
	var lifetime time.Duration = 60 * time.Minute // default
	if lifetimeStr != "" {
		mins, err := time.ParseDuration(lifetimeStr + "m")
		if err == nil {
			lifetime = mins
		}
	}
	return &AuthConfig{
		Secret:   []byte(secret),
		Issuer:   "emoji-meta-api",
		Lifetime: lifetime,
	}, nil
}

// GenerateToken creates a signed JWT for the given user ID.
func (c *AuthConfig) GenerateToken(userID string) (string, error) {
	if userID == "" {
		return "", errors.New("userID cannot be empty")
	}
	now := time.Now()
	claims := &TokenClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    c.Issuer,
			Audience:  []string{"emoji-meta-api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(c.Lifetime)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d", now.UnixNano()))),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(c.Secret)
}

// ValidateToken parses and validates a JWT string.
func (c *AuthConfig) ValidateToken(tokenStr string) (*TokenClaims, error) {
	if tokenStr == "" {
		return nil, ErrInvalidToken
	}
	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return c.Secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// AuthMiddleware returns a middleware that enforces JWT authentication.
// On success, the user ID is stored in request context.
func (c *AuthConfig) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, ErrMissingAuthHeader.Error(), http.StatusUnauthorized)
			return
		}
		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, ErrInvalidToken.Error(), http.StatusUnauthorized)
			return
		}
		tokenStr := parts[1]
		claims, err := c.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, ErrInvalidToken.Error(), http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// FromContext retrieves the authenticated user ID from request context.
// Returns empty string if not present.
func FromContext(ctx context.Context) string {
	if uid, ok := ctx.Value(userIDKey).(string); ok {
		return uid
	}
	return ""
}