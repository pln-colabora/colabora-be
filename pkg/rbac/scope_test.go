package rbac

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApplyScopeMineUsesActionableWorkflowNodes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE permohonan (id text, jenis_sambungan text, ulp_unit text)").Error)
	require.NoError(t, db.Exec("CREATE TABLE permohonan_activities (permohonan_id text, workflow_node text, status text)").Error)
	require.NoError(t, db.Exec("INSERT INTO permohonan VALUES ('owned', 'JTR', 'ULP Taman'), ('wrong-unit', 'JTR', 'ULP Lain'), ('wrong-family', 'PLG TM <5 GWNG', 'ULP Taman')").Error)
	require.NoError(t, db.Exec("INSERT INTO permohonan_activities VALUES ('owned', 'survei', 'available'), ('wrong-unit', 'survei', 'available'), ('wrong-family', 'survei', 'available')").Error)

	var ids []string
	err = ApplyScopeMine(db.Table("permohonan"), RoleTeknik, "ULP Taman").Select("id").Scan(&ids).Error
	require.NoError(t, err)
	require.Equal(t, []string{"owned"}, ids)

	statement := ApplyScopeMine(db.Session(&gorm.Session{DryRun: true}).Table("permohonan"), RoleTeknik, "ULP Taman").Find(&[]struct{}{}).Statement
	require.NoError(t, statement.Error)
	sql := statement.SQL.String()
	require.Contains(t, sql, "permohonan_activities")
	require.Contains(t, sql, "workflow_node")
	require.Contains(t, sql, "available")
	require.Contains(t, sql, "in_progress")
	require.Contains(t, sql, "permohonan.ulp_unit")
	require.False(t, strings.Contains(sql, "current_stage"))
	require.False(t, strings.Contains(sql, "owner_fn_override"))
}
