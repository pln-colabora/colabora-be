package controller

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"github.com/pln-colabora/colabora-be/modules/permohonan/validation"
	"github.com/stretchr/testify/require"
)

type phase3ControllerService struct{ createErr error }

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
