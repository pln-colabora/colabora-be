package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	documentDTO "github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestSequenceFourUsesExactConnectionOwners(t *testing.T) {
	tests := []struct {
		name       string
		connection string
		code       workflow.Code
		role       string
	}{
		{name: "JTR energize", connection: rbac.JenisSambunganJTR, code: workflow.Energize, role: rbac.RoleTeknik},
		{name: "PLG TM energize", connection: rbac.JenisSambunganPlgTmLebih5, code: workflow.Energize, role: rbac.RoleJaringan},
		{name: "JTM SR APP", connection: rbac.JenisSambunganJTMGardu, code: workflow.SRAPP, role: rbac.RoleVendorSrApp},
		{name: "PLG TM SR APP", connection: rbac.JenisSambunganPlgTmKurang5, code: workflow.SRAPP, role: rbac.RoleVendorKonstruksi},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := sequence4ReadyRequest(t, test.connection, false, false)
			repo := &phase3PermohonanRepository{byID: p}
			docRepo := &phase4DocumentRepository{}
			unit := "UP3"
			if test.role == rbac.RoleTeknik {
				unit = p.UlpUnit
			}
			user := entities.User{ID: uuid.New(), Role: test.role, Unit: unit}
			s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

			var response dto.PermohonanResponse
			var err error
			if test.code == workflow.Energize {
				response, err = s.SubmitEnergize(context.Background(), p.ID.String(), user.ID.String(), energizeRequest())
			} else {
				response, err = s.SubmitSRAPP(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
			}

			require.NoError(t, err)
			require.Equal(t, string(workflow.Completed), responseNode(t, response, test.code).Status)
			require.Equal(t, []workflow.Code{test.code}, docRepo.attachCalls)
			require.Len(t, repo.logs, 1)
			require.Equal(t, string(test.code), *repo.logs[0].WorkflowNode)
			if test.code == workflow.Energize {
				var payload map[string]any
				require.NoError(t, json.Unmarshal([]byte(persistedNode(t, repo.byID, test.code).Payload), &payload))
				require.Equal(t, "operasi jaringan selesai", payload["operation_result"])
			}
		})
	}
}

func TestSequenceFourRejectsWrongOwnersAndULPScope(t *testing.T) {
	tests := []struct {
		name       string
		connection string
		code       workflow.Code
		role       string
		unit       string
	}{
		{name: "JTR energize wrong ULP", connection: rbac.JenisSambunganJTR, code: workflow.Energize, role: rbac.RoleTeknik, unit: "ULP Lain"},
		{name: "PLG TM energize wrong role", connection: rbac.JenisSambunganPlgTmLebih5, code: workflow.Energize, role: rbac.RoleTeknik, unit: "ULP Taman"},
		{name: "JTR SR APP wrong vendor", connection: rbac.JenisSambunganJTR, code: workflow.SRAPP, role: rbac.RoleVendorKonstruksi, unit: "vendor"},
		{name: "PLG TM SR APP wrong vendor", connection: rbac.JenisSambunganPlgTmKurang5, code: workflow.SRAPP, role: rbac.RoleVendorSrApp, unit: "vendor"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := sequence4ReadyRequest(t, test.connection, false, false)
			repo := &phase3PermohonanRepository{byID: p}
			docRepo := &phase4DocumentRepository{}
			user := entities.User{ID: uuid.New(), Role: test.role, Unit: test.unit}
			s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

			var err error
			if test.code == workflow.Energize {
				_, err = s.SubmitEnergize(context.Background(), p.ID.String(), user.ID.String(), energizeRequest())
			} else {
				_, err = s.SubmitSRAPP(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
			}
			require.ErrorIs(t, err, rbac.ErrWorkflowForbidden)
			require.Empty(t, docRepo.attachCalls)
		})
	}
}

