package guardian

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// Role constants
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleStaff  = "staff"
	RoleMember = "member"
)

// Permission represents an action on a resource
// Format: "resource:action" (e.g., "members:read", "plans:create")
type Permission string

// Common permissions
const (
	PermAll               Permission = "*"
	PermMembersRead       Permission = "members:read"
	PermMembersCreate     Permission = "members:create"
	PermMembersUpdate     Permission = "members:update"
	PermMembersDelete     Permission = "members:delete"
	PermMembersAll        Permission = "members:*"
	PermPlansRead         Permission = "plans:read"
	PermPlansCreate       Permission = "plans:create"
	PermPlansUpdate       Permission = "plans:update"
	PermPlansDelete       Permission = "plans:delete"
	PermPlansAll          Permission = "plans:*"
	PermSubscriptionsRead Permission = "subscriptions:read"
	PermSubscriptionsCreate Permission = "subscriptions:create"
	PermSubscriptionsUpdate Permission = "subscriptions:update"
	PermSubscriptionsDelete Permission = "subscriptions:delete"
	PermSubscriptionsAll  Permission = "subscriptions:*"
	PermDashboardRead     Permission = "dashboard:read"
	PermVerificationAll   Permission = "verification:*"
	PermStaffRead         Permission = "staff:read"
	PermStaffCreate       Permission = "staff:create"
	PermStaffUpdate       Permission = "staff:update"
	PermStaffDelete       Permission = "staff:delete"
	PermStaffAll          Permission = "staff:*"
	// Class/Activity permissions
	PermClassesRead       Permission = "classes:read"
	PermClassesCreate     Permission = "classes:create"
	PermClassesUpdate     Permission = "classes:update"
	PermClassesDelete     Permission = "classes:delete"
	PermClassesAll        Permission = "classes:*"
	PermInstructorsRead   Permission = "instructors:read"
	PermInstructorsCreate Permission = "instructors:create"
	PermInstructorsUpdate Permission = "instructors:update"
	PermInstructorsDelete Permission = "instructors:delete"
	PermInstructorsAll    Permission = "instructors:*"
	PermSchedulesRead     Permission = "schedules:read"
	PermSchedulesCreate   Permission = "schedules:create"
	PermSchedulesUpdate   Permission = "schedules:update"
	PermSchedulesDelete   Permission = "schedules:delete"
	PermSchedulesAll      Permission = "schedules:*"
	// Product & POS permissions
	PermProductsRead   Permission = "products:read"
	PermProductsCreate Permission = "products:create"
	PermProductsUpdate Permission = "products:update"
	PermProductsDelete Permission = "products:delete"
	PermProductsAll    Permission = "products:*"
	PermPOSCreate      Permission = "pos:create"
	PermPOSRead        Permission = "pos:read"
	// Cash register permissions
	PermCashRegisterRead  Permission = "cash_register:read"
	PermCashRegisterWrite Permission = "cash_register:write"
	// Reservation permissions
	PermReservationsRead   Permission = "reservations:read"
	PermReservationsCreate Permission = "reservations:create"
	PermReservationsUpdate Permission = "reservations:update"
	PermReservationsDelete Permission = "reservations:delete"
	PermReservationsAll    Permission = "reservations:*"
)

// rolePermissions defines the permissions for each role
var rolePermissions = map[string][]Permission{
	RoleOwner: {PermAll}, // Owner has all permissions
	RoleAdmin: {
		PermMembersAll,
		PermPlansAll,
		PermSubscriptionsAll,
		PermDashboardRead,
		PermVerificationAll,
		PermStaffRead,
		PermStaffCreate,
		PermStaffUpdate,
		// Classes: admin can do everything except delete
		PermClassesRead,
		PermClassesCreate,
		PermClassesUpdate,
		PermInstructorsRead,
		PermInstructorsCreate,
		PermInstructorsUpdate,
		PermSchedulesAll,
		// Individual sessions: admin can do everything except delete
		PermReservationsRead,
		PermReservationsCreate,
		PermReservationsUpdate,
		// Products & POS
		PermProductsAll,
		PermPOSCreate,
		PermPOSRead,
		// Cash register
		PermCashRegisterRead,
		PermCashRegisterWrite,
	},
	RoleStaff: {
		PermMembersRead,
		PermMembersCreate,
		PermMembersUpdate,
		PermPlansRead,
		PermSubscriptionsRead,
		PermDashboardRead,
		PermVerificationAll,
		// Staff can only view classes
		PermClassesRead,
		PermInstructorsRead,
		PermSchedulesRead,
		// Staff can view and create individual sessions
		PermReservationsRead,
		PermReservationsCreate,
		// Staff can view products and use POS
		PermProductsRead,
		PermPOSCreate,
		PermPOSRead,
		// Staff can view cash register but not open/close
		PermCashRegisterRead,
	},
}

// HasPermission checks if a role has a specific permission
func HasPermission(role string, required Permission) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}

	for _, perm := range permissions {
		if matchPermission(perm, required) {
			return true
		}
	}

	return false
}

// matchPermission checks if a permission matches the required permission
// Supports wildcards: "*" matches everything, "resource:*" matches all actions on a resource
func matchPermission(has, required Permission) bool {
	// Full wildcard
	if has == PermAll {
		return true
	}

	// Exact match
	if has == required {
		return true
	}

	// Resource wildcard (e.g., "members:*" matches "members:read")
	hasParts := strings.Split(string(has), ":")
	reqParts := strings.Split(string(required), ":")

	if len(hasParts) == 2 && len(reqParts) == 2 {
		if hasParts[0] == reqParts[0] && hasParts[1] == "*" {
			return true
		}
	}

	return false
}

// RequirePermission returns an Echo middleware that checks if the user has the required permission
func RequirePermission(required Permission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get("role").(string)
			if !ok || role == "" {
				return echo.NewHTTPError(403, ErrPermissionDenied.Error())
			}

			if !HasPermission(role, required) {
				return echo.NewHTTPError(403, ErrPermissionDenied.Error())
			}

			return next(c)
		}
	}
}

// RequireRole returns an Echo middleware that checks if the user has one of the allowed roles
func RequireRole(allowedRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			role, ok := c.Get("role").(string)
			if !ok || role == "" {
				return echo.NewHTTPError(403, ErrPermissionDenied.Error())
			}

			for _, allowed := range allowedRoles {
				if role == allowed {
					return next(c)
				}
			}

			return echo.NewHTTPError(403, ErrPermissionDenied.Error())
		}
	}
}
