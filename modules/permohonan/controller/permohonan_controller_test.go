package controller

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	documentDTO "github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"github.com/pln-colabora/colabora-be/modules/permohonan/validation"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
)

type phase3ControllerService struct {
	createErr   error
	activityErr error
}

func (f phase3ControllerService) AssignVendor(context.Context, string, string, dto.VendorAssignmentRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}

func (f phase3ControllerService) SubmitClosing(context.Context, string, string, dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}

func (f phase3ControllerService) Create(context.Context, dto.PermohonanCreateRequest, string) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.createErr
}
func (phase3ControllerService) GetById(context.Context, string, string) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, nil
}
func (phase3ControllerService) List(context.Context, *query.PermohonanFilter, string) ([]query.Permohonan, int64, error) {
	return nil, 0, nil
}
func (phase3ControllerService) GetActivities(context.Context, string) ([]dto.WorkflowNodeResponse, error) {
	return nil, nil
}
func (phase3ControllerService) GetLogs(context.Context, string) ([]dto.ActivityLogResponse, error) {
	return nil, nil
}
func (f phase3ControllerService) SubmitSurvey(context.Context, string, string, dto.SurveySubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitRAB(context.Context, string, string, dto.RABSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitExpansion(context.Context, string, string, dto.ExpansionSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitWOTiang(context.Context, string, string, dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitWOConstruction(context.Context, string, string, dto.WOConstructionSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitWOAPP(context.Context, string, string, dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitReservationTera(context.Context, string, string, dto.ReservationTeraSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitWOPDKB(context.Context, string, string, dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitConstructionExecution(context.Context, string, string, dto.ConstructionExecutionSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitPDKBDocumentation(context.Context, string, string, dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitEnergize(context.Context, string, string, dto.EnergizeSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitSRAPP(context.Context, string, string, dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}

func TestSubmitSurveyMapsActivityErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{name: "wrong owner", err: rbac.ErrWorkflowForbidden, wantCode: http.StatusForbidden},
		{name: "not actionable", err: workflow.ErrNotActionable, wantCode: http.StatusConflict},
		{name: "missing aggregate", err: dto.ErrPermohonanNotFound, wantCode: http.StatusNotFound},
		{name: "evidence conflict", err: documentDTO.ErrDocumentAlreadyAttached, wantCode: http.StatusConflict},
		{name: "persistence error", err: dto.ErrSubmitActivity, wantCode: http.StatusInternalServerError},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := &permohonanController{
				permohonanService:    phase3ControllerService{activityErr: test.err},
				permohonanValidation: validation.NewPermohonanValidation(),
			}
			body := []byte(`{"surveyed_at":"2026-09-08","document_ids":["` + uuid.NewString() + `"]}`)
			request := httptest.NewRequest(http.MethodPost, "/api/permohonan/id/survei", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			ctx.Params = gin.Params{{Key: "id", Value: "id"}}
			ctx.Set("user_id", "actor-id")

			controller.SubmitSurvey(ctx)
			require.Equal(t, test.wantCode, recorder.Code)
		})
	}
}

func TestClosingHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, body string
		err        error
		status     int
	}{
		{"success", `{"document_ids":["%s"]}`, nil, 200},
		{"missing evidence", `{}`, nil, 400},
		{"wrong owner", `{"document_ids":["%s"]}`, rbac.ErrWorkflowForbidden, 403},
		{"unmet join", `{"document_ids":["%s"]}`, workflow.ErrNotActionable, 409},
		{"audit failure", `{"document_ids":["%s"]}`, dto.ErrSubmitActivity, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &permohonanController{permohonanService: phase3ControllerService{activityErr: tc.err}, permohonanValidation: validation.NewPermohonanValidation()}
			body := tc.body
			if body != "{}" {
				body = fmt.Sprintf(body, uuid.NewString())
			}
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/permohonan/id/closing", bytes.NewBufferString(body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Params = gin.Params{{Key: "id", Value: "id"}}
			ctx.Set("user_id", uuid.NewString())
			c.SubmitClosing(ctx)
			require.Equal(t, tc.status, recorder.Code)
		})
	}
}

func TestSubmitSurveyRejectsInvalidDateBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &permohonanController{
		permohonanService:    phase3ControllerService{},
		permohonanValidation: validation.NewPermohonanValidation(),
	}
	body := []byte(`{"surveyed_at":"08-09-2026","document_ids":["` + uuid.NewString() + `"]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/permohonan/id/survei", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	ctx.Set("user_id", "actor-id")

	controller.SubmitSurvey(ctx)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestSubmitWOConstructionAcceptsExplicitFalseDecision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &permohonanController{
		permohonanService:    phase3ControllerService{},
		permohonanValidation: validation.NewPermohonanValidation(),
	}
	body := []byte(`{"perlu_pdkb":false,"document_ids":["` + uuid.NewString() + `"]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/permohonan/id/wo-vendor/konstruksi", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	ctx.Params = gin.Params{{Key: "id", Value: "id"}}
	ctx.Set("user_id", "actor-id")

	controller.SubmitWOConstruction(ctx)
	require.Equal(t, http.StatusOK, recorder.Code)
}

func TestSubmitWOConstructionRejectsMissingDecision(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := &permohonanController{
		permohonanService:    phase3ControllerService{},
		permohonanValidation: validation.NewPermohonanValidation(),
	}
	body := []byte(`{"document_ids":["` + uuid.NewString() + `"]}`)
	request := httptest.NewRequest(http.MethodPost, "/api/permohonan/id/wo-vendor/konstruksi", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	ctx.Set("user_id", "actor-id")

	controller.SubmitWOConstruction(ctx)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestSubmitConstructionExecutionValidatesNodeSelector(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		node     string
		wantCode int
	}{
		{name: "pole installation", node: "pemasangan_tiang", wantCode: http.StatusOK},
		{name: "construction", node: "pelaksanaan_konstruksi", wantCode: http.StatusOK},
		{name: "unrelated node", node: "wo_tiang", wantCode: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := &permohonanController{
				permohonanService:    phase3ControllerService{},
				permohonanValidation: validation.NewPermohonanValidation(),
			}
			body := []byte(`{"workflow_node":"` + test.node + `","document_ids":["` + uuid.NewString() + `"]}`)
			request := httptest.NewRequest(http.MethodPost, "/api/permohonan/id/pelaksanaan-konstruksi", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			ctx.Params = gin.Params{{Key: "id", Value: "id"}}
			ctx.Set("user_id", "actor-id")

			controller.SubmitConstructionExecution(ctx)
			require.Equal(t, test.wantCode, recorder.Code)
		})
	}
}

func TestSubmitSequenceFourValidatesPayloads(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		path     string
		body     string
		submit   func(*permohonanController, *gin.Context)
		wantCode int
	}{
		{
			name: "energize result", path: "/api/permohonan/id/energize-jaringan",
			body:   `{"operation_result":"operasi jaringan selesai","document_ids":["%s"]}`,
			submit: func(c *permohonanController, ctx *gin.Context) { c.SubmitEnergize(ctx) }, wantCode: http.StatusOK,
		},
		{
			name: "blank energize result", path: "/api/permohonan/id/energize-jaringan",
			body:   `{"operation_result":"   ","document_ids":["%s"]}`,
			submit: func(c *permohonanController, ctx *gin.Context) { c.SubmitEnergize(ctx) }, wantCode: http.StatusBadRequest,
		},
		{
			name: "SR APP evidence", path: "/api/permohonan/id/pemasangan-sr-app",
			body:   `{"document_ids":["%s"]}`,
			submit: func(c *permohonanController, ctx *gin.Context) { c.SubmitSRAPP(ctx) }, wantCode: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := &permohonanController{
				permohonanService:    phase3ControllerService{},
				permohonanValidation: validation.NewPermohonanValidation(),
			}
			body := []byte(fmt.Sprintf(test.body, uuid.NewString()))
			request := httptest.NewRequest(http.MethodPost, test.path, bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			ctx.Params = gin.Params{{Key: "id", Value: "id"}}
			ctx.Set("user_id", "actor-id")

			test.submit(controller, ctx)
			require.Equal(t, test.wantCode, recorder.Code)
		})
	}
}

func TestCreateMapsEntryAuthorizationErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{name: "role and connection mismatch", err: dto.ErrCreateForbidden, wantCode: http.StatusForbidden},
		{name: "invalid PLG TM target ULP", err: dto.ErrInvalidULPUnit, wantCode: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := &permohonanController{
				permohonanService:    phase3ControllerService{createErr: test.err},
				permohonanValidation: validation.NewPermohonanValidation(),
			}
			body := []byte(`{"jenis_permohonan":"Pasang Baru (PB)","jenis_sambungan":"JTR","pelanggan_nama":"Pelanggan Uji","pelanggan_alamat":"Alamat sintetis","pelanggan_no_hp":"0800000000"}`)
			request := httptest.NewRequest(http.MethodPost, "/api/permohonan", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			ctx.Set("user_id", "actor-id")

			controller.Create(ctx)
			require.Equal(t, test.wantCode, recorder.Code)
		})
	}
}
