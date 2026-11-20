package services

import (
	"context"
	"errors"
	"fmt"

	"auth_service/internal/config"
	"auth_service/internal/domain"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const googleIssuer = "https://accounts.google.com"

type GoogleOIDCService interface {
	AuthorizationURL(state string) string
	VerifyCallback(ctx context.Context, code, nonce string) (domain.GoogleIdentity, error)
}

type googleOIDCService struct {
	clientID string
	oauth2   *oauth2.Config
}

func NewGoogleOIDCService(cfg config.GoogleOIDCConfig) (GoogleOIDCService, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.RedirectURL == "" {
		return nil, errors.New("Google OIDC client ID, client secret, and redirect URL are required")
	}

	return &googleOIDCService{
		clientID: cfg.ClientID,
		oauth2: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
		},
	}, nil
}

func (s *googleOIDCService) AuthorizationURL(state string) string {
	return s.oauth2.AuthCodeURL(state, oauth2.AccessTypeOnline, oauth2.SetAuthURLParam("nonce", state))
}

func (s *googleOIDCService) VerifyCallback(ctx context.Context, code, nonce string) (domain.GoogleIdentity, error) {
	token, err := s.oauth2.Exchange(ctx, code)
	if err != nil {
		return domain.GoogleIdentity{}, fmt.Errorf("exchange Google authorization code: %w", err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return domain.GoogleIdentity{}, errors.New("Google token response does not contain an ID token")
	}

	provider, err := oidc.NewProvider(ctx, googleIssuer)
	if err != nil {
		return domain.GoogleIdentity{}, fmt.Errorf("discover Google OIDC provider: %w", err)
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: s.clientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return domain.GoogleIdentity{}, fmt.Errorf("verify Google ID token: %w", err)
	}

	var claims struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Nonce         string `json:"nonce"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return domain.GoogleIdentity{}, fmt.Errorf("read Google ID token claims: %w", err)
	}
	if claims.Subject == "" || claims.Email == "" || !claims.EmailVerified || claims.Nonce != nonce {
		return domain.GoogleIdentity{}, errors.New("Google identity must include a matching nonce and verified email")
	}

	return domain.GoogleIdentity{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
	}, nil
}
