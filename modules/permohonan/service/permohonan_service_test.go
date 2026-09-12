package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	documentDTO "github.com/pln-colabora/colabora-be/modules/document/dto"
	documentRepository "github.com/pln-colabora/colabora-be/modules/document/repository"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	userRepository "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type phase3PermohonanRepository struct {
	created    entities.Permohonan
	activities []entities.PermohonanActivity
	byID       entities.Permohonan
	list       []query.Permohonan
	filter     *query.PermohonanFilter
	saveCalls  int
	logs       []entities.ActivityLog
	getErr     error
	saveErr    error
}

func (f *phase3PermohonanRepository) Create(_ context.Context, _ *gorm.DB, p entities.Permohonan, nodes []entities.PermohonanActivity, _ entities.ActivityLog) (entities.Permohonan, error) {
	p.ID = uuid.New()
	for i := range nodes {
		nodes[i].PermohonanID = p.ID
	}
	p.WorkflowNodes = nodes
	f.created, f.activities = p, nodes
	return p, nil
}
func (f *phase3PermohonanRepository) GetById(_ context.Context, _ *gorm.DB, _ string) (entities.Permohonan, error) {
	return f.byID, nil
}
func (f *phase3PermohonanRepository) GetByIdForUpdate(_ context.Context, _ *gorm.DB, _ string) (entities.Permohonan, error) {
	return f.byID, f.getErr
}
func (f *phase3PermohonanRepository) SaveWorkflow(_ context.Context, _ *gorm.DB, p entities.Permohonan, logs []entities.ActivityLog) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	f.byID = p
	f.logs = append([]entities.ActivityLog(nil), logs...)
	return nil
}

type phase4DocumentRepository struct {
	attachCalls []workflow.Code
	failAt      int
	attachErr   error
}

type failingSavePermohonanRepository struct {
	repository.PermohonanRepository
}

func (f failingSavePermohonanRepository) SaveWorkflow(context.Context, *gorm.DB, entities.Permohonan, []entities.ActivityLog) error {
	return errors.New("forced persistence failure")
}

func (f *phase4DocumentRepository) Create(context.Context, *gorm.DB, entities.Document) (entities.Document, error) {
	return entities.Document{}, nil
}
func (f *phase4DocumentRepository) GetById(context.Context, *gorm.DB, string) (entities.Document, error) {
	return entities.Document{}, nil
}
func (f *phase4DocumentRepository) ListByPermohonan(context.Context, *gorm.DB, string, *string) ([]entities.Document, error) {
	return nil, nil
}
func (f *phase4DocumentRepository) ExistsForWorkflowNode(context.Context, *gorm.DB, string, string) (bool, error) {
	return false, nil
}
func (f *phase4DocumentRepository) AttachToWorkflowNode(_ context.Context, _ *gorm.DB, ids []string, _, node, _ string) (int64, error) {
	f.attachCalls = append(f.attachCalls, workflow.Code(node))
	if f.failAt > 0 && len(f.attachCalls) == f.failAt {
		if f.attachErr != nil {
			return 0, f.attachErr
		}
		return int64(len(ids) - 1), nil
	}
	return int64(len(ids)), nil
}
func (f *phase3PermohonanRepository) List(_ context.Context, _ *gorm.DB, filter *query.PermohonanFilter) ([]query.Permohonan, int64, error) {
	f.filter = filter
	return f.list, int64(len(f.list)), nil
}
func (f *phase3PermohonanRepository) CountByNoPermohonanPrefix(context.Context, *gorm.DB, string) (int64, error) {
	return 0, nil
}
func (f *phase3PermohonanRepository) ListWorkflowNodes(context.Context, *gorm.DB, string) ([]entities.PermohonanActivity, error) {
	return f.byID.WorkflowNodes, nil
}
func (f *phase3PermohonanRepository) ListActivityLogs(context.Context, *gorm.DB, string) ([]entities.ActivityLog, error) {
	return nil, nil
}

type phase3SLARepository struct{}

