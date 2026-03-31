package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

// Hardcoded read-only scopes. Never add write scopes.
var oauthScopes = []string{
	drive.DriveReadonlyScope,
	docs.DocumentsReadonlyScope,
	sheets.SpreadsheetsReadonlyScope,
}

// resolveFilePath returns the path for a config file.
// It checks the given env var first, then falls back to a file
// in the same directory as the executable.
func resolveFilePath(envVar, filename string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to determine executable path: %v", err)
	}
	return filepath.Join(filepath.Dir(exe), filename)
}

// loadOAuthConfig reads credentials.json and returns an OAuth2 config.
func loadOAuthConfig(credentialsPath string, redirectURL string) (*oauth2.Config, error) {
	b, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file %q: %w", credentialsPath, err)
	}
	config, err := google.ConfigFromJSON(b, oauthScopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials file: %w", err)
	}
	config.RedirectURL = redirectURL
	return config, nil
}

// loadToken reads a token from a file.
func loadToken(path string) (*oauth2.Token, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("unable to parse token file %q: %w", path, err)
	}
	return tok, nil
}

// saveToken writes a token to a file with restricted permissions.
func saveToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal token: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("unable to write token file %q: %w", path, err)
	}
	return nil
}

// generateState creates a cryptographically random state parameter for CSRF protection.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// openBrowser opens the given URL in the user's default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform %q — open this URL manually: %s", runtime.GOOS, url)
	}
	return cmd.Start()
}

// getTokenFromWeb runs a local HTTP server to handle the OAuth callback,
// opens the browser for user consent, and returns the resulting token.
// All user-facing output goes to stderr (stdout is the MCP transport).
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	state, err := generateState()
	if err != nil {
		return nil, err
	}

	// Bind to a random available port on localhost.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start local HTTP server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	config.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errChan <- fmt.Errorf("OAuth state mismatch: possible CSRF attack")
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			http.Error(w, "Authorization failed: "+errMsg, http.StatusBadRequest)
			errChan <- fmt.Errorf("OAuth error: %s", errMsg)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No authorization code received", http.StatusBadRequest)
			errChan <- fmt.Errorf("no authorization code in callback")
			return
		}
		fmt.Fprint(w, "<html><body><h1>Authorization successful!</h1><p>You can close this window.</p></body></html>")
		codeChan <- code
	})

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	log.Printf("Opening browser for Google authorization...")
	log.Printf("If the browser doesn't open, visit this URL manually:\n%s", authURL)

	if err := openBrowser(authURL); err != nil {
		log.Printf("Could not open browser: %v", err)
	}

	// Wait for the callback or timeout.
	var code string
	select {
	case code = <-codeChan:
		// Success
	case err := <-errChan:
		_ = server.Close()
		return nil, err
	case <-time.After(5 * time.Minute):
		_ = server.Close()
		return nil, fmt.Errorf("authorization timed out after 5 minutes")
	}

	// Shut down the callback server.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	// Exchange the auth code for a token.
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}
	return token, nil
}

// getOAuthClient returns an authenticated HTTP client.
// On first run, it performs the browser-based OAuth flow.
// On subsequent runs, it loads the saved token and refreshes it as needed.
func getOAuthClient(credentialsPath, tokenPath string) (*http.Client, error) {
	// Load credentials — we need these to know the redirect URL for token loading too,
	// but we'll set the real redirect URL during the web flow if needed.
	config, err := loadOAuthConfig(credentialsPath, "")
	if err != nil {
		return nil, err
	}

	// Try loading an existing token.
	token, err := loadToken(tokenPath)
	if err != nil {
		// No existing token — run the OAuth flow.
		log.Printf("No saved token found. Starting OAuth authorization flow...")
		token, err = getTokenFromWeb(config)
		if err != nil {
			return nil, fmt.Errorf("OAuth authorization failed: %w", err)
		}
		if err := saveToken(tokenPath, token); err != nil {
			return nil, err
		}
		log.Printf("Token saved to %s", tokenPath)
	}

	// Create a token source that auto-refreshes and persists the refreshed token.
	tokenSource := config.TokenSource(context.Background(), token)
	refreshingSource := &persistingTokenSource{
		base:      oauth2.ReuseTokenSource(token, tokenSource),
		tokenPath: tokenPath,
		lastToken: token,
	}

	return oauth2.NewClient(context.Background(), refreshingSource), nil
}

// persistingTokenSource wraps a token source and saves refreshed tokens to disk.
type persistingTokenSource struct {
	base      oauth2.TokenSource
	tokenPath string
	lastToken *oauth2.Token
}

func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.base.Token()
	if err != nil {
		return nil, err
	}
	// If the token changed (was refreshed), persist it.
	if token.AccessToken != s.lastToken.AccessToken {
		if saveErr := saveToken(s.tokenPath, token); saveErr != nil {
			log.Printf("Warning: could not save refreshed token: %v", saveErr)
		}
		s.lastToken = token
	}
	return token, nil
}
