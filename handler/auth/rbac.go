package auth

import (
	"context"
	"fmt"
)

// Authorizer checks whether an identity is allowed to perform an action.
type Authorizer interface {
	Authorize(ctx context.Context, identity *Identity, resource, action string) error
}

// Permission maps a resource to allowed actions.
type Permission struct {
	Resource string
	Actions  []string
}

// Role is a named set of permissions.
type Role struct {
	Name        string
	Permissions []Permission
}

// RBACAuthorizer implements role-based access control.
type RBACAuthorizer struct {
	roles map[string]*Role
}

// NewRBACAuthorizer creates a new RBAC authorizer.
func NewRBACAuthorizer() *RBACAuthorizer {
	return &RBACAuthorizer{roles: make(map[string]*Role)}
}

// Authorize checks whether the identity's scopes allow the given action on the resource.
func (a *RBACAuthorizer) Authorize(ctx context.Context, identity *Identity, resource, action string) error {
	for _, scope := range identity.Scopes {
		role, ok := a.roles[scope]
		if !ok {
			continue
		}
		for _, perm := range role.Permissions {
			if perm.Resource != resource && perm.Resource != "*" {
				continue
			}
			for _, allowed := range perm.Actions {
				if allowed == action || allowed == "*" {
					return nil
				}
			}
		}
	}
	return fmt.Errorf("unauthorized: %s cannot %s %s", identity.ID, action, resource)
}
