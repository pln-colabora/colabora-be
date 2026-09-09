package validation

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/modules/document/dto"
)

const MaxUploadSizeBytes = 10 << 20

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
	if strings.TrimSpace(req.Type) == "" || utf8.RuneCountInString(req.Type) > 50 {
		return dto.ErrInvalidDocumentType
	}

	if req.File == nil || req.File.Size <= 0 {
		return dto.ErrInvalidFileType
	}
	if req.File.Size > MaxUploadSizeBytes {
		return dto.ErrFileTooLarge
	}

	if !allowedMimeTypes[req.File.Header.Get("Content-Type")] {
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
