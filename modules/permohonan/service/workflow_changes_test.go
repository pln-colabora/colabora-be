package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	documentRepository "github.com/pln-colabora/colabora-be/modules/document/repository"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestPDKBCalendarOwnershipConflictAndLockedBookings(t *testing.T) {
	db := phase4IntegrationDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE pdkb_schedules (id TEXT PRIMARY KEY, schedule_date DATE NOT NULL UNIQUE, title TEXT, permohonan_id TEXT UNIQUE, created_by TEXT, updated_by TEXT, created_at DATETIME, updated_at DATETIME)`).Error)
	actor := entities.User{ID: uuid.New(), Role: rbac.RolePdkb}
	users := &phase3UserRepository{user: actor}
	s := &permohonanService{db: db, userRepository: users}
	ctx := context.Background()
	date := time.Now().In(wib).AddDate(0, 0, 7).Format("2006-01-02")
	row, err := s.SavePDKBSchedule(ctx, "", actor.ID.String(), dto.PDKBScheduleRequest{Date: date, Title: "Kegiatan sintetis"})
	require.NoError(t, err)
	_, err = s.SavePDKBSchedule(ctx, "", actor.ID.String(), dto.PDKBScheduleRequest{Date: date, Title: "Bentrok"})
	require.ErrorIs(t, err, dto.ErrScheduleConflict)
	users.user.Role = rbac.RoleKonstruksi
	rows, err := s.ListPDKBSchedules(ctx, actor.ID.String(), date, date)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	_, err = s.SavePDKBSchedule(ctx, row.ID, actor.ID.String(), dto.PDKBScheduleRequest{Date: date, Title: "Tidak berhak"})
	require.ErrorIs(t, err, rbac.ErrWorkflowForbidden)
	users.user.Role = rbac.RolePdkb
	require.NoError(t, s.DeletePDKBSchedule(ctx, row.ID, actor.ID.String()))
	booking := entities.PDKBSchedule{ID: uuid.New(), ScheduleDate: time.Now().In(wib).AddDate(0, 0, 7), Title: "WO terkunci", PermohonanID: func() *uuid.UUID { id := uuid.New(); return &id }(), CreatedBy: actor.ID, UpdatedBy: actor.ID}
	require.NoError(t, db.Create(&booking).Error)
	require.ErrorIs(t, s.DeletePDKBSchedule(ctx, booking.ID.String(), actor.ID.String()), dto.ErrScheduleLocked)
	_, err = s.SavePDKBSchedule(ctx, booking.ID.String(), actor.ID.String(), dto.PDKBScheduleRequest{Date: date, Title: "Tidak boleh"})
	require.ErrorIs(t, err, dto.ErrScheduleLocked)
}

func TestSurveyRevisionPreservesClosedWorkflowAndSharedEvidence(t *testing.T) {
	db := phase4IntegrationDB(t)
	require.NoError(t, db.AutoMigrate(&entities.ActivityRevision{}))
	p := closingReady(t, rbac.JenisSambunganJTR)
	advancePhase4(t, &p, nil, workflow.PDL, workflow.AIL, workflow.Selesai)
	actor := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: p.UlpUnit}
	completed := time.Date(2026, 9, 25, 8, 0, 0, 0, wib)
	for i := range p.WorkflowNodes {
		if p.WorkflowNodes[i].WorkflowNode == string(workflow.Survei) {
			p.WorkflowNodes[i].Payload = `{"surveyed_at":"2026-09-25","notes":"awal"}`
			p.WorkflowNodes[i].CompletedAt = &completed
			p.WorkflowNodes[i].CompletedBy = &actor.ID
			p.WorkflowNodes[i].Revision = 1
		}
	}
	require.NoError(t, db.Omit("WorkflowNodes").Create(&p).Error)
	require.NoError(t, db.Create(&p.WorkflowNodes).Error)
	docs := documentRepository.NewDocumentRepository(db)
	old := entities.Document{ID: uuid.New(), UploadedBy: actor.ID, Source: "uploaded", Type: "evidence", PermohonanID: &p.ID, FilePath: "synthetic/old.pdf"}
	replacement := entities.Document{ID: uuid.New(), UploadedBy: actor.ID, Source: "uploaded", Type: "izin_pemilik_tanah", FilePath: "synthetic/new.pdf"}
	require.NoError(t, db.Create(&old).Error)
	require.NoError(t, db.Create(&replacement).Error)
	for _, code := range []workflow.Code{workflow.Survei, workflow.RAB} {
		_, err := docs.AttachToWorkflowNode(context.Background(), db, []string{old.ID.String()}, p.ID.String(), string(code), actor.ID.String())
		require.NoError(t, err)
	}
	s := &permohonanService{db: db, permohonanRepository: repository.NewPermohonanRepository(db), userRepository: &phase3UserRepository{user: actor}, documentRepository: docs}
	req := dto.SurveyRevisionRequest{SurveyedAt: "2026-09-24", Notes: "revisi", DocumentIDs: []string{replacement.ID.String()}, ExpectedRevision: 1}
	result, err := s.EditSurvey(context.Background(), p.ID.String(), actor.ID.String(), req)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	node := responseNode(t, result, workflow.Survei)
	require.Equal(t, 2, node.Revision)
	require.Equal(t, completed.Format(time.RFC3339), *node.CompletedAt)
	require.Equal(t, actor.ID.String(), *node.CompletedBy)
	require.Equal(t, persistedNode(t, p, workflow.Survei).SlaDeadline.Format("2006-01-02"), *node.SlaDeadline)
	_, err = s.EditSurvey(context.Background(), p.ID.String(), actor.ID.String(), req)
	require.ErrorIs(t, err, dto.ErrRevisionConflict)
	for _, test := range []struct {
		code string
		want uuid.UUID
	}{{string(workflow.Survei), replacement.ID}, {string(workflow.RAB), old.ID}} {
		rows, err := docs.ListByPermohonan(context.Background(), db, p.ID.String(), &test.code)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, test.want, rows[0].ID)
	}
	history, err := s.GetRevisions(context.Background(), p.ID.String(), string(workflow.Survei), actor.ID.String())
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, []string{old.ID.String()}, history[0].DocumentIDs)
	users := s.userRepository.(*phase3UserRepository)
	users.user.Unit = "ULP Lain"
	req.ExpectedRevision = 2
	_, err = s.EditSurvey(context.Background(), p.ID.String(), actor.ID.String(), req)
	require.ErrorIs(t, err, rbac.ErrWorkflowForbidden)
}

func TestPriorityGatesNPSAndRequiresPlanningOwner(t *testing.T) {
	p := phase4Request(t, rbac.JenisSambunganJTR)
	required := false
	advancePhase4(t, &p, &workflow.Decisions{KebutuhanTiang: &required}, workflow.Survei, workflow.RAB, workflow.KebutuhanTiang)
	repo := &phase3PermohonanRepository{byID: p}
	actor := entities.User{ID: uuid.New(), Role: rbac.RoleNps}
	users := &phase3UserRepository{user: actor}
	s := &permohonanService{db: phase3DB(t), permohonanRepository: repo, userRepository: users, documentRepository: &phase4DocumentRepository{}}
	_, err := s.SubmitExpansion(context.Background(), p.ID.String(), actor.ID.String(), dto.ExpansionSubmitRequest{NpsDelegationStatus: "delegated", DocumentIDs: []string{uuid.NewString()}})
	require.ErrorIs(t, err, workflow.ErrNotActionable)
	_, err = s.SubmitPriority(context.Background(), p.ID.String(), actor.ID.String(), dto.PrioritySubmitRequest{})
	require.ErrorIs(t, err, rbac.ErrWorkflowForbidden)
	users.user.Role = rbac.RolePerencanaan
	result, err := s.SubmitPriority(context.Background(), p.ID.String(), actor.ID.String(), dto.PrioritySubmitRequest{Notes: "Terverifikasi"})
	require.NoError(t, err)
	require.Equal(t, "available", responseNode(t, result, workflow.Perluasan).Status)
}
