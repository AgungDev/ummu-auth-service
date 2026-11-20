package dto

type RoleAssignmentRequest struct {
	RoleIDs     []string              `json:"role_ids"`
	Assignments []RoleScopeAssignment `json:"assignments"`
}

type RoleScopeAssignment struct {
	RoleID    string `json:"role_id" binding:"required"`
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
}

type UserRoleResponse struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ScopeType   string `json:"scope_type"`
	ScopeID     string `json:"scope_id,omitempty"`
}

type PermissionAssignmentRequest struct {
	PermissionIDs []string `json:"permission_ids" binding:"required,min=1"`
}

type RoleReplacementRequest struct {
	RoleIDs     *[]string              `json:"role_ids"`
	Assignments *[]RoleScopeAssignment `json:"assignments"`
}

type PermissionReplacementRequest struct {
	PermissionIDs *[]string `json:"permission_ids" binding:"required"`
}

type CheckPermissionRequest struct {
	Permission string `json:"permission" binding:"required"`
}
