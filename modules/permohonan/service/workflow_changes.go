package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	documentDTO "github.com/pln-colabora/colabora-be/modules/document/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"gorm.io/gorm"
)

type WorkflowChangesService interface {
	SubmitPriority(context.Context, string, string, dto.PrioritySubmitRequest) (dto.PermohonanResponse, error)
	EditSurvey(context.Context, string, string, dto.SurveyRevisionRequest) (dto.PermohonanResponse, error)
	ReviseConstructionDocuments(context.Context, string, string, dto.DocumentRevisionRequest) (dto.PermohonanResponse, error)
	GetRevisions(context.Context, string, string, string) ([]dto.ActivityRevisionResponse, error)
	ListPDKBSchedules(context.Context, string, string, string) ([]dto.PDKBScheduleResponse, error)
	SavePDKBSchedule(context.Context, string, string, dto.PDKBScheduleRequest) (dto.PDKBScheduleResponse, error)
	DeletePDKBSchedule(context.Context, string, string) error
}

var wib = time.FixedZone("WIB", 7*60*60)

func validScheduleDate(value string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", value)
	today := time.Now().In(wib).Format("2006-01-02")
	if err != nil || value < today {
		return time.Time{}, dto.ErrInvalidActivity
	}
	return date, nil
}

func (s *permohonanService) SubmitPriority(ctx context.Context, id, userID string, req dto.PrioritySubmitRequest) (dto.PermohonanResponse, error) {
	if !notesWithinLimit(req.Notes) {
		return dto.PermohonanResponse{}, dto.ErrInvalidActivity
	}
	return s.completeWorkflowNodes(ctx, id, userID, nil, []workflow.Code{workflow.Prioritas}, map[workflow.Code]any{workflow.Prioritas: req}, nil, nil)
}

func (s *permohonanService) scheduleActor(ctx context.Context, userID string, write bool) (entities.User, error) {
	actor, err := s.userRepository.GetUserById(ctx, s.db, userID)
	if err != nil || (actor.Role != rbac.RolePdkb && (write || actor.Role != rbac.RoleKonstruksi)) {
		return entities.User{}, rbac.ErrWorkflowForbidden
	}
	return actor, nil
}

func scheduleResponse(row entities.PDKBSchedule) dto.PDKBScheduleResponse {
	return dto.PDKBScheduleResponse{ID: row.ID.String(), Date: row.ScheduleDate.Format("2006-01-02"), Title: row.Title, PermohonanID: row.PermohonanID, Locked: row.PermohonanID != nil}
}

func (s *permohonanService) ListPDKBSchedules(ctx context.Context, userID, from, to string) ([]dto.PDKBScheduleResponse, error) {
	if _, err := s.scheduleActor(ctx, userID, false); err != nil {
		return nil, err
	}
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, dto.ErrInvalidActivity
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil || end.Before(start) || end.Sub(start) > 366*24*time.Hour {
		return nil, dto.ErrInvalidActivity
	}
	rows, err := (repository.WorkflowChangesRepository{}).ListSchedules(ctx, s.db, start, end)
	if err != nil {
		return nil, err
	}
	result := make([]dto.PDKBScheduleResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, scheduleResponse(row))
	}
	return result, nil
}

func (s *permohonanService) SavePDKBSchedule(ctx context.Context, id, userID string, req dto.PDKBScheduleRequest) (dto.PDKBScheduleResponse, error) {
	actor, err := s.scheduleActor(ctx, userID, true)
	if err != nil {
		return dto.PDKBScheduleResponse{}, err
	}
	date, err := validScheduleDate(req.Date)
	if err != nil || len(strings.TrimSpace(req.Title)) == 0 || len([]rune(req.Title)) > 200 {
		return dto.PDKBScheduleResponse{}, dto.ErrInvalidActivity
	}
	item := entities.PDKBSchedule{ID: uuid.New(), ScheduleDate: date, Title: strings.TrimSpace(req.Title), CreatedBy: actor.ID, UpdatedBy: actor.ID}
	if id != "" {
		item.ID, err = uuid.Parse(id)
		if err != nil {
			return dto.PDKBScheduleResponse{}, dto.ErrInvalidActivity
		}
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		return (repository.WorkflowChangesRepository{}).SaveSchedule(ctx, tx, &item, id == "")
	})
	return scheduleResponse(item), err
}

func (s *permohonanService) DeletePDKBSchedule(ctx context.Context, id, userID string) error {
	if _, err := s.scheduleActor(ctx, userID, true); err != nil {
		return err
	}
	if _, err := uuid.Parse(id); err != nil {
		return dto.ErrInvalidActivity
	}
	return s.db.Transaction(func(tx *gorm.DB) error { return (repository.WorkflowChangesRepository{}).DeleteSchedule(ctx, tx, id) })
}

func (s *permohonanService) EditSurvey(ctx context.Context, id, userID string, req dto.SurveyRevisionRequest) (dto.PermohonanResponse, error) {
	if _, err := time.Parse("2006-01-02", req.SurveyedAt); err != nil || !notesWithinLimit(req.Notes) {
		return dto.PermohonanResponse{}, dto.ErrInvalidActivity
	}
	payload, _ := json.Marshal(map[string]string{"surveyed_at": req.SurveyedAt, "notes": req.Notes})
	return s.reviseNode(ctx, id, userID, workflow.Survei, req.ExpectedRevision, req.DocumentIDs, string(payload))
}

