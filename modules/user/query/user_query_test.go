package query

import (
	"testing"

	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestVendorFilterOnlyReturnsVendorRoles(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE users (name text, role text)").Error)
	require.NoError(t, db.Exec("INSERT INTO users (name, role) VALUES (?, ?), (?, ?), (?, ?)",
		"Tiang", rbac.RoleVendorTiang,
		"Konstruksi", rbac.RoleVendorKonstruksi,
		"Admin", rbac.RoleAdmin,
	).Error)

	filter := &VendorFilter{}
	var roles []string
	require.NoError(t, filter.ApplyFilters(db.Table("users")).Select("role").Order("role").Scan(&roles).Error)
	require.Equal(t, []string{rbac.RoleVendorKonstruksi, rbac.RoleVendorTiang}, roles)
}

func TestVendorFilterCanRestrictToOneVendorRole(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE users (name text, role text)").Error)
	require.NoError(t, db.Exec("INSERT INTO users (name, role) VALUES (?, ?), (?, ?)",
		"Tiang", rbac.RoleVendorTiang,
		"Konstruksi", rbac.RoleVendorKonstruksi,
	).Error)

	filter := &VendorFilter{Role: rbac.RoleVendorKonstruksi}
	var roles []string
	require.NoError(t, filter.ApplyFilters(db.Table("users")).Select("role").Scan(&roles).Error)
	require.Equal(t, []string{rbac.RoleVendorKonstruksi}, roles)
}
