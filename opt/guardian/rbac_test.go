package guardian

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHasPermission(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		required Permission
		want     bool
	}{
		{
			name:     "success - owner has wildcard permission",
			role:     RoleOwner,
			required: PermMembersRead,
			want:     true,
		},
		{
			name:     "success - owner has all permissions via wildcard",
			role:     RoleOwner,
			required: PermPlansDelete,
			want:     true,
		},
		{
			name:     "success - admin has members wildcard",
			role:     RoleAdmin,
			required: PermMembersRead,
			want:     true,
		},
		{
			name:     "success - admin has members create via wildcard",
			role:     RoleAdmin,
			required: PermMembersCreate,
			want:     true,
		},
		{
			name:     "success - staff has exact permission",
			role:     RoleStaff,
			required: PermMembersRead,
			want:     true,
		},
		{
			name:     "success - staff has verification wildcard",
			role:     RoleStaff,
			required: PermVerificationAll,
			want:     true,
		},
		{
			name:     "error - staff lacks delete permission",
			role:     RoleStaff,
			required: PermMembersDelete,
			want:     false,
		},
		{
			name:     "error - admin lacks staff delete permission",
			role:     RoleAdmin,
			required: PermStaffDelete,
			want:     false,
		},
		{
			name:     "error - invalid role",
			role:     "invalid",
			required: PermMembersRead,
			want:     false,
		},
		{
			name:     "error - empty role",
			role:     "",
			required: PermMembersRead,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPermission(tt.role, tt.required)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatchPermission(t *testing.T) {
	tests := []struct {
		name     string
		has      Permission
		required Permission
		want     bool
	}{
		{
			name:     "success - full wildcard matches everything",
			has:      PermAll,
			required: PermMembersRead,
			want:     true,
		},
		{
			name:     "success - full wildcard matches any permission",
			has:      PermAll,
			required: "custom:action",
			want:     true,
		},
		{
			name:     "success - exact match",
			has:      PermMembersRead,
			required: PermMembersRead,
			want:     true,
		},
		{
			name:     "success - resource wildcard matches action",
			has:      PermMembersAll,
			required: PermMembersRead,
			want:     true,
		},
		{
			name:     "success - resource wildcard matches create",
			has:      PermMembersAll,
			required: PermMembersCreate,
			want:     true,
		},
		{
			name:     "success - plans wildcard matches plans read",
			has:      PermPlansAll,
			required: PermPlansRead,
			want:     true,
		},
		{
			name:     "error - resource wildcard doesn't match different resource",
			has:      PermMembersAll,
			required: PermPlansRead,
			want:     false,
		},
		{
			name:     "error - no match different permissions",
			has:      PermMembersRead,
			required: PermMembersCreate,
			want:     false,
		},
		{
			name:     "error - no match different resources",
			has:      PermMembersRead,
			required: PermPlansRead,
			want:     false,
		},
		{
			name:     "error - invalid format (no colon)",
			has:      Permission("invalid"),
			required: PermMembersRead,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchPermission(tt.has, tt.required)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name           string
		required       Permission
		role           string
		setRole        bool
		wantStatusCode int
		wantNextCalled bool
	}{
		{
			name:           "success - user has permission",
			required:       PermMembersRead,
			role:           RoleAdmin,
			setRole:        true,
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:           "success - owner has all permissions",
			required:       PermPlansDelete,
			role:           RoleOwner,
			setRole:        true,
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:           "success - staff has read permission",
			required:       PermMembersRead,
			role:           RoleStaff,
			setRole:        true,
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:           "error - user lacks permission",
			required:       PermMembersDelete,
			role:           RoleStaff,
			setRole:        true,
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "error - no role in context",
			required:       PermMembersRead,
			role:           "",
			setRole:        false,
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "error - empty role",
			required:       PermMembersRead,
			role:           "",
			setRole:        true,
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "error - invalid role",
			required:       PermMembersRead,
			role:           "invalid",
			setRole:        true,
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.setRole {
				c.Set("role", tt.role)
			}

			nextCalled := false
			handler := RequirePermission(tt.required)(func(c echo.Context) error {
				nextCalled = true
				return c.String(http.StatusOK, "success")
			})

			err := handler(c)

			assert.Equal(t, tt.wantNextCalled, nextCalled)

			if tt.wantStatusCode == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				httpErr, ok := err.(*echo.HTTPError)
				assert.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, httpErr.Code)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name           string
		allowedRoles   []string
		userRole       string
		setRole        bool
		wantStatusCode int
		wantNextCalled bool
	}{
		{
			name:           "success - user has allowed role (single)",
			allowedRoles:   []string{RoleAdmin},
			userRole:       RoleAdmin,
			setRole:        true,
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:           "success - user has one of allowed roles",
			allowedRoles:   []string{RoleOwner, RoleAdmin},
			userRole:       RoleAdmin,
			setRole:        true,
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:           "success - owner in allowed roles",
			allowedRoles:   []string{RoleOwner, RoleAdmin, RoleStaff},
			userRole:       RoleOwner,
			setRole:        true,
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:           "success - staff in allowed roles",
			allowedRoles:   []string{RoleStaff},
			userRole:       RoleStaff,
			setRole:        true,
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:           "error - user role not in allowed list",
			allowedRoles:   []string{RoleOwner, RoleAdmin},
			userRole:       RoleStaff,
			setRole:        true,
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "error - no role in context",
			allowedRoles:   []string{RoleAdmin},
			userRole:       "",
			setRole:        false,
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "error - empty role",
			allowedRoles:   []string{RoleAdmin},
			userRole:       "",
			setRole:        true,
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "error - invalid role",
			allowedRoles:   []string{RoleAdmin},
			userRole:       "invalid",
			setRole:        true,
			wantStatusCode: http.StatusForbidden,
			wantNextCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.setRole {
				c.Set("role", tt.userRole)
			}

			nextCalled := false
			handler := RequireRole(tt.allowedRoles...)(func(c echo.Context) error {
				nextCalled = true
				return c.String(http.StatusOK, "success")
			})

			err := handler(c)

			assert.Equal(t, tt.wantNextCalled, nextCalled)

			if tt.wantStatusCode == http.StatusOK {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				httpErr, ok := err.(*echo.HTTPError)
				assert.True(t, ok)
				assert.Equal(t, tt.wantStatusCode, httpErr.Code)
			}
		})
	}
}

func TestHasPermissionWithWildcards(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		required Permission
		want     bool
	}{
		{
			name:     "wildcard - owner has everything via *",
			role:     RoleOwner,
			required: "custom:resource",
			want:     true,
		},
		{
			name:     "wildcard - admin members:* matches members:read",
			role:     RoleAdmin,
			required: PermMembersRead,
			want:     true,
		},
		{
			name:     "wildcard - admin members:* matches members:update",
			role:     RoleAdmin,
			required: PermMembersUpdate,
			want:     true,
		},
		{
			name:     "wildcard - admin members:* matches members:delete",
			role:     RoleAdmin,
			required: PermMembersDelete,
			want:     true,
		},
		{
			name:     "wildcard - admin plans:* matches plans:create",
			role:     RoleAdmin,
			required: PermPlansCreate,
			want:     true,
		},
		{
			name:     "wildcard - admin subscriptions:* matches subscriptions:read",
			role:     RoleAdmin,
			required: PermSubscriptionsRead,
			want:     true,
		},
		{
			name:     "wildcard - staff verification:* matches any verification action",
			role:     RoleStaff,
			required: "verification:create",
			want:     true,
		},
		{
			name:     "wildcard - staff verification:* matches verification:read",
			role:     RoleStaff,
			required: "verification:read",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPermission(tt.role, tt.required)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRoleHierarchy(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		required Permission
		want     bool
	}{
		{
			name:     "hierarchy - owner can do everything",
			role:     RoleOwner,
			required: PermStaffDelete,
			want:     true,
		},
		{
			name:     "hierarchy - admin has more than staff",
			role:     RoleAdmin,
			required: PermMembersDelete,
			want:     true,
		},
		{
			name:     "hierarchy - staff cannot delete members",
			role:     RoleStaff,
			required: PermMembersDelete,
			want:     false,
		},
		{
			name:     "hierarchy - admin cannot delete staff",
			role:     RoleAdmin,
			required: PermStaffDelete,
			want:     false,
		},
		{
			name:     "hierarchy - staff cannot create staff",
			role:     RoleStaff,
			required: PermStaffCreate,
			want:     false,
		},
		// Private session permissions
		{
			name:     "individual sessions - admin can create",
			role:     RoleAdmin,
			required: PermReservationsCreate,
			want:     true,
		},
		{
			name:     "individual sessions - admin can read",
			role:     RoleAdmin,
			required: PermReservationsRead,
			want:     true,
		},
		{
			name:     "individual sessions - admin can update",
			role:     RoleAdmin,
			required: PermReservationsUpdate,
			want:     true,
		},
		{
			name:     "individual sessions - admin cannot delete",
			role:     RoleAdmin,
			required: PermReservationsDelete,
			want:     false,
		},
		{
			name:     "individual sessions - staff can read",
			role:     RoleStaff,
			required: PermReservationsRead,
			want:     true,
		},
		{
			name:     "individual sessions - staff can create",
			role:     RoleStaff,
			required: PermReservationsCreate,
			want:     true,
		},
		{
			name:     "individual sessions - staff cannot update",
			role:     RoleStaff,
			required: PermReservationsUpdate,
			want:     false,
		},
		{
			name:     "individual sessions - staff cannot delete",
			role:     RoleStaff,
			required: PermReservationsDelete,
			want:     false,
		},
		{
			name:     "individual sessions - owner can do everything",
			role:     RoleOwner,
			required: PermReservationsDelete,
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasPermission(tt.role, tt.required)
			assert.Equal(t, tt.want, got)
		})
	}
}
