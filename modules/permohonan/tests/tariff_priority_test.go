package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/database/seeders/seeds"
	"github.com/pln-colabora/colabora-be/modules/permohonan/controller"
	"github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	"github.com/pln-colabora/colabora-be/modules/permohonan/service"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/samber/do"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// tariffTestDB mirrors the PostgreSQL columns used by the tariff and
// workflow paths; uuid/jsonb defaults are not portable to SQLite AutoMigrate.
func tariffTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	statements := []string{
		`CREATE TABLE tariff_power_options (id TEXT PRIMARY KEY, tarif TEXT NOT NULL, golongan_tarif TEXT NOT NULL, jenis_sambungan TEXT NOT NULL, daya_min INTEGER NOT NULL, daya_max INTEGER, label TEXT NOT NULL, daya_label TEXT, pembatas TEXT, active BOOLEAN NOT NULL DEFAULT true, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE UNIQUE INDEX idx_tariff_power_option ON tariff_power_options (tarif, golongan_tarif, jenis_sambungan, daya_min, daya_max)`,
		`CREATE TABLE permohonan (id TEXT PRIMARY KEY, no_permohonan TEXT, jenis_permohonan TEXT, jenis_sambungan TEXT, tarif TEXT, daya_lama INTEGER, daya_baru INTEGER, golongan_tarif TEXT, golongan_tarif_lama TEXT, pembatas_baru TEXT, pembatas_lama TEXT, priority TEXT, ulp_unit TEXT, pelanggan_nama TEXT, pelanggan_alamat TEXT, pelanggan_no_hp TEXT, request_date DATETIME, tanggal_pembayaran DATE, nominal_pembayaran INTEGER, sla_basis TEXT DEFAULT calendar, current_stage INTEGER, status TEXT, kebutuhan_tiang NUMERIC, nps_delegation_status TEXT, perlu_pdkb NUMERIC, legacy_priority_exempt BOOLEAN DEFAULT false, energize_status TEXT, created_by TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE permohonan_number_counters (request_year INTEGER PRIMARY KEY, last_number INTEGER NOT NULL)`,
		`CREATE TABLE permohonan_activities (id TEXT PRIMARY KEY, permohonan_id TEXT, workflow_node TEXT, activity_number INTEGER, stage_number INTEGER, status TEXT, sla_deadline DATETIME, sla_basis TEXT DEFAULT calendar, payload TEXT, revision INTEGER DEFAULT 1, completed_by TEXT, completed_at DATETIME, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME, UNIQUE(permohonan_id, workflow_node))`,
		`CREATE TABLE activity_logs (id TEXT PRIMARY KEY, permohonan_id TEXT, activity_number INTEGER, actor TEXT, action TEXT, detail TEXT, workflow_node TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
	}
	for _, statement := range statements {
		require.NoError(t, db.Exec(statement).Error)
	}
	return db
}

type tariffTestUsers map[string]entities.User

func (u tariffTestUsers) Register(context.Context, *gorm.DB, entities.User) (entities.User, error) {
	return entities.User{}, nil
}
func (u tariffTestUsers) GetUserById(_ context.Context, _ *gorm.DB, id string) (entities.User, error) {
	if user, ok := u[id]; ok {
		return user, nil
	}
	return entities.User{}, gorm.ErrRecordNotFound
}
func (u tariffTestUsers) GetUserByEmail(context.Context, *gorm.DB, string) (entities.User, error) {
	return entities.User{}, nil
}
func (u tariffTestUsers) CheckEmail(context.Context, *gorm.DB, string) (entities.User, bool, error) {
	return entities.User{}, false, nil
}
func (u tariffTestUsers) Update(context.Context, *gorm.DB, entities.User) (entities.User, error) {
	return entities.User{}, nil
}
func (u tariffTestUsers) Delete(context.Context, *gorm.DB, string) error { return nil }
func (u tariffTestUsers) ExistsByUnitAndRoles(context.Context, *gorm.DB, string, []string) (bool, error) {
	return true, nil
}

type tariffTestSLA struct{}

func (tariffTestSLA) GetByActivityAndJenis(_ context.Context, _ *gorm.DB, activity int16, connection string) (entities.SLARule, error) {
	return entities.SLARule{ActivityNumber: activity, JenisSambungan: connection, OffsetDays: 2}, nil
}
func (tariffTestSLA) ListByJenis(_ context.Context, _ *gorm.DB, connection string) ([]entities.SLARule, error) {
	rules := make([]entities.SLARule, 0)
	for _, definition := range workflow.Definitions() {
		if definition.SLAActivityNumber != nil {
			rules = append(rules, entities.SLARule{ActivityNumber: *definition.SLAActivityNumber, JenisSambungan: connection, OffsetDays: 2})
		}
	}
	return rules, nil
}

type tariffTestDocuments struct{}

func (tariffTestDocuments) Create(context.Context, *gorm.DB, entities.Document) (entities.Document, error) {
	return entities.Document{}, nil
}
func (tariffTestDocuments) GetById(context.Context, *gorm.DB, string) (entities.Document, error) {
	return entities.Document{}, gorm.ErrRecordNotFound
}
func (tariffTestDocuments) ListByPermohonan(context.Context, *gorm.DB, string, *string) ([]entities.Document, error) {
	return nil, nil
}
func (tariffTestDocuments) ExistsForWorkflowNode(context.Context, *gorm.DB, string, string) (bool, error) {
	return false, nil
}
func (tariffTestDocuments) AttachToWorkflowNode(_ context.Context, _ *gorm.DB, ids []string, _, _, _ string) (int64, error) {
	return int64(len(ids)), nil
}

type tariffHarness struct {
	db       *gorm.DB
	engine   *gin.Engine
	frontage entities.User
	planner  entities.User
}

func newTariffHarness(t *testing.T) tariffHarness {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := tariffTestDB(t)
	require.NoError(t, seeds.SyncTariffPowerCatalogue(db))

	frontage := entities.User{ID: uuid.New(), Role: rbac.RolePelayananPelanggan, Unit: "ULP Taman"}
	planner := entities.User{ID: uuid.New(), Role: rbac.RolePerencanaan}
	users := tariffTestUsers{frontage.ID.String(): frontage, planner.ID.String(): planner}
	svc := service.NewPermohonanService(
		repository.NewPermohonanRepository(db), tariffTestSLA{}, repository.NewTariffPowerRepository(db),
		users, tariffTestDocuments{}, nil, repository.NewVendorAssignmentRepository(), db,
	)
	injector := do.New()
	do.ProvideNamedValue(injector, constants.DB, db)
	permohonan := controller.NewPermohonanController(injector, svc)
	changes := permohonan.(controller.WorkflowChangesController)

	engine := gin.New()
	engine.Use(func(ctx *gin.Context) { ctx.Set("user_id", ctx.GetHeader("X-Test-User")) })
	engine.POST("/api/permohonan", permohonan.Create)
	engine.GET("/api/permohonan", permohonan.GetAll)
	engine.GET("/api/permohonan/:id", permohonan.GetById)
	engine.GET("/api/permohonan/:id/activities/:workflow_node", permohonan.GetActivity)
	engine.POST("/api/permohonan/:id/verifikasi-prioritas", changes.SubmitPriority)
	engine.GET("/api/tariffs", permohonan.GetTariffOptions)
	return tariffHarness{db: db, engine: engine, frontage: frontage, planner: planner}
}

type envelope struct {
	Status  any             `json:"status"`
	Message string          `json:"message"`
	Error   any             `json:"error"`
	Data    json.RawMessage `json:"data"`
}

func (h tariffHarness) do(t *testing.T, method, path string, user entities.User, body any) (int, envelope) {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Test-User", user.ID.String())
	recorder := httptest.NewRecorder()
	h.engine.ServeHTTP(recorder, request)
	var result envelope
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &result), recorder.Body.String())
	return recorder.Code, result
}

type tariffItem struct {
	Tarif          string  `json:"tarif"`
	GolonganTarif  string  `json:"golongan_tarif"`
	JenisSambungan string  `json:"jenis_sambungan"`
	Label          string  `json:"label"`
	DayaLabel      string  `json:"daya_label"`
	DayaMin        int64   `json:"daya_min"`
	DayaMax        *int64  `json:"daya_max"`
	Pembatas       *string `json:"pembatas"`
}

func (h tariffHarness) tariffs(t *testing.T, connection string) []tariffItem {
	t.Helper()
	code, result := h.do(t, http.MethodGet, "/api/tariffs?jenis_sambungan="+strings.ReplaceAll(connection, " ", "%20"), h.frontage, nil)
	require.Equal(t, http.StatusOK, code)
	var items []tariffItem
	require.NoError(t, json.Unmarshal(result.Data, &items))
	return items
}

func byGolongan(items []tariffItem) map[string][]tariffItem {
	grouped := map[string][]tariffItem{}
	for _, item := range items {
		grouped[item.GolonganTarif] = append(grouped[item.GolonganTarif], item)
	}
	return grouped
}

func powers(items []tariffItem) []int64 {
	result := make([]int64, 0, len(items))
	for _, item := range items {
		result = append(result, item.DayaMin)
	}
	return result
}

func TestTariffsReturnExactPowerSteps(t *testing.T) {
	h := newTariffHarness(t)
	items := h.tariffs(t, rbac.JenisSambunganJTR)
	grouped := byGolongan(items)

	require.Len(t, items, 211)
	for _, golongan := range []string{"R1", "R2", "R3", "B1", "B2", "B3", "S1", "S2", "S3", "P1", "P2", "I1", "I2", "I3", "I4"} {
		require.NotEmpty(t, grouped[golongan], golongan)
	}
	for _, item := range items {
		require.NotContains(t, item.Label, "s.d.")
		require.NotContains(t, item.DayaLabel, "s.d.")
		require.NotContains(t, item.GolonganTarif, "-")
		require.Equal(t, rbac.JenisSambunganJTR, item.JenisSambungan)
		require.Equal(t, item.DayaLabel, item.Label)
		if item.GolonganTarif == "I4" {
			continue
		}
		require.NotNil(t, item.DayaMax)
		require.Equal(t, item.DayaMin, *item.DayaMax)
		require.NotNil(t, item.Pembatas)
	}

	require.Equal(t, []int64{450, 900, 1300, 2200}, powers(grouped["R1"]))
	require.Equal(t, rbac.TarifRumahTangga, grouped["R1"][0].Tarif)
	require.Equal(t, "450 VA", grouped["R1"][0].DayaLabel)
	require.Equal(t, "1x2", *grouped["R1"][0].Pembatas)
	require.Equal(t, []int64{3500, 3900, 4400, 5500}, powers(grouped["R2"]))
	require.Equal(t, []int64{220}, powers(grouped["S1"]))
	require.Equal(t, "1X1", *grouped["S1"][0].Pembatas)
	require.Equal(t, rbac.TarifSosial, grouped["S1"][0].Tarif)
	require.Len(t, grouped["S2"], 25)
	require.Equal(t, int64(450), grouped["S2"][0].DayaMin)
	require.Equal(t, int64(197000), grouped["S2"][24].DayaMin)
	require.Equal(t, powers(grouped["S2"]), powers(grouped["P1"]))
	require.Equal(t, powers(grouped["R3"]), powers(grouped["B2"]))
	require.Equal(t, []int64{450, 900, 1300, 2200, 3500, 3900, 4400, 5500}, powers(grouped["B1"]))

	i1 := grouped["I1"]
	require.Equal(t, int64(450), i1[0].DayaMin)
	require.Equal(t, int64(13200), i1[len(i1)-1].DayaMin)
	require.Len(t, i1, 13)
	require.Equal(t, int64(16500), grouped["I2"][0].DayaMin)
	require.Equal(t, int64(197000), grouped["I2"][len(grouped["I2"])-1].DayaMin)

	for _, golongan := range []string{"B3", "S3", "P2", "I3"} {
		steps := grouped[golongan]
		require.Len(t, steps, 21, golongan)
		require.Equal(t, int64(210000), steps[0].DayaMin)
		require.Equal(t, "6", *steps[0].Pembatas)
		require.Equal(t, "210.000 VA", steps[0].DayaLabel)
		require.Equal(t, int64(13800000), steps[20].DayaMin)
		require.Equal(t, "13.800.000 VA", steps[20].DayaLabel)
		require.Equal(t, "399", *steps[20].Pembatas)
	}

	require.Len(t, grouped["I4"], 1)
	i4 := grouped["I4"][0]
	require.Equal(t, rbac.TarifIndustri, i4.Tarif)
	require.Equal(t, int64(30000000), i4.DayaMin)
	require.Nil(t, i4.DayaMax)
	require.Nil(t, i4.Pembatas)
	require.Equal(t, "> 30.000.000 VA", i4.DayaLabel)

	// Every connection type keeps the full catalogue, as before.
	for _, connection := range rbac.AllJenisSambungan {
		require.Len(t, h.tariffs(t, connection), 211, connection)
	}
}

func TestTariffSyncIsIdempotentAndRetiresLegacyBands(t *testing.T) {
	db := tariffTestDB(t)
	legacyMax := int64(5500)
	legacy := entities.TariffPowerOption{
		Tarif: rbac.TarifRumahTangga, GolonganTarif: "R-2", JenisSambungan: rbac.JenisSambunganJTR,
		DayaMin: 3500, DayaMax: &legacyMax, Label: "3.500 VA s.d. 5.500 VA", Active: true,
	}
	require.NoError(t, db.Create(&legacy).Error)

	require.NoError(t, seeds.SyncTariffPowerCatalogue(db))
	var first int64
	require.NoError(t, db.Model(&entities.TariffPowerOption{}).Count(&first).Error)
	require.NoError(t, seeds.SyncTariffPowerCatalogue(db))
	var second, active int64
	require.NoError(t, db.Model(&entities.TariffPowerOption{}).Count(&second).Error)
	require.NoError(t, db.Model(&entities.TariffPowerOption{}).Where("active = ?", true).Count(&active).Error)
	require.Equal(t, first, second)
	require.Equal(t, int64(211*len(rbac.AllJenisSambungan)+1), second)
	require.Equal(t, int64(211*len(rbac.AllJenisSambungan)), active)

	var retired entities.TariffPowerOption
	require.NoError(t, db.First(&retired, "id = ?", legacy.ID).Error)
	require.False(t, retired.Active)
	options, err := repository.NewTariffPowerRepository(db).List(context.Background(), db, rbac.JenisSambunganJTR)
	require.NoError(t, err)
	for _, option := range options {
		require.NotContains(t, option.Label, "s.d.")
	}
}

func createBody(jenis string, extra map[string]any) map[string]any {
	body := map[string]any{
		"jenis_permohonan": jenis, "jenis_sambungan": rbac.JenisSambunganJTR,
		"pelanggan_nama": "Pelanggan Uji", "pelanggan_alamat": "Alamat sintetis", "pelanggan_no_hp": "0800000000",
		"tanggal_pembayaran": "2026-09-25", "nominal_pembayaran": 150000,
	}
	for key, value := range extra {
		body[key] = value
	}
	return body
}

func decodeObject(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var object map[string]any
	require.NoError(t, json.Unmarshal(raw, &object))
	return object
}

func TestCreatePermohonanValidatesGolonganAndPembatas(t *testing.T) {
	h := newTariffHarness(t)

	code, result := h.do(t, http.MethodPost, "/api/permohonan", h.frontage, createBody(rbac.JenisPermohonanPasangBaru, map[string]any{
		"tarif": rbac.TarifRumahTangga, "golongan_tarif": "R1", "daya_baru": 900, "pembatas_baru": "1x4",
	}))
	require.Equal(t, http.StatusCreated, code, result.Error)
	pb := decodeObject(t, result.Data)
	require.Equal(t, "R1", pb["golongan_tarif"])
	require.Equal(t, "1x4", pb["pembatas_baru"])
	require.Contains(t, pb, "golongan_tarif_lama")
	require.Nil(t, pb["golongan_tarif_lama"])
	require.Contains(t, pb, "pembatas_lama")
	require.Contains(t, pb, "priority")
	require.Nil(t, pb["priority"])

	code, result = h.do(t, http.MethodPost, "/api/permohonan", h.frontage, createBody(rbac.JenisPermohonanPerubahanDaya, map[string]any{
		"tarif": rbac.TarifRumahTangga, "golongan_tarif": "R2", "daya_baru": 3500, "pembatas_baru": "1x16",
		"golongan_tarif_lama": "R1", "daya_lama": 2200, "pembatas_lama": "1x10",
	}))
	require.Equal(t, http.StatusCreated, code, result.Error)
	pd := decodeObject(t, result.Data)

	code, result = h.do(t, http.MethodGet, fmt.Sprintf("/api/permohonan/%s", pd["id"]), h.frontage, nil)
	require.Equal(t, http.StatusOK, code)
	detail := decodeObject(t, result.Data)
	require.Equal(t, "R2", detail["golongan_tarif"])
	require.Equal(t, "R1", detail["golongan_tarif_lama"])
	require.Equal(t, "1x16", detail["pembatas_baru"])
	require.Equal(t, "1x10", detail["pembatas_lama"])
	require.Equal(t, float64(3500), detail["daya_baru"])
	require.Equal(t, float64(2200), detail["daya_lama"])
	require.Contains(t, detail, "priority")
	require.Nil(t, detail["priority"])

	invalid := []struct {
		name  string
		jenis string
		extra map[string]any
		want  string
	}{
		{"missing golongan", rbac.JenisPermohonanPasangBaru, map[string]any{"tarif": rbac.TarifRumahTangga, "daya_baru": 900}, "golongan_tarif is required"},
		{"R1 has no 220 VA", rbac.JenisPermohonanPasangBaru, map[string]any{"tarif": rbac.TarifRumahTangga, "golongan_tarif": "R1", "daya_baru": 220}, "golongan_tarif"},
		{"power between steps", rbac.JenisPermohonanPasangBaru, map[string]any{"tarif": rbac.TarifIndustri, "golongan_tarif": "I1", "daya_baru": 14000}, "golongan_tarif"},
		{"golongan from another tarif", rbac.JenisPermohonanPasangBaru, map[string]any{"tarif": rbac.TarifBisnis, "golongan_tarif": "R1", "daya_baru": 900}, "golongan_tarif"},
		{"legacy hyphenated code", rbac.JenisPermohonanPasangBaru, map[string]any{"tarif": rbac.TarifRumahTangga, "golongan_tarif": "R-1", "daya_baru": 900}, "golongan_tarif"},
		{"PB with old golongan", rbac.JenisPermohonanPasangBaru, map[string]any{"tarif": rbac.TarifRumahTangga, "golongan_tarif": "R1", "daya_baru": 900, "golongan_tarif_lama": "R1"}, "pasang baru"},
		{"PD old combination invalid", rbac.JenisPermohonanPerubahanDaya, map[string]any{"tarif": rbac.TarifRumahTangga, "golongan_tarif": "R2", "daya_baru": 3500, "golongan_tarif_lama": "R2", "daya_lama": 2200}, "golongan_tarif_lama"},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			code, result := h.do(t, http.MethodPost, "/api/permohonan", h.frontage, createBody(test.jenis, test.extra))
			require.Equal(t, http.StatusBadRequest, code)
			require.NotEqual(t, "failed create permohonan", result.Message)
			require.Contains(t, fmt.Sprint(result.Error), test.want)
		})
	}
}

// advanceToPriority marks the prerequisites of verifikasi_prioritas complete
// directly in storage, the same way the workflow would after survei and RAB.
func advanceToPriority(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	var p entities.Permohonan
	require.NoError(t, db.Preload("WorkflowNodes").First(&p, "id = ?", id).Error)
	snapshot := p.WorkflowSnapshot()
	required := false
	snapshot.Decisions.KebutuhanTiang = &required
	var result workflow.Result
	var err error
	for _, code := range []workflow.Code{workflow.Survei, workflow.RAB, workflow.KebutuhanTiang} {
		snapshot, result, err = workflow.Transition(snapshot, code, workflow.Completed)
		require.NoError(t, err)
	}
	require.Equal(t, workflow.Available, result.Nodes[workflow.Prioritas])
	for code, status := range result.Nodes {
		require.NoError(t, db.Model(&entities.PermohonanActivity{}).Where("permohonan_id = ? AND workflow_node = ?", id, string(code)).Update("status", string(status)).Error)
	}
	require.NoError(t, db.Model(&entities.Permohonan{}).Where("id = ?", id).Update("kebutuhan_tiang", false).Error)
}

func TestVerifikasiPrioritasRequiresPriority(t *testing.T) {
	h := newTariffHarness(t)
	code, result := h.do(t, http.MethodPost, "/api/permohonan", h.frontage, createBody(rbac.JenisPermohonanPasangBaru, map[string]any{
		"tarif": rbac.TarifBisnis, "golongan_tarif": "B3", "daya_baru": 210000, "pembatas_baru": "6",
	}))
	require.Equal(t, http.StatusCreated, code, result.Error)
	id := decodeObject(t, result.Data)["id"].(string)
	advanceToPriority(t, h.db, id)
	path := "/api/permohonan/" + id + "/verifikasi-prioritas"

	for _, body := range []map[string]any{
		{},
		{"priority": ""},
		{"priority": "high"},
		{"priority": "Urgent"},
		{"priority": 1},
		{"priority": "High", "notes": strings.Repeat("a", 2001)},
	} {
		code, result := h.do(t, http.MethodPost, path, h.planner, body)
		require.Equal(t, http.StatusBadRequest, code, body)
		require.NotEmpty(t, result.Error)
	}
	_, result = h.do(t, http.MethodPost, path, h.planner, map[string]any{"priority": "high"})
	require.Contains(t, fmt.Sprint(result.Error), "High, Medium, Low")

	code, result = h.do(t, http.MethodPost, path, h.planner, map[string]any{"priority": "High", "notes": "Pelanggan prioritas", "document_ids": []string{}})
	require.Equal(t, http.StatusOK, code, result.Error)
	require.Equal(t, "High", decodeObject(t, result.Data)["priority"])

	code, result = h.do(t, http.MethodGet, "/api/permohonan/"+id+"/activities/verifikasi_prioritas", h.planner, nil)
	require.Equal(t, http.StatusOK, code, result.Error)
	activity := decodeObject(t, result.Data)
	payload := activity["payload"].(map[string]any)
	require.Equal(t, "High", payload["priority"])
	require.Equal(t, "Pelanggan prioritas", payload["notes"])
	require.Equal(t, string(workflow.Completed), activity["status"])

	code, result = h.do(t, http.MethodGet, "/api/permohonan/"+id, h.planner, nil)
	require.Equal(t, http.StatusOK, code)
	detail := decodeObject(t, result.Data)
	require.Equal(t, "High", detail["priority"])
	require.Equal(t, "B3", detail["golongan_tarif"])
	require.Equal(t, "6", detail["pembatas_baru"])

	code, result = h.do(t, http.MethodGet, "/api/permohonan", h.planner, nil)
	require.Equal(t, http.StatusOK, code, result.Error)
	var rows []map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &rows))
	require.Len(t, rows, 1)
	require.Equal(t, "High", rows[0]["priority"])
	require.Equal(t, "B3", rows[0]["golongan_tarif"])
	require.Equal(t, "6", rows[0]["pembatas_baru"])
	require.Contains(t, rows[0], "golongan_tarif_lama")
	require.Contains(t, rows[0], "pembatas_lama")
}