func (s *permohonanService) ReviseConstructionDocuments(ctx context.Context, id, userID string, req dto.DocumentRevisionRequest) (dto.PermohonanResponse, error) {
	return s.reviseNode(ctx, id, userID, workflow.WOKonstruksi, req.ExpectedRevision, req.DocumentIDs, "")
}

func (s *permohonanService) reviseNode(ctx context.Context, id, userID string, code workflow.Code, expected int, ids []string, payload string) (dto.PermohonanResponse, error) {
	if err := validateEvidenceIDs(ids); err != nil {
		return dto.PermohonanResponse{}, err
	}
	if expected < 1 {
		return dto.PermohonanResponse{}, dto.ErrInvalidActivity
	}
	repo := repository.WorkflowChangesRepository{}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		p, err := s.permohonanRepository.GetByIdForUpdate(ctx, tx, id)
		if err != nil {
			return dto.ErrPermohonanNotFound
		}
		actor, err := s.userRepository.GetUserById(ctx, tx, userID)
		if err != nil || !rbac.OwnsWorkflowNode(actor.Role, actor.Unit, p.JenisSambungan, p.UlpUnit, code) {
			return rbac.ErrWorkflowForbidden
		}
		var node *entities.PermohonanActivity
		for i := range p.WorkflowNodes {
			if p.WorkflowNodes[i].WorkflowNode == string(code) {
				node = &p.WorkflowNodes[i]
				break
			}
		}
		if node == nil || node.Status != string(workflow.Completed) {
			return workflow.ErrNotActionable
		}
		revision := max(1, node.Revision)
		if expected != revision {
			return dto.ErrRevisionConflict
		}
		evidence, err := repo.Evidence(ctx, tx, id, string(code))
		if err != nil {
			return err
		}
		associated := map[string]bool{}
		oldIDs := []string{}
		for _, item := range evidence {
			doc, err := s.documentRepository.GetById(ctx, tx, item.DocumentID.String())
			if err != nil {
				return err
			}
			if doc.Source != "uploaded" {
				continue
			}
			associated[item.DocumentID.String()] = true
			if item.IsCurrent {
				oldIDs = append(oldIDs, item.DocumentID.String())
			}
		}
		newIDs := []string{}
		for _, docID := range ids {
			doc, err := s.documentRepository.GetById(ctx, tx, docID)
			if err != nil || doc.Source != "uploaded" || doc.ScanStatus == "infected" || doc.SupersededByID != nil {
				return documentDTO.ErrDocumentNotFound
			}
			if !associated[docID] {
				newIDs = append(newIDs, docID)
			}
		}
		if len(newIDs) > 0 {
			if _, err := s.documentRepository.AttachToWorkflowNode(ctx, tx, newIDs, id, string(code), userID); err != nil {
				return err
			}
		}
		oldJSON, _ := json.Marshal(oldIDs)
		snapshot := entities.ActivityRevision{ID: uuid.New(), PermohonanID: p.ID, WorkflowNode: string(code), Revision: revision, Payload: node.Payload, DocumentIDs: string(oldJSON), RecordedBy: actor.ID, CreatedAt: time.Now()}
		if payload == "" {
			payload = node.Payload
		} else {
			previous := map[string]any{}
			changes := map[string]any{}
			if err := json.Unmarshal([]byte(node.Payload), &previous); err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(payload), &changes); err != nil {
				return err
			}
			for key, value := range changes {
				previous[key] = value
			}
			encoded, err := json.Marshal(previous)
			if err != nil {
				return err
			}
			payload = string(encoded)
		}
		if err := repo.SaveRevision(ctx, tx, snapshot, payload, revision+1); err != nil {
			return err
		}
		if err := repo.SelectEvidence(ctx, tx, id, string(code), ids); err != nil {
			return err
		}
		return repo.LogRevision(ctx, tx, p.ID, actor.ID, string(code), fmt.Sprintf("Revisi %d → %d", revision, revision+1))
	})
	if err != nil {
		return dto.PermohonanResponse{}, err
	}
	return s.GetById(ctx, id, userID)
}

func (s *permohonanService) GetRevisions(ctx context.Context, id, code, userID string) ([]dto.ActivityRevisionResponse, error) {
	if code != string(workflow.Survei) && code != string(workflow.WOKonstruksi) {
		return nil, dto.ErrWorkflowNodeNotFound
	}
	if _, err := s.GetActivity(ctx, id, code, userID); err != nil {
		return nil, err
	}
	rows, err := (repository.WorkflowChangesRepository{}).Revisions(ctx, s.db, id, code)
	if err != nil {
		return nil, err
	}
	result := make([]dto.ActivityRevisionResponse, 0, len(rows))
	for _, row := range rows {
		ids := []string{}
		if err := json.Unmarshal([]byte(row.DocumentIDs), &ids); err != nil {
			return nil, err
		}
		result = append(result, dto.ActivityRevisionResponse{Revision: row.Revision, Payload: json.RawMessage(row.Payload), DocumentIDs: ids, RecordedBy: row.RecordedBy, CreatedAt: row.CreatedAt})
	}
	return result, nil
}
