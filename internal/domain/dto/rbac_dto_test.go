package dto

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
)

func TestUserRegisterRequestAllowsShortPassword(t *testing.T) {
	request := UserRegisterRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "a",
	}

	if err := binding.Validator.ValidateStruct(request); err != nil {
		t.Fatalf("short password should pass request validation: %v", err)
	}
}

func TestReplacementRequestsAllowEmptyAssignments(t *testing.T) {
	emptyIDs := []string{}
	requests := []any{
		RoleReplacementRequest{RoleIDs: &emptyIDs},
		PermissionReplacementRequest{PermissionIDs: &emptyIDs},
	}

	for _, request := range requests {
		if err := binding.Validator.ValidateStruct(request); err != nil {
			t.Errorf("empty replacement should pass request validation: %v", err)
		}
	}
}
