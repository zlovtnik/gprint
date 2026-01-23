// Package auth provides reusable JWT utilities with no HTTP dependencies.
// This package is designed to be used by middleware and other services
// that need to validate and parse JWT tokens.
package auth

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the claims in the JWT token.
// This is the standard claims structure used across the application.
type Claims struct {
	User         string `json:"user"`
	LoginSession string `json:"login_session"`
	TenantID     string `json:"tenant_id"`
	jwt.RegisteredClaims
}

// ParseToken parses a JWT token string without validating the signature.
// Use this only when you need to inspect claims before validation.
// For secure validation, use ValidateToken instead.
func ParseToken(tokenString string) (*jwt.Token, error) {
	return jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Return nil key - this is for parsing only, not validation
		return nil, nil
	}, jwt.WithoutClaimsValidation())
}

// ValidateToken validates a JWT token string and returns the claims if valid.
// It verifies the signature using the provided secret and ensures the token
// uses the expected HS256 signing method.
func ValidateToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
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

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}

// ClaimsFromToken extracts claims from an already-parsed token.
// Returns nil if the token is nil or claims cannot be extracted.
func ClaimsFromToken(token *jwt.Token) *Claims {
	if token == nil {
		return nil
	}
	if claims, ok := token.Claims.(*Claims); ok {
		return claims
	}
	return nil
}