func TestSequenceFourIndependentGates(t *testing.T) {
	t.Run("energize waits for applicable pole work", func(t *testing.T) {
		p := stage5Request(t, rbac.JenisSambunganJTR, true, false)
		advancePhase4(t, &p, nil, workflow.Konstruksi)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: p.UlpUnit}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitEnergize(context.Background(), p.ID.String(), user.ID.String(), energizeRequest())
		require.ErrorIs(t, err, workflow.ErrNotActionable)
		require.Empty(t, docRepo.attachCalls)
		require.Equal(t, string(workflow.Available), persistedNode(t, repo.byID, workflow.SRAPP).Status)
	})

	t.Run("energize waits for required PDKB documentation", func(t *testing.T) {
		p := stage5Request(t, rbac.JenisSambunganPlgTmLebih5, false, true)
		advancePhase4(t, &p, nil, workflow.Konstruksi)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		user := entities.User{ID: uuid.New(), Role: rbac.RoleJaringan, Unit: "UP3"}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitEnergize(context.Background(), p.ID.String(), user.ID.String(), energizeRequest())
		require.ErrorIs(t, err, workflow.ErrNotActionable)
		require.Empty(t, docRepo.attachCalls)
		require.Equal(t, string(workflow.Available), persistedNode(t, repo.byID, workflow.SRAPP).Status)
	})

	t.Run("SR APP waits for construction even when tera is complete", func(t *testing.T) {
		p := stage5Request(t, rbac.JenisSambunganJTMGardu, false, false)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		user := entities.User{ID: uuid.New(), Role: rbac.RoleVendorSrApp, Unit: "vendor"}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitSRAPP(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
		require.ErrorIs(t, err, workflow.ErrNotActionable)
		require.Empty(t, docRepo.attachCalls)
	})

	t.Run("SR APP waits for tera even when construction is complete", func(t *testing.T) {
		p := delegatedRequest(t, rbac.JenisSambunganJTR, false)
		pdkbRequired := false
		advancePhase4(t, &p, &workflow.Decisions{PerluPDKB: &pdkbRequired},
			workflow.WOKonstruksi, workflow.Konstruksi)
		repo := &phase3PermohonanRepository{byID: p}
		docRepo := &phase4DocumentRepository{}
		user := entities.User{ID: uuid.New(), Role: rbac.RoleVendorSrApp, Unit: "vendor"}
		s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

		_, err := s.SubmitSRAPP(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
		require.ErrorIs(t, err, workflow.ErrNotActionable)
		require.Empty(t, docRepo.attachCalls)
	})
}

func TestSequenceFourBranchesCompleteInEitherOrderAndJoinAtPDL(t *testing.T) {
	tests := []struct {
		name       string
		connection string
		order      []workflow.Code
	}{
		{name: "JTR energize then SR APP", connection: rbac.JenisSambunganJTR, order: []workflow.Code{workflow.Energize, workflow.SRAPP}},
		{name: "PLG TM SR APP then energize", connection: rbac.JenisSambunganPlgTmKurang5, order: []workflow.Code{workflow.SRAPP, workflow.Energize}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := sequence4ReadyRequest(t, test.connection, true, true)
			repo := &phase3PermohonanRepository{byID: p}
			docRepo := &phase4DocumentRepository{}
			s := &permohonanService{permohonanRepository: repo, documentRepository: docRepo, db: phase3DB(t)}

			for index, code := range test.order {
				user := sequence4Owner(p, code)
				s.userRepository = &phase3UserRepository{user: user}
				var response dto.PermohonanResponse
				var err error
				if code == workflow.Energize {
					response, err = s.SubmitEnergize(context.Background(), p.ID.String(), user.ID.String(), energizeRequest())
				} else {
					response, err = s.SubmitSRAPP(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
				}
				require.NoError(t, err)
				if index == 0 {
					require.Equal(t, string(workflow.Locked), responseNode(t, response, workflow.PDL).Status)
				} else {
					require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.PDL).Status)
				}
			}

			lastCode := test.order[len(test.order)-1]
			lastUser := sequence4Owner(p, lastCode)
			s.userRepository = &phase3UserRepository{user: lastUser}
			var err error
			if lastCode == workflow.Energize {
				_, err = s.SubmitEnergize(context.Background(), p.ID.String(), lastUser.ID.String(), energizeRequest())
			} else {
				_, err = s.SubmitSRAPP(context.Background(), p.ID.String(), lastUser.ID.String(), evidenceRequest())
			}
			require.ErrorIs(t, err, workflow.ErrNotActionable)
			require.Len(t, docRepo.attachCalls, 2)
		})
	}
}

func TestSequenceFourRejectsInvalidResult(t *testing.T) {
	p := sequence4ReadyRequest(t, rbac.JenisSambunganJTR, false, false)
	user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: p.UlpUnit}
	s := &permohonanService{userRepository: &phase3UserRepository{user: user}}

	_, err := s.SubmitEnergize(context.Background(), p.ID.String(), user.ID.String(), dto.EnergizeSubmitRequest{
		OperationResult: "   ", DocumentIDs: []string{uuid.NewString()},
	})
	require.ErrorIs(t, err, dto.ErrInvalidActivity)
	_, err = s.SubmitEnergize(context.Background(), p.ID.String(), user.ID.String(), dto.EnergizeSubmitRequest{
		OperationResult: strings.Repeat("a", 501), DocumentIDs: []string{uuid.NewString()},
	})
	require.ErrorIs(t, err, dto.ErrInvalidActivity)
}

func TestSequenceFourEvidenceFailureDoesNotPersistEitherBranch(t *testing.T) {
	tests := []struct {
		name string
		code workflow.Code
	}{
		{name: "energize", code: workflow.Energize},
		{name: "SR APP", code: workflow.SRAPP},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := sequence4ReadyRequest(t, rbac.JenisSambunganJTR, false, false)
			repo := &phase3PermohonanRepository{byID: p}
			docRepo := &phase4DocumentRepository{failAt: 1}
			user := sequence4Owner(p, test.code)
			s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

			var err error
			if test.code == workflow.Energize {
				_, err = s.SubmitEnergize(context.Background(), p.ID.String(), user.ID.String(), energizeRequest())
			} else {
				_, err = s.SubmitSRAPP(context.Background(), p.ID.String(), user.ID.String(), evidenceRequest())
			}
			require.ErrorIs(t, err, documentDTO.ErrDocumentAlreadyAttached)
			require.Zero(t, repo.saveCalls)
			require.Equal(t, string(workflow.Available), persistedNode(t, repo.byID, test.code).Status)
		})
	}
}

func sequence4ReadyRequest(t *testing.T, connection string, poleRequired, pdkbRequired bool) entities.Permohonan {
	t.Helper()
	p := stage5Request(t, connection, poleRequired, pdkbRequired)
	codes := []workflow.Code{workflow.Konstruksi}
	if poleRequired {
		codes = append(codes, workflow.PemasanganTiang)
	}
	if pdkbRequired {
		codes = append(codes, workflow.DokumentasiPDKB)
	}
	advancePhase4(t, &p, nil, codes...)
	return p
}

func sequence4Owner(p entities.Permohonan, code workflow.Code) entities.User {
	role, _ := workflow.Owner(code, p.JenisSambungan)
	unit := "UP3"
	if role == rbac.RoleTeknik {
		unit = p.UlpUnit
	}
	return entities.User{ID: uuid.New(), Role: role, Unit: unit}
}

func energizeRequest() dto.EnergizeSubmitRequest {
	return dto.EnergizeSubmitRequest{
		OperationResult: "operasi jaringan selesai", Notes: "synthetic operation evidence", DocumentIDs: []string{uuid.NewString()},
	}
}
