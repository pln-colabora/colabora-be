package controller

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/document/validation"
	"github.com/stretchr/testify/require"
)

type uploadErrorService struct{ err error }

func (s uploadErrorService) Upload(context.Context, string, dto.DocumentUploadRequest) (dto.DocumentResponse, error) {
	return dto.DocumentResponse{}, s.err
}
func (uploadErrorService) AttachToWorkflowNode(context.Context, string, string, string, []string) error {
	return nil
}
func (uploadErrorService) Download(context.Context, string, string) (string, error) { return "", nil }
func (uploadErrorService) List(context.Context, string, *string) ([]dto.DocumentResponse, error) {
	return nil, nil
}
func (uploadErrorService) HasEvidence(context.Context, string, string) (bool, error) {
	return false, nil
}
func (uploadErrorService) CleanupOrphans(context.Context, time.Time, int) (int, error) { return 0, nil }

func TestUploadMapsScannerOutcomes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name   string
		err    error
		status int
	}{
		{"scanner unavailable", dto.ErrScanFailed, http.StatusServiceUnavailable},
		{"infected", dto.ErrMalwareDetected, http.StatusUnprocessableEntity},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("type", "evidence"))
			part, err := writer.CreatePart(textproto.MIMEHeader{
				"Content-Disposition": []string{`form-data; name="file"; filename="synthetic.pdf"`},
				"Content-Type":        []string{"application/pdf"},
			})
			require.NoError(t, err)
			_, err = part.Write([]byte("%PDF-1.7\nsynthetic"))
			require.NoError(t, err)
			require.NoError(t, writer.Close())

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/documents", &body)
			ctx.Request.Header.Set("Content-Type", writer.FormDataContentType())
			ctx.Set("user_id", "synthetic-user")
			controller := &documentController{documentService: uploadErrorService{err: tc.err}, documentValidation: validation.NewDocumentValidation()}
			controller.Upload(ctx)
			require.Equal(t, tc.status, recorder.Code)
			require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
		})
	}
}
