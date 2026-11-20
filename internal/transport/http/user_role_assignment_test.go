package http

import (
	"testing"

	"auth_service/internal/domain/dto"
	"github.com/google/uuid"
)

func TestParseRoleAssignmentsSupportsMultipleScopedContexts(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.New()
	assignments, err := parseRoleAssignments(userID, nil, []dto.RoleScopeAssignment{
		{RoleID: roleID.String(), ScopeType: "faculty", ScopeID: "faculty-1"},
		{RoleID: roleID.String(), ScopeType: "department", ScopeID: "department-1"},
		{RoleID: roleID.String(), ScopeType: "self"},
	})
	if err != nil {
		t.Fatalf("parseRoleAssignments() error = %v", err)
	}
	if len(assignments) != 3 {
		t.Fatalf("assignment count = %d, want 3", len(assignments))
	}
	if assignments[2].ScopeType != "self" || assignments[2].ScopeID != userID.String() {
		t.Fatalf("self scope = %#v, want the target user ID", assignments[2])
	}
}

func TestParseRoleAssignmentsRejectsInvalidScopesAndMixedPayloads(t *testing.T) {
	userID := uuid.New()
	roleID := uuid.NewString()
	tests := []struct {
		name        string
		roleIDs     []string
		assignments []dto.RoleScopeAssignment
	}{
		{name: "unscoped global", assignments: []dto.RoleScopeAssignment{{RoleID: roleID, ScopeType: "global", ScopeID: "unit-1"}}},
		{name: "self scope spoofing", assignments: []dto.RoleScopeAssignment{{RoleID: roleID, ScopeType: "self", ScopeID: uuid.NewString()}}},
		{name: "missing organization scope id", assignments: []dto.RoleScopeAssignment{{RoleID: roleID, ScopeType: "faculty"}}},
		{name: "mixed legacy and scoped payload", roleIDs: []string{roleID}, assignments: []dto.RoleScopeAssignment{{RoleID: roleID, ScopeType: "global"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseRoleAssignments(userID, test.roleIDs, test.assignments); err == nil {
				t.Fatal("parseRoleAssignments() error = nil, want validation error")
			}
		})
	}
}
