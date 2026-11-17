package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/service"
	"github.com/stretchr/testify/require"
)

type eisImportControllerService struct {
	result dto.EISImportResult
	phase3ControllerService
	format    string
	deleteErr error
	deletedID string
	deletedBy string
}

func (s *eisImportControllerService) ImportEIS(_ context.Context, _, format string, _ io.ReadSeeker) (dto.EISImportResult, error) {
	s.format = format
	return s.result, nil
}

func (*eisImportControllerService) EditEISDraft(context.Context, string, string, dto.EISDraftEditRequest) (dto.PermohonanResponse, error) {
	return dto.PermohonanResponse{}, nil
}

func (s *eisImportControllerService) DeleteEISDraft(_ context.Context, id, userID string) error {
	s.deletedID, s.deletedBy = id, userID
	return s.deleteErr
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

func TestImportEISReportsSkippedRowsInMeta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "eis.xls")
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
	fake := &eisImportControllerService{result: dto.EISImportResult{
		Created: []dto.PermohonanResponse{{NoPermohonan: "PBPD-2026-0043"}},
		Skipped: []dto.EISImportSkippedRow{{Row: 3, NoAgenda: "A-1", ExistingNoPermohonan: "PBPD-2026-0042", Reason: "NOAGENDA sudah pernah di-import"}},
	}}
	(&permohonanController{permohonanService: fake}).ImportEIS(ctx)
	require.Equal(t, http.StatusCreated, recorder.Code)
	require.JSONEq(t, `{"created_count":1,"skipped_count":1,"skipped":[{"row":3,"no_agenda":"A-1","existing_no_permohonan":"PBPD-2026-0042","reason":"NOAGENDA sudah pernah di-import"}]}`,
		responseMeta(t, recorder.Body.Bytes()))
	require.Contains(t, recorder.Body.String(), `"data":[{`)
}

func responseMeta(t *testing.T, body []byte) string {
	t.Helper()
	var envelope struct {
		Meta json.RawMessage `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	return string(envelope.Meta)
}

func TestDeleteEISDraftResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{name: "deleted", status: http.StatusOK},
		{name: "not found", err: service.ErrEISDraftNotFound, status: http.StatusNotFound},
		{name: "continued or related", err: service.ErrEISDraftConflict, status: http.StatusConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodDelete, "/api/permohonan/draft-id/draft", nil)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			ctx.Params = gin.Params{{Key: "id", Value: "draft-id"}}
			ctx.Set("user_id", "actor-id")
			fake := &eisImportControllerService{deleteErr: tc.err}
			(&permohonanController{permohonanService: fake}).DeleteEISDraft(ctx)
			require.Equal(t, tc.status, recorder.Code)
			require.Equal(t, "draft-id", fake.deletedID)
			require.Equal(t, "actor-id", fake.deletedBy)
		})
	}
}
