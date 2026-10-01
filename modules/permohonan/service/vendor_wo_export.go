package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"gorm.io/gorm"
)

// VendorWOExporter is intentionally separate from PermohonanService so existing
// consumers that only implement the workflow service interface do not gain a
// new requirement.
type VendorWOExporter interface {
	ExportVendorWOPDF(ctx context.Context, id, workflowNode, userID string) ([]byte, string, error)
}

func (s *permohonanService) ExportVendorWOPDF(ctx context.Context, id, workflowNode, userID string) ([]byte, string, error) {
	store, ok := s.workflowDocumentUploader.(generatedWODocumentStore)
	if !ok {
		return nil, "", errors.New("generated document storage is unavailable")
	}

	code := workflow.Code(workflowNode)
	var docType, vendorRole, title, filenamePrefix string
	workScope := "Pekerjaan pemasangan tiang sesuai dokumen teknis"
	switch code {
	case workflow.WOTiang:
		docType, vendorRole, title, filenamePrefix = "wo_vendor_tiang", rbac.RoleVendorTiang, "SURAT PERINTAH KERJA VENDOR TIANG", "Tiang"
	case workflow.WOKonstruksi:
		docType, vendorRole, title, filenamePrefix = "wo_vendor_konstruksi", rbac.RoleVendorKonstruksi, "SURAT PERINTAH KERJA VENDOR KONSTRUKSI", "Konstruksi"
		workScope = "Pekerjaan konstruksi jaringan sesuai dokumen teknis"
	default:
		return nil, "", dto.ErrWorkflowNodeNotFound
	}

	actor, err := s.userRepository.GetUserById(ctx, s.db, userID)
	if err != nil {
		return nil, "", rbac.ErrWorkflowForbidden
	}

	documentID := ""
	filename := ""
	var compensationKey string
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		p, txErr := s.permohonanRepository.GetByIdForUpdate(ctx, tx, id)
		if txErr != nil {
			if errors.Is(txErr, gorm.ErrRecordNotFound) {
				return dto.ErrPermohonanNotFound
			}
			return txErr
		}
		if !rbac.OwnsWorkflowNode(actor.Role, actor.Unit, p.JenisSambungan, p.UlpUnit, code) {
			return rbac.ErrWorkflowForbidden
		}
		var node *entities.PermohonanActivity
		for i := range p.WorkflowNodes {
			if p.WorkflowNodes[i].WorkflowNode == workflowNode {
				node = &p.WorkflowNodes[i]
				break
			}
		}
		if node == nil {
			return dto.ErrWorkflowNodeNotFound
		}
		if node.Status != string(workflow.Completed) {
			return workflow.ErrNotActionable
		}

		var assignment entities.VendorAssignment
		if txErr = tx.Preload("Vendor").Where("permohonan_id = ? AND vendor_role = ?", p.ID, vendorRole).Take(&assignment).Error; txErr != nil {
			if errors.Is(txErr, gorm.ErrRecordNotFound) {
				return workflow.ErrNotActionable
			}
			return txErr
		}
		if assignment.Vendor.Role != vendorRole {
			return workflow.ErrNotActionable
		}

		filterNode := workflowNode
		documents, txErr := s.documentRepository.ListByPermohonan(ctx, tx, id, &filterNode)
		if txErr != nil {
			return txErr
		}
		for _, document := range documents {
			if document.Type == docType && document.Source == "generated" && document.SupersededByID == nil {
				documentID = document.ID.String()
				filename = document.OriginalFilename
				return nil
			}
		}

		issuedAt := time.Now()
		if node.CompletedAt != nil {
			issuedAt = *node.CompletedAt
		}
		data := vendorWOPDFData{
			Title: title, NoPermohonan: p.NoPermohonan,
			JenisPermohonan: p.JenisPermohonan, JenisSambungan: p.JenisSambungan, UlpUnit: p.UlpUnit,
			PelangganNama: p.PelangganNama, PelangganAlamat: p.PelangganAlamat, PelangganNoHp: p.PelangganNoHp,
			VendorNama: assignment.Vendor.Name, VendorEmail: assignment.Vendor.Email,
			VendorTelepon: assignment.Vendor.TelpNumber, IssuedAt: issuedAt, Scope: workScope,
		}
		if p.Tarif != nil {
			data.Tarif = *p.Tarif
		}
		if p.DayaLama != nil {
			data.DayaLama = strconv.FormatInt(*p.DayaLama, 10)
		}
		if p.DayaBaru != nil {
			data.DayaBaru = strconv.FormatInt(*p.DayaBaru, 10)
		}
		var payload map[string]any
		if node.Payload != "" {
			if txErr = json.Unmarshal([]byte(node.Payload), &payload); txErr != nil {
				return fmt.Errorf("decode WO payload: %w", txErr)
			}
		}
		if payload != nil {
			if notes, ok := payload["notes"].(string); ok {
				data.Notes = notes
			}
			if value, ok := payload["estimasi_tanggal_selesai"].(string); ok {
				data.EstimasiSelesai = value
			}
			if value, ok := payload["perlu_pdkb"].(bool); ok {
				if value {
					data.PerluPDKB = "Ya"
				} else {
					data.PerluPDKB = "Tidak"
				}
			}
			if coordinates, ok := payload["location_coordinates"].(map[string]any); ok {
				latitude, latOK := coordinates["latitude"].(float64)
				longitude, lonOK := coordinates["longitude"].(float64)
				if latOK && lonOK {
					data.Coordinates = fmt.Sprintf("%.6f, %.6f", latitude, longitude)
				}
			}
		}
		content, txErr := generateVendorWOPDF(data)
		if txErr != nil {
			return txErr
		}
		filename = fmt.Sprintf("WO_%s_%s.pdf", filenamePrefix, safeFilenamePart(p.NoPermohonan))
		created, txErr := store.CreateGeneratedForWorkflow(ctx, tx, userID, id, workflowNode, docType, filename, content)
		if txErr != nil {
			return txErr
		}
		documentID, compensationKey = created.ID, created.StorageKey
		activityNumber := node.ActivityNumber
		return tx.Create(&entities.ActivityLog{
			PermohonanID: p.ID, WorkflowNode: &workflowNode, ActivityNumber: activityNumber,
			Actor: actor.ID, Action: "wo_pdf_exported",
		}).Error
	})
	if err != nil {
		if compensationKey != "" {
			_ = s.workflowDocumentUploader.DeleteStoredObject(ctx, compensationKey)
		}
		return nil, "", err
	}

	reader, storedFilename, err := store.ReadAuthorizedDocument(ctx, userID, documentID)
	if err != nil {
		return nil, "", err
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", err
	}
	if storedFilename != "" {
		filename = storedFilename
	}
	return content, filename, nil
}
