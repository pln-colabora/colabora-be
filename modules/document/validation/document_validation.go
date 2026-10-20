package validation

import (
	"mime/multipart"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/modules/document/dto"
)

const (
	MaxUploadSizeBytes        = 30 << 20
	MaxMultipartBodySizeBytes = MaxUploadSizeBytes + (1 << 20)
)

const (
	MimeTypePDF  = "application/pdf"
	MimeTypeJPEG = "image/jpeg"
	MimeTypePNG  = "image/png"
	MimeTypeXLS  = "application/vnd.ms-excel"
	MimeTypeXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
)

var baseAllowedMimeTypes = map[string]bool{
	MimeTypePDF:  true,
	MimeTypeJPEG: true,
	MimeTypePNG:  true,
}

type DocumentValidation struct{}

func NewDocumentValidation() *DocumentValidation {
	return &DocumentValidation{}
}

func (v *DocumentValidation) ValidateDocumentUploadRequest(req dto.DocumentUploadRequest) error {
	return v.validateDocumentUploadRequest(req, true)
}

func (v *DocumentValidation) ValidateAccountUploadRequest(req dto.DocumentUploadRequest) error {
	return v.validateDocumentUploadRequest(req, false)
}

func (v *DocumentValidation) ValidateWorkflowUpload(file *multipart.FileHeader) error {
	return v.validateDocumentUploadRequest(dto.DocumentUploadRequest{File: file, Type: "workflow_evidence"}, true)
}

func IsAllowedMimeType(contentType string, allowExcel bool) bool {
	if baseAllowedMimeTypes[contentType] {
		return true
	}
	return allowExcel && (contentType == MimeTypeXLS || contentType == MimeTypeXLSX)
}

func (v *DocumentValidation) validateDocumentUploadRequest(req dto.DocumentUploadRequest, allowExcel bool) error {
	if strings.TrimSpace(req.Type) == "" || utf8.RuneCountInString(req.Type) > 50 {
		return dto.ErrInvalidDocumentType
	}

	if req.File == nil || req.File.Size <= 0 {
		return dto.ErrInvalidFileType
	}
	if req.File.Size > MaxUploadSizeBytes {
		return dto.ErrFileTooLarge
	}

	if !IsAllowedMimeType(req.File.Header.Get("Content-Type"), allowExcel) {
		return dto.ErrInvalidFileType
	}
	filename := strings.TrimSpace(filepath.Base(req.File.Filename))
	if filename == "" || filename == "." || filename == ".." || utf8.RuneCountInString(filename) > 255 {
		return dto.ErrInvalidFilename
	}
	for _, value := range filename {
		if unicode.IsControl(value) {
			return dto.ErrInvalidFilename
		}
	}
	if req.SupersedesDocumentID != "" {
		if _, err := uuid.Parse(req.SupersedesDocumentID); err != nil {
			return dto.ErrInvalidSupersedesID
		}
	}

	return nil
}
