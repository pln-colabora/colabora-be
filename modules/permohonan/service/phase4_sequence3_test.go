package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	documentDTO "github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestConstructionExecutionRoutesToExactVendorNode(t *testing.T) {
	tests := []struct {
		name       string
		connection string
		pole       bool
		code       workflow.Code
		role       string
	}{
		{name: "JTR pole installation", connection: rbac.JenisSambunganJTR, pole: true, code: workflow.PemasanganTiang, role: rbac.RoleVendorTiang},
		{name: "PLG TM construction", connection: rbac.JenisSambunganPlgTmLebih5, pole: false, code: workflow.Konstruksi, role: rbac.RoleVendorKonstruksi},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := stage5Request(t, test.connection, test.pole, false)
			repo := &phase3PermohonanRepository{byID: p}
			docRepo := &phase4DocumentRepository{}
			user := entities.User{ID: uuid.New(), Role: test.role, Unit: "vendor"}
			s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}
			req := executionRequest(test.code)

			response, err := s.SubmitConstructionExecution(context.Background(), p.ID.String(), user.ID.String(), req)
			require.NoError(t, err)
			require.Equal(t, string(workflow.Completed), responseNode(t, response, test.code).Status)
			require.Equal(t, []workflow.Code{test.code}, docRepo.attachCalls)
			require.Len(t, repo.logs, 1)
			require.Equal(t, string(test.code), *repo.logs[0].WorkflowNode)

			_, err = s.SubmitConstructionExecution(context.Background(), p.ID.String(), user.ID.String(), dto.ConstructionExecutionSubmitRequest{
				WorkflowNode: string(test.code), DocumentIDs: []string{uuid.NewString()},
			})
			require.ErrorIs(t, err, workflow.ErrNotActionable)
			require.Len(t, docRepo.attachCalls, 1)
		})
	}
}

func TestConstructionExecutionRejectsInvalidSelectorWrongOwnerAndInvalidState(t *testing.T) {
	t.Run("invalid selector", func(t *testing.T) {
		s := &permohonanService{}
		_, err := s.SubmitConstructionExecution(context.Background(), uuid.NewString(), uuid.NewString(), dto.ConstructionExecutionSubmitRequest{
			WorkflowNode: string(workflow.WOTiang), DocumentIDs: []string{uuid.NewString()},
		})
		require.ErrorIs(t, err, dto.ErrInvalidActivity)
	})

	t.Run("vendor cannot submit the other vendor node", func(t *testing.T) {
		p := stage5Request(t, rbac.JenisSambunganJTR, true, false)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		user := entities.User{ID: uuid.New(), Role: rbac.RoleVendorKonstruksi, Unit: "vendor"}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitConstructionExecution(context.Background(), p.ID.String(), user.ID.String(), executionRequest(workflow.PemasanganTiang))
		require.ErrorIs(t, err, rbac.ErrWorkflowForbidden)
		require.Empty(t, docRepo.attachCalls)
	})

	t.Run("pole installation is skipped when poles are not required", func(t *testing.T) {
		p := stage5Request(t, rbac.JenisSambunganJTMGardu, false, false)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		user := entities.User{ID: uuid.New(), Role: rbac.RoleVendorTiang, Unit: "vendor"}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitConstructionExecution(context.Background(), p.ID.String(), user.ID.String(), executionRequest(workflow.PemasanganTiang))
		require.ErrorIs(t, err, workflow.ErrNotActionable)
		require.Empty(t, docRepo.attachCalls)
	})

	t.Run("construction waits for WO Konstruksi", func(t *testing.T) {
		p := delegatedRequest(t, rbac.JenisSambunganPlgTmKurang5, false)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		user := entities.User{ID: uuid.New(), Role: rbac.RoleVendorKonstruksi, Unit: "vendor"}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitConstructionExecution(context.Background(), p.ID.String(), user.ID.String(), executionRequest(workflow.Konstruksi))
		require.ErrorIs(t, err, workflow.ErrNotActionable)
		require.Empty(t, docRepo.attachCalls)
	})
}

func TestPoleAndConstructionBranchesCompleteInEitherOrder(t *testing.T) {
	orders := []struct {
		name       string
		connection string
		codes      []workflow.Code
	}{
		{name: "JTR construction then pole", connection: rbac.JenisSambunganJTR, codes: []workflow.Code{workflow.Konstruksi, workflow.PemasanganTiang}},
		{name: "PLG TM pole then construction", connection: rbac.JenisSambunganPlgTmKurang5, codes: []workflow.Code{workflow.PemasanganTiang, workflow.Konstruksi}},
	}
	for _, test := range orders {
		t.Run(test.name, func(t *testing.T) {
			p := stage5Request(t, test.connection, true, false)
			repo := &phase3PermohonanRepository{byID: p}
			docRepo := &phase4DocumentRepository{}
			s := &permohonanService{permohonanRepository: repo, documentRepository: docRepo, db: phase3DB(t)}
			users := map[workflow.Code]entities.User{
				workflow.PemasanganTiang: {ID: uuid.New(), Role: rbac.RoleVendorTiang, Unit: "vendor"},
				workflow.Konstruksi:      {ID: uuid.New(), Role: rbac.RoleVendorKonstruksi, Unit: "vendor"},
			}

			for index, code := range test.codes {
				user := users[code]
				s.userRepository = &phase3UserRepository{user: user}
				response, err := s.SubmitConstructionExecution(context.Background(), p.ID.String(), user.ID.String(), executionRequest(code))
				require.NoError(t, err)
				if index == 0 {
					require.Equal(t, string(workflow.Locked), responseNode(t, response, workflow.Energize).Status)
					if code == workflow.Konstruksi {
						require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.SRAPP).Status)
					}
				} else {
					require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.Energize).Status)
					require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.SRAPP).Status)
				}
			}
		})
	}
}

