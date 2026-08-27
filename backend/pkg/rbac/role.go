// Package rbac defines the fixed set of dashboard user roles used for
// authorization throughout the backend. It has no dependencies of its own
// so every other package (JWT claims, middleware, domain modules) can
// depend on it without risking an import cycle.
package rbac

// Role identifies what a dashboard user is allowed to do.
type Role string

const (
	SuperAdmin Role = "SUPER_ADMIN"
	Admin      Role = "ADMIN"
	HR         Role = "HR"
	Management Role = "MANAGEMENT"
)

// All lists every known role, in the order they should typically be
// displayed (most to least privileged).
var All = []Role{SuperAdmin, Admin, HR, Management}

// Valid reports whether r is one of the known roles.
func (r Role) Valid() bool {
	for _, known := range All {
		if r == known {
			return true
		}
	}
	return false
}
