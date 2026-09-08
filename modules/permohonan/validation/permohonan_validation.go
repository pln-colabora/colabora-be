package validation

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
)

type PermohonanValidation struct {
	validate *validator.Validate
}

func (v *PermohonanValidation) ValidateSurveySubmitRequest(req dto.SurveySubmitRequest) error {
	if err := v.validate.Struct(req); err != nil {
		return err
	}
	if _, err := time.Parse("2006-01-02", req.SurveyedAt); err != nil {
		return fmt.Errorf("surveyed_at must use YYYY-MM-DD format")
	}
	return validateDocumentIDs(req.DocumentIDs)
}

func (v *PermohonanValidation) ValidateRABSubmitRequest(req dto.RABSubmitRequest) error {
	if err := v.validate.Struct(req); err != nil {
		return err
	}
	return validateDocumentIDs(req.DocumentIDs)
}

func (v *PermohonanValidation) ValidateExpansionSubmitRequest(req dto.ExpansionSubmitRequest) error {
	if err := v.validate.Struct(req); err != nil {
		return err
	}
	if req.NpsDelegationStatus != "delegated" && req.NpsDelegationStatus != "returned" {
		return fmt.Errorf("nps_delegation_status must be one of: delegated, returned")
	}
	return validateDocumentIDs(req.DocumentIDs)
}

func (v *PermohonanValidation) ValidateEvidenceSubmitRequest(req dto.EvidenceSubmitRequest) error {
	if err := v.validate.Struct(req); err != nil {
		return err
	}
	return validateDocumentIDs(req.DocumentIDs)
}

func (v *PermohonanValidation) ValidateWOConstructionSubmitRequest(req dto.WOConstructionSubmitRequest) error {
	if err := v.validate.Struct(req); err != nil {
		return err
	}
	return validateDocumentIDs(req.DocumentIDs)
}

func (v *PermohonanValidation) ValidateReservationTeraSubmitRequest(req dto.ReservationTeraSubmitRequest) error {
	if err := v.validate.Struct(req); err != nil {
		return err
	}
	return validateDocumentIDs(req.DocumentIDs)
}

func (v *PermohonanValidation) ValidateConstructionExecutionSubmitRequest(req dto.ConstructionExecutionSubmitRequest) error {
	if err := v.validate.Struct(req); err != nil {
		return err
	}
	if req.WorkflowNode != "pemasangan_tiang" && req.WorkflowNode != "pelaksanaan_konstruksi" {
		return fmt.Errorf("workflow_node must be one of: pemasangan_tiang, pelaksanaan_konstruksi")
	}
	return validateDocumentIDs(req.DocumentIDs)
}

func (v *PermohonanValidation) ValidateEnergizeSubmitRequest(req dto.EnergizeSubmitRequest) error {
	if err := v.validate.Struct(req); err != nil {
		return err
	}
	if strings.TrimSpace(req.OperationResult) == "" {
		return fmt.Errorf("operation_result must not be blank")
	}
	return validateDocumentIDs(req.DocumentIDs)
}

func validateDocumentIDs(ids []string) error {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return fmt.Errorf("document_ids must not contain duplicates")
		}
		seen[id] = struct{}{}
	}
	return nil
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
