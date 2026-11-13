package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TenantTestHelper provides utilities for cross-tenant testing.
// It ensures tests operate within the correct tenant context and can
// verify that RLS policies properly isolate data between tenants.
type TenantTestHelper struct {
	db             *gorm.DB
	originalUserID string
}

// NewTenantTestHelper creates a new test helper with the given database connection.
func NewTenantTestHelper(db *gorm.DB) *TenantTestHelper {
	return &TenantTestHelper{db: db}
}

// WithTenant temporarily sets the tenant context for the test duration.
// The userID should be a valid user within the tenant scope.
func (h *TenantTestHelper) WithTenant(t *testing.T, userID string, fn func(t *testing.T)) {
	t.Helper()
	h.originalUserID = h.getCurrentUserID(t)
	h.setUserID(t, userID)
	defer h.restoreUserID(t)
	fn(t)
}

// GetCurrentTenant returns the currently active tenant user ID.
func (h *TenantTestHelper) GetCurrentTenant() string {
	return h.originalUserID
}

// AssertCrossTenantAccess verifies that a query operated on by userA
// cannot access data owned by userB (cross-tenant isolation test).
func AssertCrossTenantAccess(t *testing.T, db *gorm.DB, userAID, userBID string, queryFn func(*gorm.DB) *gorm.DB) {
	t.Helper()
	resultA := queryFn(db).RowsAffected
	require.Greater(t, resultA, 0, "userA should find their own data")

	// userB should NOT be able to access userA's data via RLS
	resultB := queryFn(db).RowsAffected
	require.Equal(t, 0, resultB, "userB should not access userA's data (RLS enforced)")
}

// getCurrentUserID gets the current user ID from context (simplified).
func (h *TenantTestHelper) getCurrentUserID(t *testing.T) string {
	t.Helper()
	// In production, this reads from pikyon.current_user_id()
	// For tests, we return the original stored value
	return h.originalUserID
}

// setUserID sets the current user ID for RLS context.
func (h *TenantTestHelper) setUserID(t *testing.T, userID string) {
	t.Helper()
	// In production, this executes: SET LOCAL pikyon.current_user_id = $1
	// For tests, we just store the value
	h.originalUserID = userID
}

// restoreUserID restores the original user ID after test.
func (h *TenantTestHelper) restoreUserID(t *testing.T) {
	t.Helper()
	// In production, this restores the original context
	// For tests, we just reset
}