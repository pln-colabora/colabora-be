package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"gorm.io/gorm"
)

func (s *permohonanService) AssignVendor(ctx context.Context, id, userID string, req dto.VendorAssignmentRequest) (dto.PermohonanResponse, error) {
	if _, err := uuid.Parse(req.VendorID); err != nil {
		return dto.PermohonanResponse{}, dto.ErrInvalidActivity
	}
	var p entities.Permohonan
	var actor entities.User
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		p, err = s.permohonanRepository.GetByIdForUpdate(ctx, tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.ErrPermohonanNotFound
		}
		if err != nil {
			return dto.ErrSubmitActivity
		}
		actor, err = s.userRepository.GetUserById(ctx, tx, userID)
		if err != nil {
			return rbac.ErrWorkflowForbidden
		}
		// Authorize the requested vendor category before resolving a supplied user id.
		var wo workflow.Code
		switch req.VendorRole {
		case rbac.RoleVendorTiang:
			wo = workflow.WOTiang
		case rbac.RoleVendorKonstruksi:
			wo = workflow.WOKonstruksi
		case rbac.RoleVendorSrApp:
			owner, _ := workflow.Owner(workflow.SRAPP, p.JenisSambungan)
			if owner != req.VendorRole {
				return dto.ErrInvalidActivity
			}
			wo = workflow.WOAPP
		default:
			return dto.ErrInvalidActivity
		}
		if !rbac.OwnsWorkflowNode(actor.Role, actor.Unit, p.JenisSambungan, p.UlpUnit, wo) {
			return rbac.ErrWorkflowForbidden
		}
		result, err := workflow.Evaluate(p.WorkflowSnapshot())
		if err != nil {
			return err
		}
		if result.Status != workflow.Active || result.Nodes[workflow.NPS] != workflow.Completed || result.Nodes[wo] == workflow.Skipped {
			return workflow.ErrNotActionable
		}
		vendor, err := s.userRepository.GetUserById(ctx, tx, req.VendorID)
		if err != nil || vendor.Role != req.VendorRole {
			return dto.ErrInvalidActivity
		}
		err = s.vendorAssignmentRepository.Assign(ctx, tx, entities.VendorAssignment{
			PermohonanID: p.ID, VendorID: vendor.ID, VendorRole: vendor.Role, AssignedBy: actor.ID,
		})
		if errors.Is(err, repository.ErrVendorAlreadyAssigned) {
			return workflow.ErrNotActionable
		}
		if err != nil {
			return dto.ErrSubmitActivity
		}
		return nil
	})
	if err != nil {
		return dto.PermohonanResponse{}, err
	}
	return s.GetById(ctx, id, userID)
}