func (phase3SLARepository) GetByActivityAndJenis(_ context.Context, _ *gorm.DB, activity int16, connection string) (entities.SLARule, error) {
	return entities.SLARule{ActivityNumber: activity, JenisSambungan: connection, OffsetDays: 2}, nil
}
func (phase3SLARepository) ListByJenis(_ context.Context, _ *gorm.DB, connection string) ([]entities.SLARule, error) {
	rules := make([]entities.SLARule, 0)
	for _, definition := range workflow.Definitions() {
		if definition.SLAActivityNumber != nil {
			rules = append(rules, entities.SLARule{ActivityNumber: *definition.SLAActivityNumber, JenisSambungan: connection, OffsetDays: 2})
		}
	}
	return rules, nil
}

type phase3UserRepository struct {
	user          entities.User
	configuredULP bool
}

func (f *phase3UserRepository) Register(context.Context, *gorm.DB, entities.User) (entities.User, error) {
	return entities.User{}, nil
}
func (f *phase3UserRepository) GetUserById(context.Context, *gorm.DB, string) (entities.User, error) {
	return f.user, nil
}
func (f *phase3UserRepository) GetUserByEmail(context.Context, *gorm.DB, string) (entities.User, error) {
	return entities.User{}, nil
}
func (f *phase3UserRepository) CheckEmail(context.Context, *gorm.DB, string) (entities.User, bool, error) {
	return entities.User{}, false, nil
}
func (f *phase3UserRepository) Update(context.Context, *gorm.DB, entities.User) (entities.User, error) {
	return entities.User{}, nil
}
func (f *phase3UserRepository) Delete(context.Context, *gorm.DB, string) error { return nil }
func (f *phase3UserRepository) ExistsByUnitAndRoles(context.Context, *gorm.DB, string, []string) (bool, error) {
	return f.configuredULP, nil
}

func phase3DB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func baseCreateRequest(connection string) dto.PermohonanCreateRequest {
	return dto.PermohonanCreateRequest{
		JenisPermohonan: rbac.JenisPermohonanPasangBaru, JenisSambungan: connection,
		PelangganNama: "Pelanggan Uji", PelangganAlamat: "Alamat sintetis", PelangganNoHp: "0800000000",
	}
}

func TestCreateEntryPathByConnectionFamily(t *testing.T) {
	tests := []struct {
		name       string
		role       string
		callerUnit string
		connection string
		targetULP  *string
		configured bool
		wantULP    string
		wantErr    error
	}{
		{name: "JTR derives ULP from customer service", role: rbac.RolePelayananPelanggan, callerUnit: "ULP Taman", connection: rbac.JenisSambunganJTR, wantULP: "ULP Taman"},
		{name: "PLG TM requires NPS and configured target", role: rbac.RoleNps, callerUnit: "UP3", connection: rbac.JenisSambunganPlgTmKurang5, targetULP: stringPtr("ULP Taman"), configured: true, wantULP: "ULP Taman"},
		{name: "PLG TM rejects customer service", role: rbac.RolePelayananPelanggan, callerUnit: "ULP Taman", connection: rbac.JenisSambunganPlgTmLebih5, targetULP: stringPtr("ULP Taman"), configured: true, wantErr: dto.ErrCreateForbidden},
		{name: "JTR rejects NPS", role: rbac.RoleNps, callerUnit: "UP3", connection: rbac.JenisSambunganJTR, wantErr: dto.ErrCreateForbidden},
		{name: "PLG TM rejects missing target", role: rbac.RoleNps, callerUnit: "UP3", connection: rbac.JenisSambunganPlgTmKurang5, wantErr: dto.ErrInvalidULPUnit},
		{name: "PLG TM rejects unknown target", role: rbac.RoleNps, callerUnit: "UP3", connection: rbac.JenisSambunganPlgTmKurang5, targetULP: stringPtr("ULP Unknown"), wantErr: dto.ErrInvalidULPUnit},
		{name: "JTR rejects supplied target", role: rbac.RolePelayananPelanggan, callerUnit: "ULP Taman", connection: rbac.JenisSambunganJTR, targetULP: stringPtr("ULP Taman"), wantErr: dto.ErrInvalidULPUnit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			permohonanRepo := &phase3PermohonanRepository{}
			userRepo := &phase3UserRepository{user: entities.User{ID: uuid.New(), Role: test.role, Unit: test.callerUnit}, configuredULP: test.configured}
			s := &permohonanService{permohonanRepository: permohonanRepo, slaRuleRepository: phase3SLARepository{}, userRepository: userRepo, db: phase3DB(t)}
			req := baseCreateRequest(test.connection)
			req.UlpUnit = test.targetULP

			response, err := s.Create(context.Background(), req, userRepo.user.ID.String())
			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.wantULP, response.UlpUnit)
			require.Len(t, response.WorkflowNodes, len(workflow.Definitions()))
			require.Equal(t, int16(2), response.CurrentStage)
			require.Equal(t, "available", response.WorkflowNodes[1].Status)
		})
	}
}

