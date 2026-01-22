package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	contextKeyTenantID contextKey = "tenant_id"
	contextKeyUser     contextKey = "user"
	contextKeyClaims   contextKey = "claims"
)

// UserClaims represents the claims in the JWT token
type UserClaims struct {
	User         string `json:"user"`
	LoginSession string `json:"login_session"`
	TenantID     string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"missing authorization header"}`))
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid authorization header format"}`))
				return
			}

			tokenString := parts[1]
			claims, err := ValidateToken(tokenString, jwtSecret)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid token"}`))
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), contextKeyTenantID, claims.TenantID)
			ctx = context.WithValue(ctx, contextKeyUser, claims.User)
			ctx = context.WithValue(ctx, contextKeyClaims, claims)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString, secret string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method to prevent algorithm confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		if token.Method.Alg() != "HS256" {
			return nil, fmt.Errorf("expected HS256 signing method, got %s", token.Method.Alg())
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}

// GetTenantID retrieves the tenant ID from context
func GetTenantID(ctx context.Context) string {
	if v := ctx.Value(contextKeyTenantID); v != nil {
		return v.(string)
	}
	return ""
}

// GetUser retrieves the user from context
func GetUser(ctx context.Context) string {
	if v := ctx.Value(contextKeyUser); v != nil {
		return v.(string)
	}
	return ""
}

// GetUserClaims retrieves the full claims from context
func GetUserClaims(ctx context.Context) *UserClaims {
	if v := ctx.Value(contextKeyClaims); v != nil {
		return v.(*UserClaims)
	}
	return nil
}
