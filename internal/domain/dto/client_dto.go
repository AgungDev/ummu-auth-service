package dto

type ClientRequest struct {
	Name           string                       `json:"name" binding:"required"`
	ClientSecret   string                       `json:"client_secret" binding:"required"`
	Grants         []string                     `json:"grants" binding:"required,min=1,dive,required"`
	Scopes         []string                     `json:"scopes" binding:"required,min=1,dive,required"`
	Audiences      []string                     `json:"audiences" binding:"required,min=1,dive,required"`
	AccessContexts []ClientAccessContextRequest `json:"access_contexts"`
	RedirectURIs   []string                     `json:"redirect_uris"`
	IsConfidential bool                         `json:"is_confidential"`
	Status         string                       `json:"status"`
}

type ClientResponse struct {
	ID             string                       `json:"id"`
	Name           string                       `json:"name"`
	RedirectURIs   []string                     `json:"redirect_uris"`
	Grants         []string                     `json:"grants"`
	Scopes         []string                     `json:"scopes"`
	Audiences      []string                     `json:"audiences"`
	AccessContexts []ClientAccessContextRequest `json:"access_contexts"`
	IsConfidential bool                         `json:"is_confidential"`
	Status         string                       `json:"status"`
}

type ClientAccessContextRequest struct {
	ScopeType string `json:"scope_type" binding:"required"`
	ScopeID   string `json:"scope_id"`
}