func TestDetailReturnsCallerOwnedAvailableAction(t *testing.T) {
	p := entities.Permohonan{ID: uuid.New(), JenisSambungan: rbac.JenisSambunganJTR, UlpUnit: "ULP Taman", RequestDate: time.Now(), CreatedBy: uuid.New()}
	nodes, _, err := entities.InitializeWorkflow(p, mustSLARules(t, p.JenisSambungan), time.Now())
	require.NoError(t, err)
	p.WorkflowNodes = nodes
	repo := &phase3PermohonanRepository{byID: p}
	user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Taman"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, db: phase3DB(t)}

	response, err := s.GetById(context.Background(), p.ID.String(), user.ID.String())
	require.NoError(t, err)
	require.Len(t, response.AvailableActions, 1)
	require.Equal(t, string(workflow.Survei), response.AvailableActions[0].WorkflowNode)
	require.Equal(t, "/api/permohonan/{id}/survei", response.AvailableActions[0].Path)
}

func TestListProjectsOnlyCallerOwnedParallelAction(t *testing.T) {
	p := entities.Permohonan{ID: uuid.New(), JenisSambungan: rbac.JenisSambunganJTR, UlpUnit: "ULP Taman", RequestDate: time.Now(), CreatedBy: uuid.New()}
	nodes, _, err := entities.InitializeWorkflow(p, mustSLARules(t, p.JenisSambungan), time.Now())
	require.NoError(t, err)
	completed := map[string]bool{
		string(workflow.Permohonan): true, string(workflow.Survei): true, string(workflow.RAB): true,
		string(workflow.KebutuhanTiang): true, string(workflow.Perluasan): true, string(workflow.NPS): true,
	}
	for i := range nodes {
		if completed[nodes[i].WorkflowNode] {
			nodes[i].Status = string(workflow.Completed)
		}
	}
	poleRequired := true
	delegated := string(workflow.Delegated)

	tests := []struct {
		role string
		node workflow.Code
	}{
		{role: rbac.RolePerencanaan, node: workflow.WOTiang},
		{role: rbac.RoleKonstruksi, node: workflow.WOKonstruksi},
		{role: rbac.RoleTransaksiEnergi, node: workflow.WOAPP},
	}
	for _, test := range tests {
		t.Run(test.role, func(t *testing.T) {
			repo := &phase3PermohonanRepository{list: []query.Permohonan{{
				ID: p.ID.String(), JenisSambungan: p.JenisSambungan, UlpUnit: p.UlpUnit,
				KebutuhanTiang: &poleRequired, NpsDelegationStatus: &delegated, WorkflowNodes: nodes,
			}}}
			user := entities.User{ID: uuid.New(), Role: test.role, Unit: "UP3"}
			s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, db: phase3DB(t)}
			filter := &query.PermohonanFilter{Scope: "mine"}

			results, total, err := s.List(context.Background(), filter, user.ID.String())
			require.NoError(t, err)
			require.EqualValues(t, 1, total)
			require.Len(t, results[0].AvailableActions, 1)
			require.Equal(t, string(test.node), results[0].AvailableActions[0].WorkflowNode)
			require.Equal(t, test.role, repo.filter.CurrentRole)
		})
	}
}

