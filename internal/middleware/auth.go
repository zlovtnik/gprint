package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/zlovtnik/gprint/pkg/auth"
)

type contextKey string

const (
	contextKeyTenantID contextKey = "tenant_id"
	contextKeyUser     contextKey = "user"
	contextKeyClaims   contextKey = "claims"

	// HTTP header constants
	headerContentType = "Content-Type"
	contentTypeJSON   = "application/json"
)

// UserClaims is an alias for auth.Claims for backward compatibility.
// Prefer using auth.Claims directly in new code.
type UserClaims = auth.Claims

// AuthMiddleware validates JWT tokens.
// This is a thin HTTP wrapper that delegates token validation to pkg/auth.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set(headerContentType, contentTypeJSON)
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"missing authorization header"}`))
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				w.Header().Set(headerContentType, contentTypeJSON)
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid authorization header format"}`))
				return
			}

			tokenString := parts[1]
			claims, err := auth.ValidateToken(tokenString, jwtSecret)
			if err != nil {
				w.Header().Set(headerContentType, contentTypeJSON)
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"invalid token"}`))
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

// GetUserID is an alias for GetUser for API consistency
func GetUserID(ctx context.Context) string {
	return GetUser(ctx)
}

// GetUserClaims retrieves the full claims from context
func GetUserClaims(ctx context.Context) *UserClaims {
	if v := ctx.Value(contextKeyClaims); v != nil {
		return v.(*UserClaims)
	}
	return nil
}
