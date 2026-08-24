package validation

import (
	"strings"

	"github.com/pln-colabora/colabora-be/modules/document/dto"
)

const maxUploadSizeBytes = 10 << 20 // 10MB — plenty for a phone-camera evidence photo or scanned PDF.

var allowedMimeTypes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
}

type DocumentValidation struct{}

func NewDocumentValidation() *DocumentValidation {
	return &DocumentValidation{}
}

func (v *DocumentValidation) ValidateDocumentUploadRequest(req dto.DocumentUploadRequest) error {
	if strings.TrimSpace(req.Type) == "" {
		return dto.ErrInvalidDocumentType
	}

	if req.File.Size > maxUploadSizeBytes {
		return dto.ErrFileTooLarge
	}

	if !allowedMimeTypes[req.File.Header.Get("Content-Type")] {
		return dto.ErrInvalidFileType
	}

	return nil
}