func TestAggregateSLAUsesMostUrgentActionableNode(t *testing.T) {
	now := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	overdueDate := now.AddDate(0, 0, -1)
	dueSoonDate := now.AddDate(0, 0, 1)
	onTimeDate := now.AddDate(0, 0, 10)
	nodes := []entities.PermohonanActivity{
		{WorkflowNode: string(workflow.Survei), SlaDeadline: &onTimeDate},
		{WorkflowNode: string(workflow.RAB), SlaDeadline: &dueSoonDate},
		{WorkflowNode: string(workflow.WOAPP), SlaDeadline: &overdueDate},
		{WorkflowNode: string(workflow.Permohonan), SlaDeadline: &now},
	}
	evaluated := workflow.Result{Nodes: map[workflow.Code]workflow.Status{
		workflow.Survei:     workflow.Available,
		workflow.RAB:        workflow.InProgress,
		workflow.WOAPP:      workflow.Available,
		workflow.Permohonan: workflow.Completed,
	}}

	deadline, status := aggregateSLA(nodes, evaluated, now)
	require.NotNil(t, deadline)
	require.Equal(t, overdueDate.Format("2006-01-02"), *deadline)
	require.Equal(t, "overdue", status)
}

func TestAggregateSLAReturnsNoneWithoutActionableDeadline(t *testing.T) {
	deadline, status := aggregateSLA(nil, workflow.Result{Nodes: map[workflow.Code]workflow.Status{}}, time.Now())
	require.Nil(t, deadline)
	require.Equal(t, "none", status)
}

func TestSubmitSurveySupportsBothConnectionFamilies(t *testing.T) {
	tests := []struct {
		name       string
		connection string
		role       string
		unit       string
	}{
		{name: "JTR survey belongs to matching ULP teknik", connection: rbac.JenisSambunganJTR, role: rbac.RoleTeknik, unit: "ULP Taman"},
		{name: "PLG TM survey belongs to perencanaan", connection: rbac.JenisSambunganPlgTmKurang5, role: rbac.RolePerencanaan, unit: "UP3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := phase4Request(t, test.connection)
			repo := &phase3PermohonanRepository{byID: p}
			docRepo := &phase4DocumentRepository{}
			user := entities.User{ID: uuid.New(), Role: test.role, Unit: test.unit}
			s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

			response, err := s.SubmitSurvey(context.Background(), p.ID.String(), user.ID.String(), dto.SurveySubmitRequest{
				SurveyedAt: "2026-09-08", Notes: "survey evidence reviewed", DocumentIDs: []string{uuid.NewString()},
			})
			require.NoError(t, err)
			require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.Survei).Status)
			require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.RAB).Status)
			require.Equal(t, []workflow.Code{workflow.Survei}, docRepo.attachCalls)
			require.Equal(t, 1, repo.saveCalls)
			require.Len(t, repo.logs, 1)
		})
	}
}

func TestSubmitSurveyRejectsWrongOwnerAndDuplicate(t *testing.T) {
	p := phase4Request(t, rbac.JenisSambunganJTR)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{}
	wrongUser := entities.User{ID: uuid.New(), Role: rbac.RolePerencanaan, Unit: "UP3"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: wrongUser}, documentRepository: docRepo, db: phase3DB(t)}
	req := dto.SurveySubmitRequest{SurveyedAt: "2026-09-08", DocumentIDs: []string{uuid.NewString()}}

	_, err := s.SubmitSurvey(context.Background(), p.ID.String(), wrongUser.ID.String(), req)
	require.ErrorIs(t, err, rbac.ErrWorkflowForbidden)
	require.Empty(t, docRepo.attachCalls)
	mismatchedULPUser := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Lain"}
	s.userRepository = &phase3UserRepository{user: mismatchedULPUser}
	_, err = s.SubmitSurvey(context.Background(), p.ID.String(), mismatchedULPUser.ID.String(), req)
	require.ErrorIs(t, err, rbac.ErrWorkflowForbidden)
	require.Empty(t, docRepo.attachCalls)

	owner := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Taman"}
	s.userRepository = &phase3UserRepository{user: owner}
	_, err = s.SubmitSurvey(context.Background(), p.ID.String(), owner.ID.String(), req)
	require.NoError(t, err)
	_, err = s.SubmitSurvey(context.Background(), p.ID.String(), owner.ID.String(), req)
	require.ErrorIs(t, err, workflow.ErrNotActionable)
	require.Len(t, docRepo.attachCalls, 1)
}

