package http

import (
	"net/url"
	"testing"
)

func TestBuildGoogleFrontendCallbackURL(t *testing.T) {
	callbackURL, err := buildGoogleFrontendCallbackURL(
		"http://localhost:5173/oauth/callback",
		url.Values{
			"access_token":  {"token+/="},
			"refresh_token": {"refresh-token"},
		},
	)
	if err != nil {
		t.Fatalf("buildGoogleFrontendCallbackURL() error = %v", err)
	}

	parsed, err := url.Parse(callbackURL)
	if err != nil {
		t.Fatalf("returned callback URL is invalid: %v", err)
	}
	if parsed.Scheme != "http" || parsed.Host != "localhost:5173" || parsed.Path != "/oauth/callback" {
		t.Fatalf("callback URL target = %q", callbackURL)
	}
	values, err := url.ParseQuery(parsed.Fragment)
	if err != nil {
		t.Fatalf("callback URL fragment is invalid: %v", err)
	}
	if values.Get("access_token") != "token+/=" || values.Get("refresh_token") != "refresh-token" {
		t.Fatalf("callback URL fragment did not preserve token values: %q", parsed.Fragment)
	}
}

func TestBuildGoogleFrontendCallbackURLRejectsUnsafeTarget(t *testing.T) {
	for _, target := range []string{
		"javascript:alert(1)",
		"https://user:password@example.com/callback",
		"https://example.com/callback?next=/",
		"https://example.com/callback#existing",
		"http://example.com/callback",
	} {
		t.Run(target, func(t *testing.T) {
			if _, err := buildGoogleFrontendCallbackURL(target, url.Values{}); err == nil {
				t.Fatal("buildGoogleFrontendCallbackURL() error = nil, want invalid callback URL error")
			}
		})
	}
}
