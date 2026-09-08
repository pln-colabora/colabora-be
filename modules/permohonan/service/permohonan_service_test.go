package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
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

func mustSLARules(t *testing.T, connection string) []entities.SLARule {
	t.Helper()
	rules, err := (phase3SLARepository{}).ListByJenis(context.Background(), nil, connection)
	require.NoError(t, err)
	return rules
}

func stringPtr(value string) *string { return &value }