func TestSubmitRABRejectsOutOfOrderAndCompletesPoleDecision(t *testing.T) {
	p := phase4Request(t, rbac.JenisSambunganJTR)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{}
	user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Taman"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}
	poleRequired := true
	req := dto.RABSubmitRequest{KebutuhanTiang: &poleRequired, DocumentIDs: []string{uuid.NewString()}}

	_, err := s.SubmitRAB(context.Background(), p.ID.String(), user.ID.String(), req)
	require.ErrorIs(t, err, workflow.ErrNotActionable)
	require.Zero(t, repo.saveCalls)

	advancePhase4(t, &p, nil, workflow.Survei)
	repo.byID = p
	response, err := s.SubmitRAB(context.Background(), p.ID.String(), user.ID.String(), req)
	require.NoError(t, err)
	require.NotNil(t, response.KebutuhanTiang)
	require.True(t, *response.KebutuhanTiang)
	require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.RAB).Status)
	require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.KebutuhanTiang).Status)
	require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.Perluasan).Status)
	require.Equal(t, []workflow.Code{workflow.RAB, workflow.KebutuhanTiang}, docRepo.attachCalls)
	require.Len(t, repo.logs, 2)
}

func TestSubmitExpansionDelegatesOrReturns(t *testing.T) {
	for _, outcome := range []string{string(workflow.Delegated), string(workflow.Return)} {
		t.Run(outcome, func(t *testing.T) {
			p := phase4Request(t, rbac.JenisSambunganPlgTmLebih5)
			poleRequired := true
			advancePhase4(t, &p, &workflow.Decisions{KebutuhanTiang: &poleRequired}, workflow.Survei, workflow.RAB, workflow.KebutuhanTiang)
			repo := &phase3PermohonanRepository{byID: p}
			docRepo := &phase4DocumentRepository{}
			user := entities.User{ID: uuid.New(), Role: rbac.RoleNps, Unit: "UP3"}
			s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

			response, err := s.SubmitExpansion(context.Background(), p.ID.String(), user.ID.String(), dto.ExpansionSubmitRequest{
				NpsDelegationStatus: outcome, DocumentIDs: []string{uuid.NewString()},
			})
			require.NoError(t, err)
			require.Equal(t, outcome, *response.NpsDelegationStatus)
			require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.Perluasan).Status)
			require.Equal(t, string(workflow.Completed), responseNode(t, response, workflow.NPS).Status)
			require.Equal(t, "nps_"+outcome, repo.logs[1].Action)
			if outcome == string(workflow.Return) {
				require.Equal(t, string(workflow.Returned), response.Status)
				require.Empty(t, response.AvailableActions)
			} else {
				require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.WOTiang).Status)
				require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.WOKonstruksi).Status)
				require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.WOAPP).Status)
			}
		})
	}
}

func TestDelegationSkipsPoleBranchWhenRABSaysNoPole(t *testing.T) {
	p := phase4Request(t, rbac.JenisSambunganJTMGardu)
	poleRequired := false
	advancePhase4(t, &p, &workflow.Decisions{KebutuhanTiang: &poleRequired}, workflow.Survei, workflow.RAB, workflow.KebutuhanTiang)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{}
	user := entities.User{ID: uuid.New(), Role: rbac.RoleNps, Unit: "UP3"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

	response, err := s.SubmitExpansion(context.Background(), p.ID.String(), user.ID.String(), dto.ExpansionSubmitRequest{
		NpsDelegationStatus: string(workflow.Delegated), DocumentIDs: []string{uuid.NewString()},
	})
	require.NoError(t, err)
	require.Equal(t, string(workflow.Skipped), responseNode(t, response, workflow.WOTiang).Status)
	require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.WOKonstruksi).Status)
	require.Equal(t, string(workflow.Available), responseNode(t, response, workflow.WOAPP).Status)
}

