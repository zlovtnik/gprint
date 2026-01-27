package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zlovtnik/gprint/internal/models"
	"github.com/zlovtnik/gprint/pkg/auth"
)

// internalTokenTTL is the time-to-live for internally minted JWT tokens
const internalTokenTTL = 24 * time.Hour

// minJWTSecretLen is the minimum required length for JWT secrets (32 bytes for HMAC-SHA256)
const minJWTSecretLen = 32

// logoutMessage is the standard response message for logout operations
const logoutMessage = "logged out"

// msgInvalidRequestBody is the standard error message for invalid request bodies
const msgInvalidRequestBody = "invalid request body"

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	keycloak  *auth.KeycloakClient
	jwtSecret string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(keycloak *auth.KeycloakClient, jwtSecret string) *AuthHandler {
	if keycloak == nil {
		panic("keycloak client is required")
	}
	if jwtSecret == "" {
		panic("jwt secret is required")
	}
	if len(jwtSecret) < minJWTSecretLen {
		panic("jwt secret must be at least 32 bytes for HMAC-SHA256 security")
	}
	return &AuthHandler{
		keycloak:  keycloak,
		jwtSecret: jwtSecret,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	User         string `json:"user"`
	TenantID     string `json:"tenant_id,omitempty"`
}

// RefreshRequest represents the refresh token request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutRequest represents the logout request
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Login authenticates a user with Keycloak and returns tokens
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, msgInvalidRequestBody)
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "username and password are required")
		return
	}

	// Authenticate with Keycloak
	tokenResp, err := h.keycloak.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		// Check for specific Keycloak errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "invalid_grant") {
			writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid username or password")
			return
		}
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "authentication failed")
		return
	}

	// Get user info from Keycloak
	userInfo, err := h.keycloak.GetUserInfo(r.Context(), tokenResp.AccessToken)
	if err != nil {
		// Log the error but continue - we can still use token claims
		userInfo = &auth.UserInfo{
			PreferredUsername: req.Username,
		}
	}

	// Create our own JWT that includes tenant_id for multi-tenant isolation
	// The tenant_id would typically come from Keycloak user attributes or a separate lookup
	tenantID := extractTenantID(userInfo)

	// Create internal JWT with required claims
	internalToken, err := h.createInternalToken(userInfo.PreferredUsername, tenantID, tokenResp.SessionState)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, "failed to create session token")
		return
	}

	response := LoginResponse{
		AccessToken:  internalToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    int(internalTokenTTL.Seconds()), // Use internal token TTL, not Keycloak's
		TokenType:    "Bearer",
		User:         userInfo.PreferredUsername,
		TenantID:     tenantID,
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(response))
}

// Refresh exchanges a refresh token for new tokens
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, msgInvalidRequestBody)
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "refresh_token is required")
		return
	}

	// Refresh with Keycloak
	tokenResp, err := h.keycloak.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid or expired refresh token")
		return
	}

	// Get user info with new token - fallback to empty userInfo like Login does
	userInfo, err := h.keycloak.GetUserInfo(r.Context(), tokenResp.AccessToken)
	if err != nil {
		// Fallback: we don't have the original username in refresh, use empty
		// The session state from Keycloak can still be used
		userInfo = &auth.UserInfo{}
	}

	tenantID := extractTenantID(userInfo)

	// Create new internal JWT
	internalToken, err := h.createInternalToken(userInfo.PreferredUsername, tenantID, tokenResp.SessionState)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, "failed to create session token")
		return
	}

	response := LoginResponse{
		AccessToken:  internalToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    int(internalTokenTTL.Seconds()), // Use internal token TTL, not Keycloak's
		TokenType:    "Bearer",
		User:         userInfo.PreferredUsername,
		TenantID:     tenantID,
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(response))
}

// Logout invalidates the user's refresh token
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, msgInvalidRequestBody)
		return
	}

	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "refresh_token is required")
		return
	}

	// Logout from Keycloak
	if err := h.keycloak.Logout(r.Context(), req.RefreshToken); err != nil {
		// Log but don't fail - token might already be invalid
		// Use same message as success to prevent information leakage
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(map[string]string{
		"message": logoutMessage,
	}))
}

// Me returns the current user's info
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "missing authorization header")
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid authorization header")
		return
	}

	// Validate our internal token
	claims, err := auth.ValidateToken(parts[1], h.jwtSecret)
	if err != nil {
		writeError(w, http.StatusUnauthorized, ErrCodeUnauthorized, "invalid token")
		return
	}

	writeJSON(w, http.StatusOK, models.SuccessResponse(map[string]interface{}{
		"user":          claims.User,
		"tenant_id":     claims.TenantID,
		"login_session": claims.LoginSession,
	}))
}

// createInternalToken creates a JWT token for internal use
func (h *AuthHandler) createInternalToken(username, tenantID, sessionState string) (string, error) {
	now := time.Now()
	claims := auth.Claims{
		User:         username,
		TenantID:     tenantID,
		LoginSession: sessionState,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(internalTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "gprint",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

// publicEmailDomains contains domains from public email providers
// that should not be used as tenant identifiers
var publicEmailDomains = map[string]bool{
	"gmail.com":      true,
	"googlemail.com": true,
	"outlook.com":    true,
	"hotmail.com":    true,
	"live.com":       true,
	"yahoo.com":      true,
	"icloud.com":     true,
	"me.com":         true,
	"aol.com":        true,
	"protonmail.com": true,
	"proton.me":      true,
	"mail.com":       true,
	"zoho.com":       true,
}

// isPublicEmailDomain checks if the given domain is a public email provider
func isPublicEmailDomain(domain string) bool {
	return publicEmailDomains[strings.ToLower(domain)]
}

// extractTenantID extracts tenant ID from user info
// In a real scenario, this would come from Keycloak user attributes
// or a database lookup based on the user
func extractTenantID(userInfo *auth.UserInfo) string {
	// Default tenant ID - in production this should come from:
	// 1. Keycloak user attributes (e.g., custom claim)
	// 2. A database lookup based on user/email
	// 3. Organization/group membership in Keycloak
	//
	// For now, we use a default tenant ID for testing
	// You can customize this based on your multi-tenant strategy
	if userInfo.Email != "" {
		// Example: use email domain as tenant ID
		parts := strings.Split(userInfo.Email, "@")
		if len(parts) == 2 {
			domain := parts[1]
			// Don't use public email domains as tenant IDs
			if !isPublicEmailDomain(domain) {
				return domain
			}
			// For public domains, fall through to username-based tenant
		}
	}
	// Fallback to username or default
	if userInfo.PreferredUsername != "" {
		return "tenant_" + userInfo.PreferredUsername
	}
	return "default"
}
