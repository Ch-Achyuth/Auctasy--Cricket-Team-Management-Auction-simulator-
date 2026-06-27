package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ctxKey is an unexported type for context keys in this package.
type ctxKey string

// CtxUserID is the context key under which the authenticated user's UUID is stored.
const CtxUserID ctxKey = "user_id"

// ValidateToken validates a raw JWT string and returns the subject (user UUID).
// Used by the /ws handler which reads the token from a query param rather than
// a header, because the browser WebSocket API cannot send custom headers.
func ValidateToken(tokenStr, jwtSecret string) (string, error) {
	claims, err := parseHS256JWT(tokenStr, []byte(jwtSecret))
	if err != nil {
		return "", err
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", errors.New("missing sub claim")
	}
	return sub, nil
}

// RequireAuth returns middleware that validates a Supabase-issued HS256 JWT.
// On success it injects the caller's UUID into the request context under CtxUserID.
// The secret must be the SUPABASE_JWT_SECRET from the project settings.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
	secret := []byte(jwtSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerToken(r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			claims, err := parseHS256JWT(token, secret)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			sub, _ := claims["sub"].(string)
			if sub == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), CtxUserID, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bearerToken extracts the token string from an "Authorization: Bearer <token>" header.
func bearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return "", errors.New("missing bearer token")
	}
	return strings.TrimPrefix(h, "Bearer "), nil
}

// parseHS256JWT validates a JWT signed with HMAC-SHA256 and returns its payload claims.
// It checks the signature and the exp claim; it does NOT validate iss or aud.
func parseHS256JWT(tokenStr string, secret []byte) (map[string]interface{}, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}

	// Recompute HMAC-SHA256 over "header.payload"
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(signingInput))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	// Constant-time comparison prevents timing attacks
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, errors.New("invalid signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, err
	}

	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return nil, errors.New("token expired")
	}

	return claims, nil
}
