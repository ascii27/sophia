package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// TokenManager handles OAuth2 token operations
type TokenManager struct {
	tokenDir string
}

// NewTokenManager creates a new token manager
func NewTokenManager(tokenDir string) *TokenManager {
	return &TokenManager{
		tokenDir: tokenDir,
	}
}

// GetToken retrieves a token, either from file or by initiating the web flow
func (tm *TokenManager) GetToken(ctx context.Context, config *oauth2.Config, serviceName string) (*oauth2.Token, error) {
	tokFile := filepath.Join(tm.tokenDir, fmt.Sprintf("%s_token.json", serviceName))
	tok, err := tm.tokenFromFile(tokFile)
	if err != nil {
		tok, err = tm.getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to get token from web: %w", err)
		}
		if err := tm.saveToken(tokFile, tok); err != nil {
			return nil, fmt.Errorf("failed to save token: %w", err)
		}
	}
	return tok, nil
}

// getTokenFromWeb requests a token from the web
func (tm *TokenManager) getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("failed to read authorization code: %w", err)
	}

	tok, err := config.Exchange(ctx, authCode)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange token: %w", err)
	}
	return tok, nil
}

// tokenFromFile retrieves a token from a local file
func (tm *TokenManager) tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken saves a token to a file path
func (tm *TokenManager) saveToken(path string, token *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create token directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create token file: %w", err)
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}
