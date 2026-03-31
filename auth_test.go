package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	original := &oauth2.Token{
		AccessToken:  "ya29.test-access-token",
		TokenType:    "Bearer",
		RefreshToken: "1//test-refresh-token",
		Expiry:       time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
	}

	if err := saveToken(path, original); err != nil {
		t.Fatalf("saveToken failed: %v", err)
	}

	// Verify file permissions (unix only).
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat token file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("token file permissions = %o, want 0600", perm)
	}

	loaded, err := loadToken(path)
	if err != nil {
		t.Fatalf("loadToken failed: %v", err)
	}

	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken = %q, want %q", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", loaded.RefreshToken, original.RefreshToken)
	}
	if loaded.TokenType != original.TokenType {
		t.Errorf("TokenType = %q, want %q", loaded.TokenType, original.TokenType)
	}
	if !loaded.Expiry.Equal(original.Expiry) {
		t.Errorf("Expiry = %v, want %v", loaded.Expiry, original.Expiry)
	}
}

func TestTokenRoundTripPreservesJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	original := &oauth2.Token{
		AccessToken:  "ya29.access",
		RefreshToken: "1//refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour).Truncate(time.Second),
	}
	if err := saveToken(path, original); err != nil {
		t.Fatalf("saveToken: %v", err)
	}

	// Verify the file is valid JSON with expected structure.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("token file is not valid JSON: %v", err)
	}
	if _, ok := raw["access_token"]; !ok {
		t.Error("token JSON missing 'access_token' field")
	}
	if _, ok := raw["refresh_token"]; !ok {
		t.Error("token JSON missing 'refresh_token' field")
	}
}

func TestLoadTokenNonexistent(t *testing.T) {
	_, err := loadToken("/nonexistent/path/token.json")
	if err == nil {
		t.Error("expected error loading nonexistent token, got nil")
	}
}

func TestLoadTokenInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token.json")

	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := loadToken(path)
	if err == nil {
		t.Error("expected error loading invalid JSON token, got nil")
	}
}

func TestLoadOAuthConfigValidShape(t *testing.T) {
	dir := t.TempDir()
	credPath := filepath.Join(dir, "credentials.json")

	// Realistic credentials.json for a desktop OAuth client.
	creds := map[string]any{
		"installed": map[string]any{
			"client_id":                   "123456789-test.apps.googleusercontent.com",
			"client_secret":               "GOCSPX-test-secret",
			"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
			"token_uri":                   "https://oauth2.googleapis.com/token",
			"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
			"redirect_uris":               []string{"http://localhost"},
		},
	}
	data, _ := json.Marshal(creds)
	if err := os.WriteFile(credPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	config, err := loadOAuthConfig(credPath, "http://localhost:12345/callback")
	if err != nil {
		t.Fatalf("loadOAuthConfig failed: %v", err)
	}

	if config.ClientID != "123456789-test.apps.googleusercontent.com" {
		t.Errorf("ClientID = %q, want test client ID", config.ClientID)
	}
	if config.RedirectURL != "http://localhost:12345/callback" {
		t.Errorf("RedirectURL = %q, want http://localhost:12345/callback", config.RedirectURL)
	}

	// Verify scopes match our hardcoded read-only scopes.
	if len(config.Scopes) != len(oauthScopes) {
		t.Fatalf("got %d scopes, want %d", len(config.Scopes), len(oauthScopes))
	}
	for i, scope := range config.Scopes {
		if scope != oauthScopes[i] {
			t.Errorf("scope[%d] = %q, want %q", i, scope, oauthScopes[i])
		}
	}
}

func TestLoadOAuthConfigMissingFile(t *testing.T) {
	_, err := loadOAuthConfig("/nonexistent/credentials.json", "")
	if err == nil {
		t.Error("expected error for missing credentials file, got nil")
	}
}

func TestLoadOAuthConfigInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials.json")
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := loadOAuthConfig(path, "")
	if err == nil {
		t.Error("expected error for invalid credentials JSON, got nil")
	}
}

func TestGenerateStateUniqueness(t *testing.T) {
	states := make(map[string]bool)
	for range 100 {
		s, err := generateState()
		if err != nil {
			t.Fatalf("generateState: %v", err)
		}
		if len(s) != 32 { // 16 bytes = 32 hex chars
			t.Errorf("state length = %d, want 32", len(s))
		}
		if states[s] {
			t.Errorf("duplicate state generated: %s", s)
		}
		states[s] = true
	}
}

func TestResolveFilePath(t *testing.T) {
	t.Run("env var takes precedence", func(t *testing.T) {
		t.Setenv("TEST_CRED_PATH", "/custom/path/credentials.json")
		got := resolveFilePath("TEST_CRED_PATH", "credentials.json")
		if got != "/custom/path/credentials.json" {
			t.Errorf("resolveFilePath = %q, want /custom/path/credentials.json", got)
		}
	})

	t.Run("fallback to executable dir", func(t *testing.T) {
		// Unset env var to test fallback.
		t.Setenv("TEST_UNUSED_VAR", "")
		got := resolveFilePath("NONEXISTENT_ENV_VAR_12345", "credentials.json")
		if filepath.Base(got) != "credentials.json" {
			t.Errorf("resolveFilePath base = %q, want credentials.json", filepath.Base(got))
		}
	})
}

func TestOAuthScopesAreReadOnly(t *testing.T) {
	for _, scope := range oauthScopes {
		if scope == "https://www.googleapis.com/auth/drive" ||
			scope == "https://www.googleapis.com/auth/documents" ||
			scope == "https://www.googleapis.com/auth/spreadsheets" {
			t.Errorf("scope %q is a write scope — only read-only scopes are allowed", scope)
		}
	}

	// Verify the exact expected scopes.
	expected := []string{
		"https://www.googleapis.com/auth/drive.readonly",
		"https://www.googleapis.com/auth/documents.readonly",
		"https://www.googleapis.com/auth/spreadsheets.readonly",
	}
	if len(oauthScopes) != len(expected) {
		t.Fatalf("got %d scopes, want %d", len(oauthScopes), len(expected))
	}
	for i, scope := range oauthScopes {
		if scope != expected[i] {
			t.Errorf("oauthScopes[%d] = %q, want %q", i, scope, expected[i])
		}
	}
}
