// Package auth provides Keycloak OAuth2 authentication utilities.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTP header constants
const (
	headerContentType     = "Content-Type"
	contentTypeFormURLEnc = "application/x-www-form-urlencoded"
)

// KeycloakConfig holds Keycloak server configuration
type KeycloakConfig struct {
	BaseURL      string // e.g., https://keycloak.example.com or http://localhost:8180
	Realm        string // e.g., master
	ClientID     string // e.g., gprint-client
	ClientSecret string // optional, for confidential clients
}

// Validate checks that required configuration fields are set
func (c KeycloakConfig) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("keycloak BaseURL is required")
	}
	if c.Realm == "" {
		return fmt.Errorf("keycloak Realm is required")
	}
	if c.ClientID == "" {
		return fmt.Errorf("keycloak ClientID is required")
	}
	return nil
}

// KeycloakClient handles OAuth2 token operations with Keycloak
type KeycloakClient struct {
	config     KeycloakConfig
	httpClient *http.Client
}

// TokenResponse represents the OAuth2 token response from Keycloak
type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	IDToken          string `json:"id_token,omitempty"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
}

// KeycloakError represents an error response from Keycloak
type KeycloakError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// UserInfo represents the user info from Keycloak
type UserInfo struct {
	Sub               string `json:"sub"`
	EmailVerified     bool   `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	GivenName         string `json:"given_name"`
	FamilyName        string `json:"family_name"`
	Email             string `json:"email"`
}

// PKCEChallenge holds PKCE parameters for Authorization Code flow
type PKCEChallenge struct {
	CodeVerifier        string
	CodeChallenge       string
	CodeChallengeMethod string
}

// NewKeycloakClient creates a new Keycloak client with validated config
// Panics if required config fields are missing
func NewKeycloakClient(config KeycloakConfig) *KeycloakClient {
	if err := config.Validate(); err != nil {
		panic(err.Error())
	}
	return &KeycloakClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKeycloakClientWithHTTPClient creates a Keycloak client with a custom HTTP client
// Useful for testing with mock HTTP clients
func NewKeycloakClientWithHTTPClient(config KeycloakConfig, httpClient *http.Client) *KeycloakClient {
	if err := config.Validate(); err != nil {
		panic(err.Error())
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &KeycloakClient{
		config:     config,
		httpClient: httpClient,
	}
}

// tokenEndpoint returns the OAuth2 token endpoint URL
func (k *KeycloakClient) tokenEndpoint() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token",
		strings.TrimSuffix(k.config.BaseURL, "/"),
		url.PathEscape(k.config.Realm),
	)
}

// userInfoEndpoint returns the userinfo endpoint URL
func (k *KeycloakClient) userInfoEndpoint() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/userinfo",
		strings.TrimSuffix(k.config.BaseURL, "/"),
		url.PathEscape(k.config.Realm),
	)
}

// logoutEndpoint returns the logout endpoint URL
func (k *KeycloakClient) logoutEndpoint() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/logout",
		strings.TrimSuffix(k.config.BaseURL, "/"),
		url.PathEscape(k.config.Realm),
	)
}

// authorizationEndpoint returns the OAuth2 authorization endpoint URL
func (k *KeycloakClient) authorizationEndpoint() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/auth",
		strings.TrimSuffix(k.config.BaseURL, "/"),
		url.PathEscape(k.config.Realm),
	)
}

// Login authenticates a user with username and password.
//
// Deprecated: This method uses the Resource Owner Password Credentials (ROPC) grant
// which is considered insecure and deprecated by OAuth 2.1. It exposes user credentials
// directly to the client application and should only be used for legacy integrations
// or internal tools where no alternative is available.
//
// For new implementations, use the Authorization Code flow with PKCE instead:
//   - Call StartAuthCodePKCE() to generate PKCE parameters and redirect URL
//   - Redirect user to Keycloak login page
//   - Exchange the authorization code using ExchangeAuthCode()
func (k *KeycloakClient) Login(ctx context.Context, username, password string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("client_id", k.config.ClientID)
	data.Set("username", username)
	data.Set("password", password)
	data.Set("scope", "openid profile email")

	if k.config.ClientSecret != "" {
		data.Set("client_secret", k.config.ClientSecret)
	}

	return k.doTokenRequest(ctx, data)
}

