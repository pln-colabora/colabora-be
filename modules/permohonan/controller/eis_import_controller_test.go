package controller

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/stretchr/testify/require"
)

type eisImportControllerService struct {
	phase3ControllerService
	format string
}

func (s *eisImportControllerService) ImportEIS(_ context.Context, _, format string, _ io.ReadSeeker) ([]dto.PermohonanResponse, error) {
	s.format = format
	return nil, nil
}

func (*eisImportControllerService) EditEISDraft(context.Context, string, string, dto.EISDraftEditRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, nil
}

func (*eisImportControllerService) ContinueEISDraft(context.Context, string, string, []*multipart.FileHeader) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, nil
}

func TestImportEISAcceptsXLSAndXLSX(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		filename string
		format   string
		status   int
	}{
		{"eis.xls", ".xls", http.StatusCreated},
		{"eis.XLSX", ".xlsx", http.StatusCreated},
		{"eis.csv", "", http.StatusBadRequest},
	} {
		t.Run(tc.filename, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			file, err := writer.CreateFormFile("file", tc.filename)
			require.NoError(t, err)
			_, err = file.Write([]byte("sample"))
			require.NoError(t, err)
			require.NoError(t, writer.Close())
			request := httptest.NewRequest(http.MethodPost, "/api/permohonan/import-eis", &body)
			request.Header.Set("Content-Type", writer.FormDataContentType())
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			ctx.Set("user_id", "actor-id")
			fake := &eisImportControllerService{}
			(&permohonanController{permohonanService: fake}).ImportEIS(ctx)
			require.Equal(t, tc.status, recorder.Code)
			require.Equal(t, tc.format, fake.format)
		})
	}
}