func TestSubmitRABForPLGTMUsesPlanningOwner(t *testing.T) {
	p := phase4Request(t, rbac.JenisSambunganPlgTmKurang5)
	advancePhase4(t, &p, nil, workflow.Survei)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{}
	user := entities.User{ID: uuid.New(), Role: rbac.RolePerencanaan, Unit: "UP3"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}
	poleRequired := false

	response, err := s.SubmitRAB(context.Background(), p.ID.String(), user.ID.String(), dto.RABSubmitRequest{
		KebutuhanTiang: &poleRequired, DocumentIDs: []string{uuid.NewString()},
	})
	require.NoError(t, err)
	require.NotNil(t, response.KebutuhanTiang)
	require.False(t, *response.KebutuhanTiang)
}

func TestEvidenceFailurePreventsWorkflowPersistence(t *testing.T) {
	p := phase4Request(t, rbac.JenisSambunganJTR)
	advancePhase4(t, &p, nil, workflow.Survei)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{failAt: 2}
	user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Taman"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}
	poleRequired := false

	_, err := s.SubmitRAB(context.Background(), p.ID.String(), user.ID.String(), dto.RABSubmitRequest{
		KebutuhanTiang: &poleRequired, DocumentIDs: []string{uuid.NewString()},
	})
	require.ErrorIs(t, err, documentDTO.ErrDocumentAlreadyAttached)
	require.Zero(t, repo.saveCalls)
	require.Equal(t, string(workflow.Available), persistedNode(t, repo.byID, workflow.RAB).Status)
	require.Equal(t, string(workflow.Locked), persistedNode(t, repo.byID, workflow.KebutuhanTiang).Status)
}

func TestMissingEvidenceIsReportedWithoutWorkflowPersistence(t *testing.T) {
	p := phase4Request(t, rbac.JenisSambunganJTR)
	repo := &phase3PermohonanRepository{byID: p}
	docRepo := &phase4DocumentRepository{failAt: 1, attachErr: documentRepository.ErrDocumentNotFound}
	user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Taman"}
	s := &permohonanService{permohonanRepository: repo, userRepository: &phase3UserRepository{user: user}, documentRepository: docRepo, db: phase3DB(t)}

	_, err := s.SubmitSurvey(context.Background(), p.ID.String(), user.ID.String(), dto.SurveySubmitRequest{
		SurveyedAt: "2026-09-08", DocumentIDs: []string{uuid.NewString()},
	})
	require.ErrorIs(t, err, documentDTO.ErrDocumentNotFound)
	require.Zero(t, repo.saveCalls)
}

func TestEvidenceFailureRollsBackDatabaseChanges(t *testing.T) {
	db := phase4IntegrationDB(t)
	user := entities.User{ID: uuid.New(), Name: "Teknik", Email: uuid.NewString() + "@example.test", Role: rbac.RoleTeknik, Unit: "ULP Taman"}
	require.NoError(t, db.Create(&user).Error)
	p := phase4Request(t, rbac.JenisSambunganJTR)
	require.NoError(t, db.Omit("WorkflowNodes").Create(&p).Error)
	for i := range p.WorkflowNodes {
		p.WorkflowNodes[i].PermohonanID = p.ID
	}
	require.NoError(t, db.Create(&p.WorkflowNodes).Error)
	document := entities.Document{ID: uuid.New(), Type: "survey", FilePath: "private/test.pdf", UploadedBy: user.ID}
	require.NoError(t, db.Create(&document).Error)

	realPermohonanRepository := repository.NewPermohonanRepository(db)
	s := &permohonanService{
		permohonanRepository: failingSavePermohonanRepository{PermohonanRepository: realPermohonanRepository},
		userRepository:       userRepository.NewUserRepository(db),
		documentRepository:   documentRepository.NewDocumentRepository(db),
		db:                   db,
	}
	_, err := s.SubmitSurvey(context.Background(), p.ID.String(), user.ID.String(), dto.SurveySubmitRequest{
		SurveyedAt: "2026-09-08", DocumentIDs: []string{document.ID.String()},
	})
	require.ErrorIs(t, err, dto.ErrSubmitActivity)

	var reloaded entities.Document
	require.NoError(t, db.First(&reloaded, "id = ?", document.ID).Error)
	require.Nil(t, reloaded.PermohonanID)
	var evidenceCount int64
	require.NoError(t, db.Model(&entities.DocumentEvidence{}).Count(&evidenceCount).Error)
	require.Zero(t, evidenceCount)
	var node entities.PermohonanActivity
	require.NoError(t, db.Where("permohonan_id = ? AND workflow_node = ?", p.ID, workflow.Survei).First(&node).Error)
	require.Equal(t, string(workflow.Available), node.Status)
}

