package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRetirePKVendorMigrationPreservesCompletedHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE permohonan_activities (workflow_node TEXT, status TEXT)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO permohonan_activities (workflow_node, status) VALUES
		('pk_vendor', 'completed'), ('pk_vendor', 'available'), ('pk_vendor', 'locked'), ('wo_konstruksi', 'locked')`).Error)

	require.NoError(t, UpRetirePKVendor(db))
	var statuses []string
	require.NoError(t, db.Table("permohonan_activities").Where("workflow_node = ?", "pk_vendor").Order("status").Pluck("status", &statuses).Error)
	require.Equal(t, []string{"completed", "skipped", "skipped"}, statuses)

	require.NoError(t, DownRetirePKVendor(db))
	statuses = nil
	require.NoError(t, db.Table("permohonan_activities").Where("workflow_node = ?", "pk_vendor").Order("status").Pluck("status", &statuses).Error)
	require.Equal(t, []string{"completed", "locked", "locked"}, statuses)
}
