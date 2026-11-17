package dto

type RoleAssignmentRequest struct {
	RoleIDs []string `json:"role_ids" binding:"required,min=1"`
}

type PermissionAssignmentRequest struct {
	PermissionIDs []string `json:"permission_ids" binding:"required,min=1"`
}

type RoleReplacementRequest struct {
	RoleIDs *[]string `json:"role_ids" binding:"required"`
}

type PermissionReplacementRequest struct {
	PermissionIDs *[]string `json:"permission_ids" binding:"required"`
}

type CheckPermissionRequest struct {
	Permission string `json:"permission" binding:"required"`
}