func phase4Request(t *testing.T, connection string) entities.Permohonan {
	t.Helper()
	p := entities.Permohonan{
		ID: uuid.New(), NoPermohonan: "PBPD-2026-" + uuid.NewString()[:8], JenisPermohonan: rbac.JenisPermohonanPasangBaru,
		JenisSambungan: connection, UlpUnit: "ULP Taman", PelangganNama: "Pelanggan Uji",
		PelangganAlamat: "Alamat sintetis", PelangganNoHp: "0800000000", RequestDate: time.Now(),
		Status: string(workflow.Active), CreatedBy: uuid.New(),
	}
	nodes, result, err := entities.InitializeWorkflow(p, mustSLARules(t, connection), time.Now())
	require.NoError(t, err)
	p.WorkflowNodes = nodes
	p.CurrentStage = result.CurrentStage
	return p
}

func advancePhase4(t *testing.T, p *entities.Permohonan, decisions *workflow.Decisions, codes ...workflow.Code) {
	t.Helper()
	snapshot := p.WorkflowSnapshot()
	if decisions != nil {
		if decisions.KebutuhanTiang != nil {
			value := *decisions.KebutuhanTiang
			snapshot.Decisions.KebutuhanTiang = &value
		}
		if decisions.NPS != "" {
			snapshot.Decisions.NPS = decisions.NPS
		}
		if decisions.PerluPDKB != nil {
			value := *decisions.PerluPDKB
			snapshot.Decisions.PerluPDKB = &value
		}
	}
	var result workflow.Result
	var err error
	for _, code := range codes {
		snapshot, result, err = workflow.Transition(snapshot, code, workflow.Completed)
		require.NoError(t, err)
	}
	for i := range p.WorkflowNodes {
		p.WorkflowNodes[i].Status = string(result.Nodes[workflow.Code(p.WorkflowNodes[i].WorkflowNode)])
	}
	p.CurrentStage = result.CurrentStage
	p.Status = string(result.Status)
	p.KebutuhanTiang = cloneBool(snapshot.Decisions.KebutuhanTiang)
	p.PerluPdkb = cloneBool(snapshot.Decisions.PerluPDKB)
	if snapshot.Decisions.NPS != "" {
		value := string(snapshot.Decisions.NPS)
		p.NpsDelegationStatus = &value
	}
}

func persistedNode(t *testing.T, p entities.Permohonan, code workflow.Code) entities.PermohonanActivity {
	t.Helper()
	for _, node := range p.WorkflowNodes {
		if node.WorkflowNode == string(code) {
			return node
		}
	}
	t.Fatalf("node %s not found", code)
	return entities.PermohonanActivity{}
}

func responseNode(t *testing.T, p dto.PermohonanResponse, code workflow.Code) dto.WorkflowNodeResponse {
	t.Helper()
	for _, node := range p.WorkflowNodes {
		if node.WorkflowNode == string(code) {
			return node
		}
	}
	t.Fatalf("node %s not found", code)
	return dto.WorkflowNodeResponse{}
}