// GeneratePKCE creates PKCE code verifier and challenge for Authorization Code flow
func GeneratePKCE() (*PKCEChallenge, error) {
	// Generate 32 random bytes for code verifier
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}

	// Base64 URL encode without padding
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Create SHA256 hash of verifier for challenge
	hash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(hash[:])

	return &PKCEChallenge{
		CodeVerifier:        codeVerifier,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: "S256",
	}, nil
}

// StartAuthCodePKCE generates the authorization URL for OAuth2 Authorization Code flow with PKCE.
// Returns the PKCE challenge (store code_verifier for token exchange) and the redirect URL.
func (k *KeycloakClient) StartAuthCodePKCE(redirectURI, state string) (*PKCEChallenge, string, error) {
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, "", err
	}

	params := url.Values{}
	params.Set("client_id", k.config.ClientID)
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("redirect_uri", redirectURI)
	params.Set("state", state)
	params.Set("code_challenge", pkce.CodeChallenge)
	params.Set("code_challenge_method", pkce.CodeChallengeMethod)

	authURL := fmt.Sprintf("%s?%s", k.authorizationEndpoint(), params.Encode())

	return pkce, authURL, nil
}

// ExchangeAuthCode exchanges an authorization code for tokens using PKCE
func (k *KeycloakClient) ExchangeAuthCode(ctx context.Context, code, redirectURI, codeVerifier string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", k.config.ClientID)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("code_verifier", codeVerifier)

	if k.config.ClientSecret != "" {
		data.Set("client_secret", k.config.ClientSecret)
	}

	return k.doTokenRequest(ctx, data)
}

// RefreshToken exchanges a refresh token for a new access token
func (k *KeycloakClient) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", k.config.ClientID)
	data.Set("refresh_token", refreshToken)

	if k.config.ClientSecret != "" {
		data.Set("client_secret", k.config.ClientSecret)
	}

	return k.doTokenRequest(ctx, data)
}

// Logout invalidates the refresh token
func (k *KeycloakClient) Logout(ctx context.Context, refreshToken string) error {
	data := url.Values{}
	data.Set("client_id", k.config.ClientID)
	data.Set("refresh_token", refreshToken)

	if k.config.ClientSecret != "" {
		data.Set("client_secret", k.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.logoutEndpoint(), strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create logout request: %w", err)
	}

	req.Header.Set(headerContentType, contentTypeFormURLEnc)

	resp, err := k.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("logout request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("logout failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetUserInfo retrieves user information using an access token
func (k *KeycloakClient) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, k.userInfoEndpoint(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := k.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo failed with status %d: %s", resp.StatusCode, string(body))
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}

	return &userInfo, nil
}

// doTokenRequest performs a token request to Keycloak
func (k *KeycloakClient) doTokenRequest(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, k.tokenEndpoint(), strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set(headerContentType, contentTypeFormURLEnc)

	resp, err := k.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var keycloakErr KeycloakError
		if err := json.Unmarshal(body, &keycloakErr); err == nil && keycloakErr.Error != "" {
			return nil, fmt.Errorf("%s: %s", keycloakErr.Error, keycloakErr.ErrorDescription)
		}
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// IntrospectToken validates a token with Keycloak (optional, for token introspection)
func (k *KeycloakClient) IntrospectToken(ctx context.Context, token string) (map[string]interface{}, error) {
	introspectURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token/introspect",
		strings.TrimSuffix(k.config.BaseURL, "/"),
		k.config.Realm,
	)

	data := url.Values{}
	data.Set("token", token)
	data.Set("client_id", k.config.ClientID)

	if k.config.ClientSecret != "" {
		data.Set("client_secret", k.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, introspectURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create introspect request: %w", err)
	}

	req.Header.Set(headerContentType, contentTypeFormURLEnc)

	resp, err := k.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("introspect request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful HTTP status before decoding
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("introspect failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode introspect response: %w", err)
	}

	return result, nil
}
