package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	documentDTO "github.com/pln-colabora/colabora-be/modules/document/dto"
	documentRepository "github.com/pln-colabora/colabora-be/modules/document/repository"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	permohonanRepository "github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	userRepository "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestStage4ParallelWorkOrdersCanCompleteInEitherOrder(t *testing.T) {
	orders := []struct {
		name  string
		nodes []workflow.Code
	}{
		{name: "pole construction app", nodes: []workflow.Code{workflow.WOTiang, workflow.WOKonstruksi, workflow.WOAPP}},
		{name: "app construction pole", nodes: []workflow.Code{workflow.WOAPP, workflow.WOKonstruksi, workflow.WOTiang}},
	}
	for _, test := range orders {
		t.Run(test.name, func(t *testing.T) {
			p := delegatedRequest(t, rbac.JenisSambunganJTR, true)
			repo := &phase3PermohonanRepository{byID: p}
			docRepo := &phase4DocumentRepository{}
			users := map[workflow.Code]entities.User{
				workflow.WOTiang:      {ID: uuid.New(), Role: rbac.RolePerencanaan, Unit: "UP3"},
				workflow.WOKonstruksi: {ID: uuid.New(), Role: rbac.RoleKonstruksi, Unit: "UP3"},
				workflow.WOAPP:        {ID: uuid.New(), Role: rbac.RoleTransaksiEnergi, Unit: "UP3"},
			}
			s := &permohonanService{permohonanRepository: repo, documentRepository: docRepo, db: phase3DB(t)}
			pdkbRequired := false

			for _, code := range test.nodes {
				user := users[code]
				s.userRepository = &phase3UserRepository{user: user}
				var err error
				switch code {
				case workflow.WOTiang:
					_, err = s.SubmitWOTiang(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
				case workflow.WOKonstruksi:
					_, err = s.SubmitWOConstruction(context.Background(), p.ID.String(), user.ID.String(), dto.WOConstructionSubmitRequest{
						PerluPdkb: &pdkbRequired, DocumentIDs: []string{uuid.NewString()},
					})
				case workflow.WOAPP:
					_, err = s.SubmitWOAPP(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
				}
				require.NoError(t, err)
			}

			require.Equal(t, string(workflow.Completed), persistedNode(t, repo.byID, workflow.WOTiang).Status)
			require.Equal(t, string(workflow.Completed), persistedNode(t, repo.byID, workflow.WOKonstruksi).Status)
			require.Equal(t, string(workflow.Completed), persistedNode(t, repo.byID, workflow.WOAPP).Status)
			require.Equal(t, string(workflow.Available), persistedNode(t, repo.byID, workflow.PemasanganTiang).Status)
			require.Equal(t, string(workflow.Available), persistedNode(t, repo.byID, workflow.Konstruksi).Status)
			require.Equal(t, string(workflow.Available), persistedNode(t, repo.byID, workflow.Reservasi).Status)
		})
	}
}

func TestWOTiangRejectsWrongOwnerAndInapplicableBranch(t *testing.T) {
	t.Run("wrong owner", func(t *testing.T) {
		p := delegatedRequest(t, rbac.JenisSambunganPlgTmKurang5, true)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		user := entities.User{ID: uuid.New(), Role: rbac.RoleKonstruksi, Unit: "UP3"}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitWOTiang(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
		require.ErrorIs(t, err, rbac.ErrWorkflowForbidden)
		require.Empty(t, docRepo.attachCalls)
	})

	t.Run("pole branch skipped", func(t *testing.T) {
		p := delegatedRequest(t, rbac.JenisSambunganJTMGardu, false)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		user := entities.User{ID: uuid.New(), Role: rbac.RolePerencanaan, Unit: "UP3"}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitWOTiang(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
		require.ErrorIs(t, err, workflow.ErrNotActionable)
		require.Empty(t, docRepo.attachCalls)
	})
}

func TestWOConstructionControlsPDKBAndAuditsDerivedSkips(t *testing.T) {
	tests := []struct {
		name             string
		connection       string
		required         bool
		wantWOPDKB       workflow.Status
		wantConstruction workflow.Status
		wantLogSize      int
		wantAction       string
	}{
		{name: "JTR requires PDKB", connection: rbac.JenisSambunganJTR, required: true, wantWOPDKB: workflow.Available, wantConstruction: workflow.Locked, wantLogSize: 1, wantAction: "pdkb_required"},
		{name: "PLG TM skips PDKB", connection: rbac.JenisSambunganPlgTmLebih5, required: false, wantWOPDKB: workflow.Skipped, wantConstruction: workflow.Available, wantLogSize: 3, wantAction: "pdkb_not_required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := delegatedRequest(t, test.connection, false)
			repo := &phase3PermohonanRepository{byID: p}
			docRepo := &phase4DocumentRepository{}
			user := entities.User{ID: uuid.New(), Role: rbac.RoleKonstruksi, Unit: "UP3"}
			s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

			response, err := s.SubmitWOConstruction(context.Background(), p.ID.String(), user.ID.String(), dto.WOConstructionSubmitRequest{
				PerluPdkb: &test.required, Notes: "synthetic work order", DocumentIDs: []string{uuid.NewString()},
			})
			require.NoError(t, err)
			require.NotNil(t, response.PerluPdkb)
			require.Equal(t, test.required, *response.PerluPdkb)
			require.Equal(t, string(test.wantWOPDKB), responseNode(t, response, workflow.WOPDKB).Status)
			require.Equal(t, string(test.wantConstruction), responseNode(t, response, workflow.Konstruksi).Status)
			if test.required {
				require.Len(t, response.AvailableActions, 1)
				require.Equal(t, string(workflow.WOPDKB), response.AvailableActions[0].WorkflowNode)
			} else {
				require.Empty(t, response.AvailableActions)
			}
			require.Len(t, repo.logs, test.wantLogSize)
			require.Equal(t, test.wantAction, repo.logs[0].Action)
			if !test.required {
				require.Equal(t, "node_skipped", repo.logs[1].Action)
				require.Equal(t, "node_skipped", repo.logs[2].Action)
			}

			_, err = s.SubmitWOConstruction(context.Background(), p.ID.String(), user.ID.String(), dto.WOConstructionSubmitRequest{
				PerluPdkb: &test.required, DocumentIDs: []string{uuid.NewString()},
			})
			require.Error(t, err)
			require.Len(t, docRepo.attachCalls, 1)
		})
	}
}

func TestWOPDKBMustPrecedeConstructionWhenRequired(t *testing.T) {
	p := delegatedRequest(t, rbac.JenisSambunganJTR, false)
	pdkbRequired := true
	advancePhase4(t, &p, &workflow.Decisions{PerluPDKB: &pdkbRequired}, workflow.WOKonstruksi)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{}
	user := entities.User{ID: uuid.New(), Role: rbac.RoleVendorKonstruksi, Unit: "vendor"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

	_, err := s.SubmitConstructionExecution(context.Background(), p.ID.String(), user.ID.String(), executionRequest(workflow.Konstruksi))
	require.ErrorIs(t, err, workflow.ErrNotActionable)
	require.Empty(t, docRepo.attachCalls)

	construction := entities.User{ID: uuid.New(), Role: rbac.RoleKonstruksi, Unit: "UP3"}
	s.userRepository = &phase3UserRepository{user: construction}
	response, err := s.SubmitWOPDKB(context.Background(), p.ID.String(), construction.ID.String(), evidenceRequest())
	require.NoError(t, err)
	require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.WOPDKB).Status)
	require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.Konstruksi).Status)

	vendor := entities.User{ID: uuid.New(), Role: rbac.RoleVendorKonstruksi, Unit: "vendor"}
	s.userRepository = &phase3UserRepository{user: vendor}
	response, err = s.SubmitConstructionExecution(context.Background(), p.ID.String(), vendor.ID.String(), executionRequest(workflow.Konstruksi))
	require.NoError(t, err)
	require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.Konstruksi).Status)
}