func TestGetLogsReturnsActorNames(t *testing.T) {
	db := phase4IntegrationDB(t)
	s := &permohonanService{db: db, permohonanRepository: repository.NewPermohonanRepository(db)}
	p := entities.Permohonan{ID: uuid.New()}
	require.NoError(t, db.Create(&p).Error)
	users := []entities.User{
		{ID: uuid.New(), Name: "Budi Santoso", Email: "budi@example.test"},
		{ID: uuid.New(), Name: "Siti Rahma", Email: "siti@example.test"},
	}
	require.NoError(t, db.Create(&users).Error)
	now := time.Now().UTC().Truncate(time.Second)
	logs := make([]entities.ActivityLog, len(users))
	for i, user := range users {
		logs[i] = entities.ActivityLog{
			ID: uuid.New(), PermohonanID: p.ID, Actor: user.ID, Action: "node_completed",
			WorkflowNode: stringPtr("survei"), Detail: stringPtr("Aktivitas selesai"),
		}
		logs[i].CreatedAt = now.Add(time.Duration(i) * time.Minute)
	}
	require.NoError(t, db.Create(&logs).Error)
	otherLog := entities.ActivityLog{ID: uuid.New(), PermohonanID: uuid.New(), Actor: users[0].ID, Action: "created"}
	require.NoError(t, db.Create(&otherLog).Error)

	results, err := s.GetLogs(context.Background(), p.ID.String())
	require.NoError(t, err)
	require.Len(t, results, len(logs))
	for i, result := range results {
		require.Equal(t, dto.ActivityLogResponse{
			ID: logs[i].ID.String(), Actor: users[i].Name, Action: logs[i].Action,
			WorkflowNode: logs[i].WorkflowNode, Detail: logs[i].Detail,
			CreatedAt: logs[i].CreatedAt.Format(time.RFC3339),
		}, result)
	}
}

func phase4IntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	statements := []string{
		`CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, email TEXT, telp_number TEXT, password TEXT, role TEXT, unit TEXT, image_url TEXT, is_verified NUMERIC, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE permohonan (id TEXT PRIMARY KEY, no_permohonan TEXT, jenis_permohonan TEXT, jenis_sambungan TEXT, ulp_unit TEXT, pelanggan_nama TEXT, pelanggan_alamat TEXT, pelanggan_no_hp TEXT, request_date DATETIME, current_stage INTEGER, status TEXT, kebutuhan_tiang NUMERIC, nps_delegation_status TEXT, perlu_pdkb NUMERIC, created_by TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE permohonan_activities (id TEXT PRIMARY KEY, permohonan_id TEXT, workflow_node TEXT, activity_number INTEGER, stage_number INTEGER, status TEXT, sla_deadline DATETIME, payload TEXT, completed_by TEXT, completed_at DATETIME, created_at DATETIME, updated_at DATETIME, UNIQUE(permohonan_id, workflow_node))`,
		`CREATE TABLE documents (id TEXT PRIMARY KEY, type TEXT, file_path TEXT, original_filename TEXT, mime_type TEXT, size_bytes INTEGER, checksum_sha256 TEXT, source TEXT, classification TEXT, scan_status TEXT, scan_checked_at DATETIME, revision INTEGER, supersedes_id TEXT, superseded_by_id TEXT, uploaded_by TEXT, permohonan_id TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE document_evidence (id TEXT PRIMARY KEY, document_id TEXT, permohonan_id TEXT, workflow_node TEXT, attached_by TEXT, created_at DATETIME, updated_at DATETIME, UNIQUE(document_id, workflow_node))`,
		`CREATE TABLE activity_logs (id TEXT PRIMARY KEY, permohonan_id TEXT, activity_number INTEGER, actor TEXT, action TEXT, detail TEXT, workflow_node TEXT, created_at DATETIME, updated_at DATETIME)`,
	}
	for _, statement := range statements {
		require.NoError(t, db.Exec(statement).Error)
	}
	return db
}

func mustSLARules(t *testing.T, connection string) []entities.SLARule {
	t.Helper()
	rules, err := (phase3SLARepository{}).ListByJenis(context.Background(), nil, connection)
	require.NoError(t, err)
	return rules
}

func stringPtr(value string) *string { return &value }
