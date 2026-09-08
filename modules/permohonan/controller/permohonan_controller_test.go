package controller

import (
	"bytes"
	"context"
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
func (f phase3ControllerService) SubmitPKVendor(context.Context, string, string, dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, f.activityErr
}
func (f phase3ControllerService) SubmitWOPDKB(context.Context, string, string, dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
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