func TestConstructionOpensDirectlyWhenPDKBNotRequired(t *testing.T) {
	p := delegatedRequest(t, rbac.JenisSambunganPlgTmKurang5, false)
	pdkbRequired := false
	advancePhase4(t, &p, &workflow.Decisions{PerluPDKB: &pdkbRequired}, workflow.WOKonstruksi)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{}
	construction := entities.User{ID: uuid.New(), Role: rbac.RoleKonstruksi, Unit: "UP3"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: construction}, documentRepository: docRepo, db: phase3DB(t)}

	_, err := s.SubmitWOPDKB(context.Background(), p.ID.String(), construction.ID.String(), evidenceRequest())
	require.ErrorIs(t, err, workflow.ErrNotActionable)
	vendor := entities.User{ID: uuid.New(), Role: rbac.RoleVendorKonstruksi, Unit: "vendor"}
	s.userRepository = &phase3UserRepository{user: vendor}
	response, err := s.SubmitConstructionExecution(context.Background(), p.ID.String(), vendor.ID.String(), executionRequest(workflow.Konstruksi))
	require.NoError(t, err)
	require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.Konstruksi).Status)
}

func TestWOAPPThenReservationAndTeraBundle(t *testing.T) {
	p := delegatedRequest(t, rbac.JenisSambunganPlgTmLebih5, false)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{}
	user := entities.User{ID: uuid.New(), Role: rbac.RoleTransaksiEnergi, Unit: "UP3"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}
	reservation := dto.ReservationTeraSubmitRequest{
		ReservationNotes: "synthetic material reservation", TeraNotes: "synthetic tera evidence", DocumentIDs: []string{uuid.NewString()},
	}

	_, err := s.SubmitReservationTera(context.Background(), p.ID.String(), user.ID.String(), reservation)
	require.ErrorIs(t, err, workflow.ErrNotActionable)
	require.Empty(t, docRepo.attachCalls)

	response, err := s.SubmitWOAPP(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
	require.NoError(t, err)
	require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.Reservasi).Status)

	response, err = s.SubmitReservationTera(context.Background(), p.ID.String(), user.ID.String(), reservation)
	require.NoError(t, err)
	require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.Reservasi).Status)
	require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.Tera).Status)
	require.Equal(t, []workflow.Code{workflow.WOAPP, workflow.Reservasi, workflow.Tera}, docRepo.attachCalls)

	_, err = s.SubmitReservationTera(context.Background(), p.ID.String(), user.ID.String(), reservation)
	require.Error(t, err)
	require.Len(t, docRepo.attachCalls, 3)
}

