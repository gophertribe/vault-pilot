package google

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

// NewHTTPClient creates an authenticated HTTP client from a service account JSON key file.
// This is useful for APIs that require an *http.Client (e.g., Gmail).
func NewHTTPClient(ctx context.Context, credentialsFile string, scopes ...string) (*http.Client, error) {
	data, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	conf, err := google.JWTConfigFromJSON(data, scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return conf.Client(ctx), nil
}

// ClientOption returns an option.ClientOption for use with Google API service
// constructors (Calendar, Drive, etc.).
func ClientOption(credentialsFile string) option.ClientOption {
	return option.WithCredentialsFile(credentialsFile)
}
