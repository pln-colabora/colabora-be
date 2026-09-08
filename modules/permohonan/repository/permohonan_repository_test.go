package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSaveWorkflowPersistsProjectionNodesAndLogs(t *testing.T) {
	db := workflowRepositoryDB(t)
	repo := NewPermohonanRepository(db)
	p := entities.Permohonan{
		ID: uuid.New(), NoPermohonan: "PBPD-2026-TEST", JenisPermohonan: "Pasang Baru (PB)",
		JenisSambungan: "JTR", UlpUnit: "ULP Taman", PelangganNama: "Pelanggan Uji",
		PelangganAlamat: "Alamat sintetis", PelangganNoHp: "0800000000", RequestDate: time.Now(),
		CurrentStage: 2, Status: string(workflow.Active), CreatedBy: uuid.New(),
	}
	node := entities.PermohonanActivity{
		ID: uuid.New(), PermohonanID: p.ID, WorkflowNode: string(workflow.Survei), StageNumber: 2,
		Status: string(workflow.Available), Payload: "{}",
	}
	require.NoError(t, db.Create(&p).Error)
	require.NoError(t, db.Create(&node).Error)

	loaded, err := repo.GetByIdForUpdate(context.Background(), db, p.ID.String())
	require.NoError(t, err)
	require.Len(t, loaded.WorkflowNodes, 1)
	poleRequired := true
	actor := uuid.New()
	now := time.Now()
	loaded.CurrentStage = 3
	loaded.KebutuhanTiang = &poleRequired
	loaded.WorkflowNodes[0].Status = string(workflow.Completed)
	loaded.WorkflowNodes[0].Payload = `{"surveyed_at":"2026-09-08"}`
	loaded.WorkflowNodes[0].CompletedBy = &actor
	loaded.WorkflowNodes[0].CompletedAt = &now
	nodeCode := string(workflow.Survei)
	logs := []entities.ActivityLog{{
		ID: uuid.New(), PermohonanID: p.ID, Actor: actor, Action: "node_completed", WorkflowNode: &nodeCode,
	}}

	require.NoError(t, repo.SaveWorkflow(context.Background(), db, loaded, logs))
	var saved entities.Permohonan
	require.NoError(t, db.First(&saved, "id = ?", p.ID).Error)
	require.Equal(t, int16(3), saved.CurrentStage)
	require.NotNil(t, saved.KebutuhanTiang)
	require.True(t, *saved.KebutuhanTiang)
	var savedNode entities.PermohonanActivity
	require.NoError(t, db.First(&savedNode, "id = ?", node.ID).Error)
	require.Equal(t, string(workflow.Completed), savedNode.Status)
	require.Equal(t, `{"surveyed_at":"2026-09-08"}`, savedNode.Payload)
	var logCount int64
	require.NoError(t, db.Model(&entities.ActivityLog{}).Count(&logCount).Error)
	require.EqualValues(t, 1, logCount)
}

func workflowRepositoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	statements := []string{
		`CREATE TABLE permohonan (id TEXT PRIMARY KEY, no_permohonan TEXT, jenis_permohonan TEXT, jenis_sambungan TEXT, ulp_unit TEXT, pelanggan_nama TEXT, pelanggan_alamat TEXT, pelanggan_no_hp TEXT, request_date DATETIME, current_stage INTEGER, status TEXT, kebutuhan_tiang NUMERIC, nps_delegation_status TEXT, perlu_pdkb NUMERIC, created_by TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE permohonan_activities (id TEXT PRIMARY KEY, permohonan_id TEXT, workflow_node TEXT, activity_number INTEGER, stage_number INTEGER, status TEXT, sla_deadline DATETIME, payload TEXT, completed_by TEXT, completed_at DATETIME, created_at DATETIME, updated_at DATETIME, UNIQUE(permohonan_id, workflow_node))`,
		`CREATE TABLE activity_logs (id TEXT PRIMARY KEY, permohonan_id TEXT, activity_number INTEGER, actor TEXT, action TEXT, detail TEXT, workflow_node TEXT, created_at DATETIME, updated_at DATETIME)`,
	}
	for _, statement := range statements {
		require.NoError(t, db.Exec(statement).Error)
	}
	return db
}
