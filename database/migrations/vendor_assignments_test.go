package migrations

import (
	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"os"
	"strings"
	"testing"
)

func TestVendorAssignmentMigrationPostgres(t *testing.T) {
	dsn := os.Getenv("COLABORA_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set COLABORA_TEST_POSTGRES_DSN for isolated PostgreSQL migration verification")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	schema := "test_vendor_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, tx.Exec("CREATE SCHEMA "+schema).Error)
	require.NoError(t, tx.Exec("SET LOCAL search_path TO "+schema+", public").Error)
	require.NoError(t, tx.Exec("CREATE FUNCTION "+schema+".uuid_generate_v4() RETURNS uuid LANGUAGE SQL AS 'SELECT gen_random_uuid()'").Error)
	require.NoError(t, UpVendorAssignments(tx))
	require.True(t, tx.Migrator().HasTable(&entities.VendorAssignment{}))
	require.NoError(t, DownVendorAssignments(tx))
	require.False(t, tx.Migrator().HasTable(&entities.VendorAssignment{}))
	require.NoError(t, UpVendorAssignments(tx))
	var constraints int64
	require.NoError(t, tx.Raw("SELECT count(*) FROM information_schema.table_constraints WHERE table_schema = ? AND table_name = 'vendor_assignments' AND constraint_type = 'FOREIGN KEY'", schema).Scan(&constraints).Error)
	require.EqualValues(t, 3, constraints)
	require.NoError(t, tx.Raw("SELECT count(*) FROM information_schema.table_constraints WHERE table_schema = ? AND table_name = 'vendor_assignments' AND constraint_type = 'PRIMARY KEY'", schema).Scan(&constraints).Error)
	require.EqualValues(t, 1, constraints)
}
