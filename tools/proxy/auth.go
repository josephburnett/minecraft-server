package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sandertv/gophertunnel/minecraft/auth"
	"golang.org/x/oauth2"
)

const tokenFile = ".realm-token"

// getTokenSource returns an OAuth2 token source for Xbox Live authentication.
// It tries to load a cached token first, falling back to browser-based auth.
func getTokenSource() (oauth2.TokenSource, error) {
	token, err := loadToken()
	if err == nil {
		slog.Info("using cached authentication")
		return auth.RefreshTokenSource(token), nil
	}

	slog.Info("authenticating (check browser)...")
	token, err = auth.RequestLiveToken()
	if err != nil {
		return nil, err
	}

	if err := saveToken(token); err != nil {
		slog.Warn("could not cache token", "error", err)
	}

	return auth.RefreshTokenSource(token), nil
}

func loadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func saveToken(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(tokenFile, data, 0600)
}

// getRealmInvite returns the Realm invite code from env or file.
func getRealmInvite() (string, error) {
	if code := os.Getenv("REALM_INVITE"); code != "" {
		return code, nil
	}

	// Check .realm-invite file relative to executable, then CWD
	for _, path := range []string{
		findRelativeToExe(".realm-invite"),
		".realm-invite",
	} {
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		if err == nil {
			return trimSpace(string(data)), nil
		}
	}

	return "", fmt.Errorf("no realm invite found; set REALM_INVITE env var or create .realm-invite file")
}

func findRelativeToExe(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(filepath.Dir(exe))
	return filepath.Join(dir, name)
}

func trimSpace(s string) string {
	// Trim whitespace and newlines
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	for len(s) > 0 && (s[0] == '\n' || s[0] == '\r' || s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	return s
}
