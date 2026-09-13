package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/document/service"
	"github.com/pln-colabora/colabora-be/modules/document/validation"
	"github.com/stretchr/testify/require"
)

type uploadErrorService struct {
	err     error
	content service.DocumentContent
}

func (s uploadErrorService) Upload(context.Context, string, dto.DocumentUploadRequest) (dto.DocumentResponse, error) {
	return dto.DocumentResponse{}, s.err
}
func (uploadErrorService) AttachToWorkflowNode(context.Context, string, string, string, []string) error {
	return nil
}
func (s uploadErrorService) Download(context.Context, string, string) (service.DocumentContent, error) {
	return s.content, s.err
}
func (s uploadErrorService) Preview(context.Context, string, string) (service.DocumentContent, error) {
	return s.content, s.err
}
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

func TestPreviewAndDownloadStreamAuthorizedContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name        string
		disposition string
		invoke      func(*documentController, *gin.Context)
	}{
		{
			name:        "preview is inline",
			disposition: "inline",
			invoke: func(controller *documentController, ctx *gin.Context) {
				ctx.Set("user_id", "synthetic-user")
				controller.Preview(ctx)
			},
		},
		{
			name:        "download is attachment",
			disposition: "attachment",
			invoke: func(controller *documentController, ctx *gin.Context) {
				ctx.Set("user_id", "synthetic-user")
				ctx.Params = []gin.Param{{Key: "id", Value: "document-id"}}
				controller.Download(ctx)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte("private document bytes")
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/documents/document-id/preview", nil)
			controller := &documentController{documentService: uploadErrorService{content: service.DocumentContent{
				Body: io.NopCloser(bytes.NewReader(body)), MimeType: "application/pdf", Filename: "evidence.pdf", SizeBytes: int64(len(body)),
			}}}

			tc.invoke(controller, ctx)

			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, "application/pdf", recorder.Header().Get("Content-Type"))
			require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
			require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
			require.Equal(t, fmt.Sprint(len(body)), recorder.Header().Get("Content-Length"))
			disposition, params, err := mime.ParseMediaType(recorder.Header().Get("Content-Disposition"))
			require.NoError(t, err)
			require.Equal(t, tc.disposition, disposition)
			require.Equal(t, "evidence.pdf", params["filename"])
			require.Equal(t, body, recorder.Body.Bytes())
			require.NotContains(t, recorder.Body.String(), "X-Amz-")
		})
	}
}

func TestDocumentStorageFailureIsGeneric(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/documents/document-id/preview", nil)
	ctx.Set("user_id", "synthetic-user")
	controller := &documentController{documentService: uploadErrorService{err: fmt.Errorf("%w: http://garage.internal/private-key", dto.ErrDocumentStorageUnavailable)}}

	controller.Preview(ctx)

	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Contains(t, recorder.Body.String(), dto.ErrDocumentStorageUnavailable.Error())
	require.NotContains(t, recorder.Body.String(), "garage.internal")
	require.NotContains(t, recorder.Body.String(), "private-key")
}
