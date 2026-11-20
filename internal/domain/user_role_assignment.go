package domain

import "github.com/google/uuid"

type UserRoleAssignment struct {
	Role      Role
	ScopeType string
	ScopeID   string
}

type RoleAssignmentInput struct {
	RoleID    uuid.UUID
	ScopeType string
	ScopeID   string
}
