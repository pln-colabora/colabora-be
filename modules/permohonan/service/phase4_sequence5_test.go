package service

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	documentDTO "github.com/pln-colabora/colabora-be/modules/document/dto"
	documentRepository "github.com/pln-colabora/colabora-be/modules/document/repository"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	userRepository "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func closingReady(t *testing.T, connection string) entities.Permohonan {
	t.Helper()
	p := sequence4ReadyRequest(t, connection, true, true)
	advancePhase4(t, &p, nil, workflow.Energize, workflow.SRAPP)
	return p
}

func TestClosingCompletesEveryConnectionAndRejectsResubmission(t *testing.T) {
	for _, connection := range rbac.AllJenisSambungan {
		t.Run(connection, func(t *testing.T) {
			p := closingReady(t, connection)
			repo := &phase3PermohonanRepository{byID: p}
			docs := &phase4DocumentRepository{}
			user := entities.User{ID: uuid.New(), Role: rbac.RolePelayananPelanggan, Unit: p.UlpUnit}
			s := &permohonanService{db: phase3DB(t), permohonanRepository: repo, documentRepository: docs, userRepository: &phase3UserRepository{user: user}}
			result, err := s.SubmitClosing(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
			require.NoError(t, err)
			require.Equal(t, string(workflow.Finished), result.Status)
			require.EqualValues(t, 7, result.CurrentStage)
			require.Empty(t, result.AvailableActions)
			codes := []workflow.Code{workflow.PDL, workflow.AIL, workflow.Selesai}
			require.Equal(t, codes, docs.attachCalls)
			require.Len(t, repo.logs, 3)
			for i, code := range codes {
				node := responseNode(t, result, code)
				require.Equal(t, string(workflow.Completed), node.Status)
				require.Equal(t, user.ID.String(), *node.CompletedBy)
				require.NotNil(t, node.CompletedAt)
				require.Equal(t, string(code), *repo.logs[i].WorkflowNode)
				require.Equal(t, "node_completed", repo.logs[i].Action)
			}
			_, err = s.SubmitClosing(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
			require.ErrorIs(t, err, workflow.ErrNotActionable)
			require.Equal(t, 1, repo.saveCalls)
		})
	}
}

func TestClosingRejectsOwnersPrerequisitesAndReturnedRequests(t *testing.T) {
	for _, scenario := range []string{"wrong role", "wrong ULP", "super user", "missing energize", "missing SRAPP", "returned"} {
		t.Run(scenario, func(t *testing.T) {
			p := closingReady(t, rbac.JenisSambunganPlgTmLebih5)
			user := entities.User{ID: uuid.New(), Role: rbac.RolePelayananPelanggan, Unit: p.UlpUnit}
			want := workflow.ErrNotActionable
			switch scenario {
			case "wrong role":
				user.Role = rbac.RoleNps
				want = rbac.ErrWorkflowForbidden
			case "wrong ULP":
				user.Unit = "ULP Lain"
				want = rbac.ErrWorkflowForbidden
			case "super user":
				user.Role = rbac.RoleSuperUser
				want = rbac.ErrWorkflowForbidden
			case "missing energize", "missing SRAPP":
				p = sequence4ReadyRequest(t, p.JenisSambungan, false, false)
				code := workflow.Energize
				if scenario == "missing energize" {
					code = workflow.SRAPP
				}
				advancePhase4(t, &p, nil, code)
			case "returned":
				p = phase4Request(t, p.JenisSambungan)
				poles := false
				advancePhase4(t, &p, &workflow.Decisions{KebutuhanTiang: &poles, NPS: workflow.Return},
					workflow.Survei, workflow.RAB, workflow.KebutuhanTiang, workflow.Perluasan, workflow.NPS)
			}
			repo := &phase3PermohonanRepository{byID: p}
			docs := &phase4DocumentRepository{}
			s := &permohonanService{db: phase3DB(t), permohonanRepository: repo, documentRepository: docs, userRepository: &phase3UserRepository{user: user}}
			_, err := s.SubmitClosing(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
			require.ErrorIs(t, err, want)
			require.Empty(t, docs.attachCalls)
			require.Zero(t, repo.saveCalls)
		})
	}
}

func TestClosingValidatesBeforeTransaction(t *testing.T) {
	s := &permohonanService{}
	_, err := s.SubmitClosing(context.Background(), "", "", dto.EvidenceSubmitRequest{})
	require.ErrorIs(t, err, dto.ErrEvidenceRequired)
	_, err = s.SubmitClosing(context.Background(), "", "", dto.EvidenceSubmitRequest{Notes: strings.Repeat("a", 2001)})
	require.ErrorIs(t, err, dto.ErrInvalidActivity)
	id := uuid.NewString()
	_, err = s.SubmitClosing(context.Background(), "", "", dto.EvidenceSubmitRequest{DocumentIDs: []string{id, id}})
	require.ErrorIs(t, err, dto.ErrInvalidEvidence)
}

func TestClosingDatabaseAtomicity(t *testing.T) {
	for _, scenario := range []string{"success", "last evidence conflict", "audit failure"} {
		t.Run(scenario, func(t *testing.T) {
			db := phase4IntegrationDB(t)
			p := closingReady(t, rbac.JenisSambunganJTR)
			user := entities.User{ID: uuid.New(), Name: "Synthetic closer", Email: uuid.NewString() + "@example.test", Role: rbac.RolePelayananPelanggan, Unit: p.UlpUnit}
			require.NoError(t, db.Create(&user).Error)
			require.NoError(t, db.Omit("WorkflowNodes").Create(&p).Error)
			for i := range p.WorkflowNodes {
				p.WorkflowNodes[i].PermohonanID = p.ID
			}
			require.NoError(t, db.Create(&p.WorkflowNodes).Error)
			doc := entities.Document{ID: uuid.New(), Type: "closing", FilePath: "private/synthetic-closing.pdf", UploadedBy: user.ID}
			if scenario == "last evidence conflict" {
				doc.PermohonanID = &p.ID
			}
			require.NoError(t, db.Create(&doc).Error)
			if scenario == "last evidence conflict" {
				require.NoError(t, db.Create(&entities.DocumentEvidence{ID: uuid.New(), DocumentID: doc.ID, PermohonanID: p.ID, WorkflowNode: string(workflow.Selesai), AttachedBy: user.ID}).Error)
			}
			if scenario == "audit failure" {
				require.NoError(t, db.Exec("CREATE TRIGGER fail_closing_audit BEFORE INSERT ON activity_logs BEGIN SELECT RAISE(ABORT, 'synthetic audit failure'); END").Error)
			}
			s := &permohonanService{db: db, permohonanRepository: repository.NewPermohonanRepository(db), userRepository: userRepository.NewUserRepository(db), documentRepository: documentRepository.NewDocumentRepository(db)}
			_, err := s.SubmitClosing(context.Background(), p.ID.String(), user.ID.String(), dto.EvidenceSubmitRequest{DocumentIDs: []string{doc.ID.String()}, Notes: "synthetic closing"})
			if scenario == "success" {
				require.NoError(t, err)
			} else if scenario == "audit failure" {
				require.ErrorIs(t, err, dto.ErrSubmitActivity)
			} else {
				require.ErrorIs(t, err, documentDTO.ErrDocumentAlreadyAttached)
			}
			var reloaded entities.Permohonan
			require.NoError(t, db.Preload("WorkflowNodes").First(&reloaded, "id = ?", p.ID).Error)
			var evidenceCount, logCount int64
			require.NoError(t, db.Model(&entities.DocumentEvidence{}).Count(&evidenceCount).Error)
			require.NoError(t, db.Model(&entities.ActivityLog{}).Count(&logCount).Error)
			if scenario == "success" {
				require.Equal(t, string(workflow.Finished), reloaded.Status)
				require.EqualValues(t, 3, evidenceCount)
				require.EqualValues(t, 3, logCount)
			} else {
				require.Equal(t, string(workflow.Active), reloaded.Status)
				require.Zero(t, logCount)
				wantCount := int64(0)
				if scenario == "last evidence conflict" {
					wantCount = 1
				}
				require.Equal(t, wantCount, evidenceCount)
				for _, code := range []workflow.Code{workflow.PDL, workflow.AIL, workflow.Selesai} {
					node := persistedNode(t, reloaded, code)
					require.Equal(t, persistedNode(t, p, code).Status, node.Status)
					require.Nil(t, node.CompletedAt)
					require.Nil(t, node.CompletedBy)
				}
				require.NoError(t, db.First(&doc, "id = ?", doc.ID).Error)
				if scenario == "audit failure" {
					require.Nil(t, doc.PermohonanID)
				}
			}
		})
	}
}
