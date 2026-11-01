package migrations

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/businessday"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func migrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("COLABORA_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set COLABORA_TEST_POSTGRES_DSN for isolated PostgreSQL migration verification")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })
	schema := "test_workflow_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, tx.Exec("CREATE SCHEMA "+schema).Error)
	require.NoError(t, tx.Exec("SET LOCAL search_path TO "+schema+", public").Error)
	require.NoError(t, tx.AutoMigrate(&entities.User{}, &entities.Permohonan{}, &entities.PermohonanActivity{}, &entities.Document{}, &entities.DocumentEvidence{}))
	return tx
}

func TestWorkflowUpdatesPreserveLegacyProgressAndDeadlinesPostgres(t *testing.T) {
	db := migrationTestDB(t)
	actor := entities.User{Name: "Synthetic", Email: uuid.NewString() + "@example.test", Role: "teknik", Unit: "ULP Uji"}
	require.NoError(t, db.Create(&actor).Error)
	original := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	for index, scenario := range []string{"before_nps", "delegated", "returned", "completed"} {
		no := false
		p := entities.Permohonan{NoPermohonan: fmt.Sprintf("SYNTHETIC-%d", index), JenisSambungan: "JTR", CreatedBy: actor.ID, RequestDate: original, KebutuhanTiang: &no, PerluPdkb: &no}
		state := workflow.Snapshot{Connection: "JTR", Nodes: map[workflow.Code]workflow.Status{}, Decisions: workflow.Decisions{KebutuhanTiang: &no, PerluPDKB: &no, NPS: workflow.Delegated}}
		if scenario == "returned" {
			state.Decisions.NPS = workflow.Return
		}
		for _, definition := range workflow.Definitions() {
			if scenario == "before_nps" && definition.Code == workflow.Prioritas {
				break
			}
			evaluated, err := workflow.Evaluate(state)
			require.NoError(t, err)
			if evaluated.Nodes[definition.Code] == workflow.Available {
				state, _, err = workflow.Transition(state, definition.Code, workflow.Completed)
				require.NoError(t, err)
			}
			if scenario != "completed" && definition.Code == workflow.NPS {
				break
			}
		}
		result, err := workflow.Evaluate(state)
		require.NoError(t, err)
		p.Status = string(result.Status)
		p.CurrentStage = result.CurrentStage
		if state.Nodes[workflow.NPS] == workflow.Completed {
			value := string(state.Decisions.NPS)
			p.NpsDelegationStatus = &value
		}
		require.NoError(t, db.Omit("WorkflowNodes").Create(&p).Error)
		for _, definition := range workflow.Definitions() {
			if definition.Code == workflow.Prioritas {
				continue
			}
			status := result.Nodes[definition.Code]
			node := entities.PermohonanActivity{PermohonanID: p.ID, WorkflowNode: string(definition.Code), ActivityNumber: definition.ActivityNumber, StageNumber: definition.Stage, Status: string(status), Payload: "{}", SlaDeadline: &original}
			require.NoError(t, db.Create(&node).Error)
		}
		if scenario == "completed" {
			require.NoError(t, db.Create(&entities.PermohonanActivity{PermohonanID: p.ID, WorkflowNode: "wo_app", StageNumber: 4, Status: "completed", Payload: "{}"}).Error)
		}
	}
	require.NoError(t, upWorkflowUpdates(db))
	require.NoError(t, upPaymentWorkdaySLA(db))
	var requests []entities.Permohonan
	require.NoError(t, db.Preload("WorkflowNodes").Order("no_permohonan").Find(&requests).Error)
	require.Len(t, requests, 4)
	for index, p := range requests {
		for _, node := range p.WorkflowNodes {
			if node.SlaDeadline != nil {
				require.Equal(t, "2026-10-26", node.SlaDeadline.Format("2006-01-02"))
			}
			if node.WorkflowNode == "verifikasi_prioritas" {
				if index == 0 {
					require.Equal(t, "available", node.Status)
				} else {
					require.Equal(t, "skipped", node.Status)
				}
			}
			if node.WorkflowNode == "wo_app" {
				require.Equal(t, "completed", node.Status)
			}
		}
	}
	require.Equal(t, "returned", requests[2].Status)
	require.Equal(t, "completed", requests[3].Status)
	require.Error(t, downWorkflowUpdates(db))
}

func TestPaymentWorkdaySQLMatchesCalendarAndRoundTripsPostgres(t *testing.T) {
	db := migrationTestDB(t)
	require.NoError(t, upPaymentWorkdaySLA(db))
	for _, pair := range [][2]string{{"2026-10-23", "2026-10-25"}, {"2026-10-23", "2026-10-26"}, {"2026-06-02", "2026-05-29"}, {"2027-01-04", "2026-12-24"}, {"2027-03-12", "2027-03-09"}} {
		deadline, _ := time.Parse("2006-01-02", pair[0])
		now, _ := time.Parse("2006-01-02", pair[1])
		want, err := businessday.Remaining(deadline, now)
		require.NoError(t, err)
		var got int
		require.NoError(t, db.Raw("SELECT colabora_sla_remaining(?::date,?,?::date)", pair[0], businessday.PaymentBasis, pair[1]).Scan(&got).Error)
		require.Equal(t, want, got)
	}
	require.NoError(t, db.Exec("CREATE TABLE date_roundtrip (day date)").Error)
	payment, _ := time.Parse("2006-01-02", "2026-10-24")
	deadline, err := businessday.Deadline(payment, 2)
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO date_roundtrip(day) VALUES (?)", deadline).Error)
	var actual string
	require.NoError(t, db.Raw("SELECT day::text FROM date_roundtrip").Scan(&actual).Error)
	require.Equal(t, "2026-10-28", actual)
	var unavailable *int
	require.NoError(t, db.Raw("SELECT colabora_sla_remaining('2028-01-01',?,'2027-12-31')", businessday.PaymentBasis).Scan(&unavailable).Error)
	require.Nil(t, unavailable)
	require.NoError(t, downPaymentWorkdaySLA(db))
	require.NoError(t, upPaymentWorkdaySLA(db))
}
