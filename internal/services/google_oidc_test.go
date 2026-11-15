package services

import (
	"net/url"
	"strings"
	"testing"

	"auth_service/internal/config"
)

func TestGoogleAuthorizationURL(t *testing.T) {
	service, err := NewGoogleOIDCService(config.GoogleOIDCConfig{
		ClientID:     "google-client-id",
		ClientSecret: "google-client-secret",
		RedirectURL:  "http://localhost:9001/auth/google/callback",
	})
	if err != nil {
		t.Fatalf("NewGoogleOIDCService() error = %v", err)
	}

	authorizationURL, err := url.Parse(service.AuthorizationURL("test-state"))
	if err != nil {
		t.Fatalf("AuthorizationURL() returned an invalid URL: %v", err)
	}
	query := authorizationURL.Query()
	if query.Get("redirect_uri") != "http://localhost:9001/auth/google/callback" {
		t.Errorf("redirect_uri = %q", query.Get("redirect_uri"))
	}
	if query.Get("state") != "test-state" || query.Get("nonce") != "test-state" {
		t.Errorf("state and nonce must match; state=%q nonce=%q", query.Get("state"), query.Get("nonce"))
	}
	for _, scope := range []string{"openid", "email", "profile"} {
		if !strings.Contains(" "+query.Get("scope")+" ", " "+scope+" ") {
			t.Errorf("scope %q is missing from %q", scope, query.Get("scope"))
		}
	}
}

func TestNewGoogleOIDCServiceRequiresCredentials(t *testing.T) {
	if _, err := NewGoogleOIDCService(config.GoogleOIDCConfig{}); err == nil {
		t.Fatal("NewGoogleOIDCService() error = nil, want missing-credentials error")
	}
}
