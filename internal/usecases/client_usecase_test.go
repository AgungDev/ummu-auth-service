package usecases

import (
	"testing"

	"auth_service/internal/domain/dto"
)

func TestValidateClientAuthorizationRejectsPasswordGrant(t *testing.T) {
	request := dto.ClientRequest{
		Grants:         []string{"password"},
		Scopes:         []string{"student:read"},
		Audiences:      []string{"student-service"},
		IsConfidential: true,
	}
	if err := validateClientAuthorization(request); err == nil {
		t.Fatal("validateClientAuthorization() error = nil, want password grant rejection")
	}
}

func TestValidateClientAuthorizationRequiresConfidentialClientCredentials(t *testing.T) {
	request := dto.ClientRequest{
		Grants:    []string{"client_credentials"},
		Scopes:    []string{"student:read"},
		Audiences: []string{"student-service"},
	}
	if err := validateClientAuthorization(request); err == nil {
		t.Fatal("validateClientAuthorization() error = nil, want confidential-client validation")
	}
}