func TestReservationTeraEvidenceFailureRollsBackBothNodes(t *testing.T) {
	p := delegatedRequest(t, rbac.JenisSambunganJTR, false)
	advancePhase4(t, &p, nil, workflow.WOAPP)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{failAt: 2}
	user := entities.User{ID: uuid.New(), Role: rbac.RoleTransaksiEnergi, Unit: "UP3"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

	_, err := s.SubmitReservationTera(context.Background(), p.ID.String(), user.ID.String(), dto.ReservationTeraSubmitRequest{
		DocumentIDs: []string{uuid.NewString()},
	})
	require.ErrorIs(t, err, documentDTO.ErrDocumentAlreadyAttached)
	require.Zero(t, repo.saveCalls)
	require.Equal(t, string(workflow.Available), persistedNode(t, repo.byID, workflow.Reservasi).Status)
	require.Equal(t, string(workflow.Locked), persistedNode(t, repo.byID, workflow.Tera).Status)
}

func TestReservationTeraSecondAttachmentConflictRollsBackDatabase(t *testing.T) {
	db := phase4IntegrationDB(t)
	user := entities.User{ID: uuid.New(), Name: "Transaksi Energi", Email: uuid.NewString() + "@example.test", Role: rbac.RoleTransaksiEnergi, Unit: "UP3"}
	require.NoError(t, db.Create(&user).Error)
	p := delegatedRequest(t, rbac.JenisSambunganJTR, false)
	advancePhase4(t, &p, nil, workflow.WOAPP)
	require.NoError(t, db.Omit("WorkflowNodes").Create(&p).Error)
	for i := range p.WorkflowNodes {
		p.WorkflowNodes[i].PermohonanID = p.ID
	}
	require.NoError(t, db.Create(&p.WorkflowNodes).Error)
	document := entities.Document{
		ID: uuid.New(), Type: "tera", FilePath: "private/tera.pdf", UploadedBy: user.ID, PermohonanID: &p.ID,
	}
	require.NoError(t, db.Create(&document).Error)
	existingEvidence := entities.DocumentEvidence{
		ID: uuid.New(), DocumentID: document.ID, PermohonanID: p.ID,
		WorkflowNode: string(workflow.Tera), AttachedBy: user.ID,
	}
	require.NoError(t, db.Create(&existingEvidence).Error)

	s := &permohonanService{
		permohonanRepository: permohonanRepository.NewPermohonanRepository(db),
		userRepository:       userRepository.NewUserRepository(db),
		documentRepository:   documentRepository.NewDocumentRepository(db),
		db:                   db,
	}
	_, err := s.SubmitReservationTera(context.Background(), p.ID.String(), user.ID.String(), dto.ReservationTeraSubmitRequest{
		DocumentIDs: []string{document.ID.String()},
	})
	require.ErrorIs(t, err, documentDTO.ErrDocumentAlreadyAttached)

	var reservationEvidence int64
	require.NoError(t, db.Model(&entities.DocumentEvidence{}).
		Where("document_id = ? AND workflow_node = ?", document.ID, workflow.Reservasi).
		Count(&reservationEvidence).Error)
	require.Zero(t, reservationEvidence, "first attachment must roll back when the second bundled attachment fails")
	var reservationNode entities.PermohonanActivity
	require.NoError(t, db.Where("permohonan_id = ? AND workflow_node = ?", p.ID, workflow.Reservasi).First(&reservationNode).Error)
	require.Equal(t, string(workflow.Available), reservationNode.Status)
	var teraNode entities.PermohonanActivity
	require.NoError(t, db.Where("permohonan_id = ? AND workflow_node = ?", p.ID, workflow.Tera).First(&teraNode).Error)
	require.Equal(t, string(workflow.Locked), teraNode.Status)
}

func TestReturnedAggregateRejectsStage4Submission(t *testing.T) {
	p := phase4Request(t, rbac.JenisSambunganJTR)
	poleRequired := true
	decisions := workflow.Decisions{KebutuhanTiang: &poleRequired, NPS: workflow.Return}
	advancePhase4(t, &p, &decisions, workflow.Survei, workflow.RAB, workflow.KebutuhanTiang, workflow.Perluasan, workflow.NPS)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{}
	user := entities.User{ID: uuid.New(), Role: rbac.RolePerencanaan, Unit: "UP3"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

	_, err := s.SubmitWOTiang(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
	require.ErrorIs(t, err, workflow.ErrNotActionable)
	require.Empty(t, docRepo.attachCalls)
}

func TestSequence2ExitUnlocksApplicableStage5Branches(t *testing.T) {
	p := delegatedRequest(t, rbac.JenisSambunganPlgTmKurang5, true)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{}
	s := &permohonanService{permohonanRepository: repo, documentRepository: docRepo, db: phase3DB(t)}
	pdkbRequired := false

	planning := entities.User{ID: uuid.New(), Role: rbac.RolePerencanaan, Unit: "UP3"}
	s.userRepository = &phase3UserRepository{user: planning}
	_, err := s.SubmitWOTiang(context.Background(), p.ID.String(), planning.ID.String(), evidenceRequest())
	require.NoError(t, err)

	construction := entities.User{ID: uuid.New(), Role: rbac.RoleKonstruksi, Unit: "UP3"}
	s.userRepository = &phase3UserRepository{user: construction}
	_, err = s.SubmitWOConstruction(context.Background(), p.ID.String(), construction.ID.String(), dto.WOConstructionSubmitRequest{
		PerluPdkb: &pdkbRequired, DocumentIDs: []string{uuid.NewString()},
	})
	require.NoError(t, err)
	vendor := entities.User{ID: uuid.New(), Role: rbac.RoleVendorKonstruksi, Unit: "vendor"}
	s.userRepository = &phase3UserRepository{user: vendor}
	_, err = s.SubmitConstructionExecution(context.Background(), p.ID.String(), vendor.ID.String(), executionRequest(workflow.Konstruksi))
	require.NoError(t, err)

	transactionEnergy := entities.User{ID: uuid.New(), Role: rbac.RoleTransaksiEnergi, Unit: "UP3"}
	s.userRepository = &phase3UserRepository{user: transactionEnergy}
	_, err = s.SubmitWOAPP(context.Background(), p.ID.String(), transactionEnergy.ID.String(), evidenceRequest())
	require.NoError(t, err)
	response, err := s.SubmitReservationTera(context.Background(), p.ID.String(), transactionEnergy.ID.String(), dto.ReservationTeraSubmitRequest{
		DocumentIDs: []string{uuid.NewString()},
	})
	require.NoError(t, err)
	require.Equal(t, int16(5), response.CurrentStage)
	require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.PemasanganTiang).Status)
	require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.Konstruksi).Status)
	require.Equal(t, string(workflow.Skipped), responseNode(t, response, workflow.WOPDKB).Status)
	require.Equal(t, string(workflow.Skipped), responseNode(t, response, workflow.DokumentasiPDKB).Status)
}

func delegatedRequest(t *testing.T, connection string, poleRequired bool) entities.Permohonan {
	t.Helper()
	p := phase4Request(t, connection)
	decisions := workflow.Decisions{KebutuhanTiang: &poleRequired, NPS: workflow.Delegated}
	advancePhase4(t, &p, &decisions, workflow.Survei, workflow.RAB, workflow.KebutuhanTiang, workflow.Perluasan, workflow.NPS)
	return p
}

func evidenceRequest() dto.EvidenceSubmitRequest {
	return dto.EvidenceSubmitRequest{Notes: "synthetic evidence", DocumentIDs: []string{uuid.NewString()}}
}
