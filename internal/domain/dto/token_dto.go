package dto

type TokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type" binding:"required"`
	Audience     string `json:"audience" form:"audience" binding:"required"`
	Scope        string `json:"scope" form:"scope" binding:"required"`
	ClientID     string `json:"client_id" form:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

type AuthLoginResponse struct {
	AccessToken  string       `json:"access_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int          `json:"expires_in"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type IntrospectRequest struct {
	Token string `json:"token" binding:"required"`
}

type IntrospectResponse struct {
	Active         bool                    `json:"active"`
	User           map[string]string       `json:"user,omitempty"`
	PrincipalType  string                  `json:"principal_type,omitempty"`
	Subject        string                  `json:"sub,omitempty"`
	ClientID       string                  `json:"client_id,omitempty"`
	Scope          string                  `json:"scope,omitempty"`
	Roles          []string                `json:"roles,omitempty"`
	Permissions    []string                `json:"permissions,omitempty"`
	AccessContexts []AccessContextResponse `json:"access_contexts,omitempty"`
}

type AccessContextResponse struct {
	Role        string              `json:"role,omitempty"`
	Scope       AccessScopeResponse `json:"scope"`
	Permissions []string            `json:"permissions,omitempty"`
}

type AccessScopeResponse struct {
	Type string  `json:"type"`
	ID   *string `json:"id"`
}

type PermissionsResponse struct {
	Permissions []string `json:"permissions"`
}

type AuthorizeRequest struct {
	UserID     string `json:"user_id" binding:"required"`
	Permission string `json:"permission" binding:"required"`
}

type AuthorizeResponse struct {
	Authorized bool   `json:"authorized"`
	UserID     string `json:"user_id"`
	Permission string `json:"permission"`
}
