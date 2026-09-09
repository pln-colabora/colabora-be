package validation

import (
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/stretchr/testify/assert"
)

func fileHeader(size int64, contentType string) *multipart.FileHeader {
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", contentType)
	return &multipart.FileHeader{
		Filename: "evidence.pdf",
		Header:   header,
		Size:     size,
	}
}

func TestValidateDocumentUploadRequest(t *testing.T) {
	v := NewDocumentValidation()

	tests := []struct {
		name    string
		req     dto.DocumentUploadRequest
		wantErr error
	}{
		{
			name: "valid pdf under the size limit",
			req: dto.DocumentUploadRequest{
				File: fileHeader(1<<20, "application/pdf"),
				Type: "evidence",
			},
			wantErr: nil,
		},
		{
			name: "valid jpeg",
			req: dto.DocumentUploadRequest{
				File: fileHeader(1<<20, "image/jpeg"),
				Type: "evidence",
			},
			wantErr: nil,
		},
		{
			name: "empty type",
			req: dto.DocumentUploadRequest{
				File: fileHeader(1<<20, "application/pdf"),
				Type: "   ",
			},
			wantErr: dto.ErrInvalidDocumentType,
		},
		{
			name: "file too large",
			req: dto.DocumentUploadRequest{
				File: fileHeader(11<<20, "application/pdf"),
				Type: "evidence",
			},
			wantErr: dto.ErrFileTooLarge,
		},
		{
			name: "disallowed mime type",
			req: dto.DocumentUploadRequest{
				File: fileHeader(1<<20, "application/zip"),
				Type: "evidence",
			},
			wantErr: dto.ErrInvalidFileType,
		},
		{
			name:    "invalid supersedes id",
			req:     dto.DocumentUploadRequest{File: fileHeader(10, "application/pdf"), Type: "evidence", SupersedesDocumentID: "not-a-uuid"},
			wantErr: dto.ErrInvalidSupersedesID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateDocumentUploadRequest(tt.req)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func TestUploadRejectsMissingFileAndOversizeCategory(t *testing.T) {
	v := NewDocumentValidation()
	assert.ErrorIs(t, v.ValidateDocumentUploadRequest(dto.DocumentUploadRequest{Type: "evidence"}), dto.ErrInvalidFileType)
	assert.ErrorIs(t, v.ValidateDocumentUploadRequest(dto.DocumentUploadRequest{Type: string(make([]byte, 51)), File: fileHeader(10, "application/pdf")}), dto.ErrInvalidDocumentType)
}