func TestPDKBDocumentationGatesEnergizeOnlyWhenRequired(t *testing.T) {
	t.Run("required", func(t *testing.T) {
		p := stage5Request(t, rbac.JenisSambunganPlgTmLebih5, false, true)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		pdkbUser := entities.User{ID: uuid.New(), Role: rbac.RolePdkb, Unit: "UP3"}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: pdkbUser}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitPDKBDocumentation(context.Background(), p.ID.String(), pdkbUser.ID.String(), evidenceRequest())
		require.ErrorIs(t, err, workflow.ErrNotActionable)
		require.Empty(t, docRepo.attachCalls)

		vendor := entities.User{ID: uuid.New(), Role: rbac.RoleVendorKonstruksi, Unit: "vendor"}
		s.userRepository = &phase3UserRepository{user: vendor}
		response, err := s.SubmitConstructionExecution(context.Background(), p.ID.String(), vendor.ID.String(), executionRequest(workflow.Konstruksi))
		require.NoError(t, err)
		require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.DokumentasiPDKB).Status)
		require.Equal(t, string(workflow.Locked), responseNode(t, response, workflow.Energize).Status)
		require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.SRAPP).Status)

		wrongUser := entities.User{ID: uuid.New(), Role: rbac.RoleKonstruksi, Unit: "UP3"}
		s.userRepository = &phase3UserRepository{user: wrongUser}
		_, err = s.SubmitPDKBDocumentation(context.Background(), p.ID.String(), wrongUser.ID.String(), evidenceRequest())
		require.ErrorIs(t, err, rbac.ErrWorkflowForbidden)
		require.Len(t, docRepo.attachCalls, 1)

		s.userRepository = &phase3UserRepository{user: pdkbUser}
		response, err = s.SubmitPDKBDocumentation(context.Background(), p.ID.String(), pdkbUser.ID.String(), evidenceRequest())
		require.NoError(t, err)
		require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.DokumentasiPDKB).Status)
		require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.Energize).Status)
	})

	t.Run("not required", func(t *testing.T) {
		p := stage5Request(t, rbac.JenisSambunganJTR, false, false)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		user := entities.User{ID: uuid.New(), Role: rbac.RolePdkb, Unit: "UP3"}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitPDKBDocumentation(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
		require.ErrorIs(t, err, workflow.ErrNotActionable)
		require.Empty(t, docRepo.attachCalls)
	})
}

func TestConstructionExecutionEvidenceFailureDoesNotPersistNode(t *testing.T) {
	p := stage5Request(t, rbac.JenisSambunganJTR, false, false)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{failAt: 1}
	user := entities.User{ID: uuid.New(), Role: rbac.RoleVendorKonstruksi, Unit: "vendor"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

	_, err := s.SubmitConstructionExecution(context.Background(), p.ID.String(), user.ID.String(), executionRequest(workflow.Konstruksi))
	require.ErrorIs(t, err, documentDTO.ErrDocumentAlreadyAttached)
	require.Zero(t, repo.saveCalls)
	require.Equal(t, string(workflow.Available), persistedNode(t, repo.byID, workflow.Konstruksi).Status)
}

func stage5Request(t *testing.T, connection string, poleRequired, pdkbRequired bool) entities.Permohonan {
	t.Helper()
	p := phase4Request(t, connection)
	decisions := workflow.Decisions{
		KebutuhanTiang: &poleRequired,
		PerluPDKB:      &pdkbRequired,
		NPS:            workflow.Delegated,
	}
	codes := []workflow.Code{
		workflow.Survei, workflow.RAB, workflow.KebutuhanTiang, workflow.Perluasan, workflow.NPS,
	}
	if poleRequired {
		codes = append(codes, workflow.WOTiang)
	}
	codes = append(codes, workflow.WOKonstruksi)
	if pdkbRequired {
		codes = append(codes, workflow.WOPDKB)
	}
	codes = append(codes, workflow.WOAPP, workflow.Reservasi, workflow.Tera)
	advancePhase4(t, &p, &decisions, codes...)
	return p
}

func executionRequest(code workflow.Code) dto.ConstructionExecutionSubmitRequest {
	return dto.ConstructionExecutionSubmitRequest{
		WorkflowNode: string(code), Notes: "synthetic execution evidence", DocumentIDs: []string{uuid.NewString()},
	}
}
