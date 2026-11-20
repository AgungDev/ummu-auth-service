package http

import (
	"encoding/base64"
	"net/http/httptest"
	"testing"

	"auth_service/internal/domain"
	"github.com/gin-gonic/gin"
)

func TestRequestedClientScopesOnlyReturnsGrantedUniqueScopes(t *testing.T) {
	granted := []domain.ClientScope{{Scope: "student:read"}, {Scope: "student:write"}}

	got, err := requestedClientScopes(granted, "student:write student:read")
	if err != nil {
		t.Fatalf("requestedClientScopes() error = %v", err)
	}
	if len(got) != 2 || got[0] != "student:read" || got[1] != "student:write" {
		t.Fatalf("requestedClientScopes() = %v, want sorted allowed scopes", got)
	}

	for _, requested := range []string{"student:read student:read", "student:delete", " "} {
		if _, err := requestedClientScopes(granted, requested); err == nil {
			t.Errorf("requestedClientScopes(%q) error = nil, want invalid scope error", requested)
		}
	}
}

func TestClientCredentialsAcceptsBasicOrBodyCredentials(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/api/v1/oauth/token", nil)
	if id, secret, ok := clientCredentials(ctx, "body-client", "body-secret"); !ok || id != "body-client" || secret != "body-secret" {
		t.Fatalf("body credentials = %q/%q/%v", id, secret, ok)
	}

	ctx.Request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("basic-client:basic-secret")))
	if id, secret, ok := clientCredentials(ctx, "ignored", "ignored"); !ok || id != "basic-client" || secret != "basic-secret" {
		t.Fatalf("Basic credentials = %q/%q/%v", id, secret, ok)
	}
}
