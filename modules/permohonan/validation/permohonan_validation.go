package validation

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
)

type PermohonanValidation struct {
	validate *validator.Validate
}

func NewPermohonanValidation() *PermohonanValidation {
	return &PermohonanValidation{
		validate: validator.New(),
	}
}

// ValidatePermohonanCreateRequest checks standard struct tags (required/min/max) via the
// go-playground validator, then checks the two enum-like fields separately — a plain
// RegisterValidation'd tag only fires for calls that go through this instance's Struct()
// method, but ctx.ShouldBind (gin's own, separate validator engine) runs first and would
// error on an unrecognized tag name if it were declared in the struct's `binding:"..."` tag.
func (v *PermohonanValidation) ValidatePermohonanCreateRequest(req dto.PermohonanCreateRequest) error {
	if err := v.validate.Struct(req); err != nil {
		return err
	}

	if !contains(rbac.AllJenisPermohonan, req.JenisPermohonan) {
		return fmt.Errorf("jenis_permohonan must be one of: %s", strings.Join(rbac.AllJenisPermohonan, ", "))
	}

	if !contains(rbac.AllJenisSambungan, req.JenisSambungan) {
		return fmt.Errorf("jenis_sambungan must be one of: %s", strings.Join(rbac.AllJenisSambungan, ", "))
	}

	return nil
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
