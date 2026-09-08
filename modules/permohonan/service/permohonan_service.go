package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	documentDTO "github.com/pln-colabora/colabora-be/modules/document/dto"
	documentRepository "github.com/pln-colabora/colabora-be/modules/document/repository"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	userRepository "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"gorm.io/gorm"
)

type PermohonanService interface {
	Create(ctx context.Context, req dto.PermohonanCreateRequest, userId string) (dto.PermohonanResponse, error)
	GetById(ctx context.Context, id string, userId string) (dto.PermohonanResponse, error)
	List(ctx context.Context, filter *query.PermohonanFilter, userId string) ([]query.Permohonan, int64, error)
	GetActivities(ctx context.Context, id string) ([]dto.WorkflowNodeResponse, error)
	GetLogs(ctx context.Context, id string) ([]dto.ActivityLogResponse, error)
	SubmitSurvey(ctx context.Context, id, userID string, req dto.SurveySubmitRequest) (dto.PermohonanResponse, error)
	SubmitRAB(ctx context.Context, id, userID string, req dto.RABSubmitRequest) (dto.PermohonanResponse, error)
	SubmitExpansion(ctx context.Context, id, userID string, req dto.ExpansionSubmitRequest) (dto.PermohonanResponse, error)
	SubmitWOTiang(ctx context.Context, id, userID string, req dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error)
	SubmitWOConstruction(ctx context.Context, id, userID string, req dto.WOConstructionSubmitRequest) (dto.PermohonanResponse, error)
	SubmitWOAPP(ctx context.Context, id, userID string, req dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error)
	SubmitReservationTera(ctx context.Context, id, userID string, req dto.ReservationTeraSubmitRequest) (dto.PermohonanResponse, error)
	SubmitPKVendor(ctx context.Context, id, userID string, req dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error)
	SubmitWOPDKB(ctx context.Context, id, userID string, req dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error)
}

type permohonanService struct {
	permohonanRepository repository.PermohonanRepository
	slaRuleRepository    repository.SLARuleRepository
	userRepository       userRepository.UserRepository
	documentRepository   documentRepository.DocumentRepository
	db                   *gorm.DB
}

func NewPermohonanService(
	permohonanRepo repository.PermohonanRepository,
	slaRuleRepo repository.SLARuleRepository,
	userRepo userRepository.UserRepository,
	documentRepo documentRepository.DocumentRepository,
	db *gorm.DB,
) PermohonanService {
	return &permohonanService{
		permohonanRepository: permohonanRepo,
		slaRuleRepository:    slaRuleRepo,
		userRepository:       userRepo,
		documentRepository:   documentRepo,
		db:                   db,
	}
}

func (s *permohonanService) SubmitSurvey(ctx context.Context, id, userID string, req dto.SurveySubmitRequest) (dto.PermohonanResponse, error) {
	if _, err := time.Parse("2006-01-02", req.SurveyedAt); err != nil || !notesWithinLimit(req.Notes) {
		return dto.PermohonanResponse{}, dto.ErrInvalidActivity
	}
	payloads := map[workflow.Code]any{
		workflow.Survei: struct {
			SurveyedAt string `json:"surveyed_at"`
			Notes      string `json:"notes,omitempty"`
		}{SurveyedAt: req.SurveyedAt, Notes: req.Notes},
	}
	return s.completeWorkflowNodes(ctx, id, userID, req.DocumentIDs, []workflow.Code{workflow.Survei}, payloads, nil, nil)
}

func (s *permohonanService) SubmitRAB(ctx context.Context, id, userID string, req dto.RABSubmitRequest) (dto.PermohonanResponse, error) {
	if req.KebutuhanTiang == nil || !notesWithinLimit(req.Notes) {
		return dto.PermohonanResponse{}, dto.ErrInvalidActivity
	}
	payloads := map[workflow.Code]any{
		workflow.RAB: struct {
			Notes string `json:"notes,omitempty"`
		}{Notes: req.Notes},
		workflow.KebutuhanTiang: struct {
			KebutuhanTiang bool `json:"kebutuhan_tiang"`
		}{KebutuhanTiang: *req.KebutuhanTiang},
	}
	applyDecision := func(snapshot *workflow.Snapshot) {
		value := *req.KebutuhanTiang
		snapshot.Decisions.KebutuhanTiang = &value
	}
	return s.completeWorkflowNodes(ctx, id, userID, req.DocumentIDs,
		[]workflow.Code{workflow.RAB, workflow.KebutuhanTiang}, payloads, applyDecision, nil)
}

func (s *permohonanService) SubmitExpansion(ctx context.Context, id, userID string, req dto.ExpansionSubmitRequest) (dto.PermohonanResponse, error) {
	delegation := workflow.Delegation(req.NpsDelegationStatus)
	if (delegation != workflow.Delegated && delegation != workflow.Return) || !notesWithinLimit(req.Notes) {
		return dto.PermohonanResponse{}, dto.ErrInvalidActivity
	}
	payloads := map[workflow.Code]any{
		workflow.Perluasan: struct {
			Notes string `json:"notes,omitempty"`
		}{Notes: req.Notes},
		workflow.NPS: struct {
			NPSDelegationStatus string `json:"nps_delegation_status"`
		}{NPSDelegationStatus: req.NpsDelegationStatus},
	}
	applyDecision := func(snapshot *workflow.Snapshot) { snapshot.Decisions.NPS = delegation }
	actions := map[workflow.Code]string{workflow.NPS: "nps_" + req.NpsDelegationStatus}
	return s.completeWorkflowNodes(ctx, id, userID, req.DocumentIDs,
		[]workflow.Code{workflow.Perluasan, workflow.NPS}, payloads, applyDecision, actions)
}

func (s *permohonanService) SubmitWOTiang(ctx context.Context, id, userID string, req dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return s.completeEvidenceNode(ctx, id, userID, req, workflow.WOTiang)
}

func (s *permohonanService) SubmitWOConstruction(ctx context.Context, id, userID string, req dto.WOConstructionSubmitRequest) (dto.PermohonanResponse, error) {
	if req.PerluPdkb == nil || !notesWithinLimit(req.Notes) {
		return dto.PermohonanResponse{}, dto.ErrInvalidActivity
	}
	payloads := map[workflow.Code]any{
		workflow.WOKonstruksi: struct {
			PerluPDKB bool   `json:"perlu_pdkb"`
			Notes     string `json:"notes,omitempty"`
		}{PerluPDKB: *req.PerluPdkb, Notes: req.Notes},
	}
	applyDecision := func(snapshot *workflow.Snapshot) {
		value := *req.PerluPdkb
		snapshot.Decisions.PerluPDKB = &value
	}
	decision := "not_required"
	if *req.PerluPdkb {
		decision = "required"
	}
	actions := map[workflow.Code]string{workflow.WOKonstruksi: "pdkb_" + decision}
	return s.completeWorkflowNodes(ctx, id, userID, req.DocumentIDs,
		[]workflow.Code{workflow.WOKonstruksi}, payloads, applyDecision, actions)
}

func (s *permohonanService) SubmitWOAPP(ctx context.Context, id, userID string, req dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return s.completeEvidenceNode(ctx, id, userID, req, workflow.WOAPP)
}

func (s *permohonanService) SubmitReservationTera(ctx context.Context, id, userID string, req dto.ReservationTeraSubmitRequest) (dto.PermohonanResponse, error) {
	if !notesWithinLimit(req.ReservationNotes, req.TeraNotes) {
		return dto.PermohonanResponse{}, dto.ErrInvalidActivity
	}
	payloads := map[workflow.Code]any{
		workflow.Reservasi: struct {
			Notes string `json:"notes,omitempty"`
		}{Notes: req.ReservationNotes},
		workflow.Tera: struct {
			Notes string `json:"notes,omitempty"`
		}{Notes: req.TeraNotes},
	}
	return s.completeWorkflowNodes(ctx, id, userID, req.DocumentIDs,
		[]workflow.Code{workflow.Reservasi, workflow.Tera}, payloads, nil, nil)
}

func (s *permohonanService) SubmitPKVendor(ctx context.Context, id, userID string, req dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return s.completeEvidenceNode(ctx, id, userID, req, workflow.PKVendor)
}

func (s *permohonanService) SubmitWOPDKB(ctx context.Context, id, userID string, req dto.EvidenceSubmitRequest) (dto.PermohonanResponse, error) {
	return s.completeEvidenceNode(ctx, id, userID, req, workflow.WOPDKB)
}

func (s *permohonanService) completeEvidenceNode(
	ctx context.Context, id, userID string, req dto.EvidenceSubmitRequest, code workflow.Code,
) (dto.PermohonanResponse, error) {
	if !notesWithinLimit(req.Notes) {
		return dto.PermohonanResponse{}, dto.ErrInvalidActivity
	}
	payloads := map[workflow.Code]any{
		code: struct {
			Notes string `json:"notes,omitempty"`
		}{Notes: req.Notes},
	}
	return s.completeWorkflowNodes(ctx, id, userID, req.DocumentIDs, []workflow.Code{code}, payloads, nil, nil)
}

func (s *permohonanService) completeWorkflowNodes(
	ctx context.Context,
	id, userID string,
	documentIDs []string,
	codes []workflow.Code,
	payloads map[workflow.Code]any,
	applyDecision func(*workflow.Snapshot),
	actions map[workflow.Code]string,
) (dto.PermohonanResponse, error) {
	if err := validateEvidenceIDs(documentIDs); err != nil {
		return dto.PermohonanResponse{}, err
	}

	encodedPayloads := make(map[workflow.Code]string, len(payloads))
	for code, payload := range payloads {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return dto.PermohonanResponse{}, dto.ErrSubmitActivity
		}
		encodedPayloads[code] = string(encoded)
	}

	var updated entities.Permohonan
	var actor entities.User
	err := s.db.Transaction(func(tx *gorm.DB) error {
		permohonan, err := s.permohonanRepository.GetByIdForUpdate(ctx, tx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return dto.ErrPermohonanNotFound
			}
			return dto.ErrSubmitActivity
		}
		actor, err = s.userRepository.GetUserById(ctx, tx, userID)
		if err != nil {
			return rbac.ErrWorkflowForbidden
		}

		snapshot := permohonan.WorkflowSnapshot()
		if applyDecision != nil {
			applyDecision(&snapshot)
		}
		before, err := workflow.Evaluate(snapshot)
		if err != nil {
			return err
		}
		var result workflow.Result
		for _, code := range codes {
			if err := rbac.AuthorizeWorkflowNode(actor.Role, actor.Unit, permohonan.UlpUnit, snapshot, code); err != nil {
				return err
			}
			var transitionErr error
			snapshot, result, transitionErr = workflow.Transition(snapshot, code, workflow.Completed)
			if transitionErr != nil {
				return transitionErr
			}
		}

		for _, code := range codes {
			rows, attachErr := s.documentRepository.AttachToWorkflowNode(
				ctx, tx, documentIDs, permohonan.ID.String(), string(code), actor.ID.String(),
			)
			if attachErr != nil {
				if errors.Is(attachErr, documentRepository.ErrDocumentNotFound) {
					return documentDTO.ErrDocumentNotFound
				}
				if errors.Is(attachErr, documentRepository.ErrDocumentAttachConflict) {
					return documentDTO.ErrDocumentAlreadyAttached
				}
				return attachErr
			}
			if rows != int64(len(documentIDs)) {
				return documentDTO.ErrDocumentAlreadyAttached
			}
		}

		permohonan.CurrentStage = result.CurrentStage
		permohonan.Status = string(result.Status)
		permohonan.KebutuhanTiang = cloneBool(snapshot.Decisions.KebutuhanTiang)
		permohonan.PerluPdkb = cloneBool(snapshot.Decisions.PerluPDKB)
		if snapshot.Decisions.NPS == "" {
			permohonan.NpsDelegationStatus = nil
		} else {
			value := string(snapshot.Decisions.NPS)
			permohonan.NpsDelegationStatus = &value
		}

		now := time.Now()
		completed := make(map[workflow.Code]struct{}, len(codes))
		for _, code := range codes {
			completed[code] = struct{}{}
		}
		found := 0
		for i := range permohonan.WorkflowNodes {
			code := workflow.Code(permohonan.WorkflowNodes[i].WorkflowNode)
			permohonan.WorkflowNodes[i].Status = string(result.Nodes[code])
			if _, ok := completed[code]; ok {
				found++
				actorID := actor.ID
				completedAt := now
				permohonan.WorkflowNodes[i].Payload = encodedPayloads[code]
				permohonan.WorkflowNodes[i].CompletedBy = &actorID
				permohonan.WorkflowNodes[i].CompletedAt = &completedAt
			}
		}
		if found != len(codes) {
			return dto.ErrSubmitActivity
		}

		logs := make([]entities.ActivityLog, 0, len(codes))
		for _, code := range codes {
			definition, _ := workflow.Lookup(code)
			action := "node_completed"
			if custom := actions[code]; custom != "" {
				action = custom
			}
			codeValue := string(code)
			logs = append(logs, entities.ActivityLog{
				PermohonanID: permohonan.ID, ActivityNumber: definition.ActivityNumber,
				Actor: actor.ID, Action: action, WorkflowNode: &codeValue,
			})
		}
		for _, definition := range workflow.Definitions() {
			if before.Nodes[definition.Code] == workflow.Skipped || result.Nodes[definition.Code] != workflow.Skipped {
				continue
			}
			codeValue := string(definition.Code)
			logs = append(logs, entities.ActivityLog{
				PermohonanID: permohonan.ID, ActivityNumber: definition.ActivityNumber,
				Actor: actor.ID, Action: "node_skipped", WorkflowNode: &codeValue,
			})
		}
		if err := s.permohonanRepository.SaveWorkflow(ctx, tx, permohonan, logs); err != nil {
			return dto.ErrSubmitActivity
		}
		updated = permohonan
		return nil
	})
	if err != nil {
		return dto.PermohonanResponse{}, err
	}

	response, err := toPermohonanResponse(updated, actor.Role, actor.Unit, time.Now())
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrSubmitActivity
	}
	return response, nil
}

func validateEvidenceIDs(documentIDs []string) error {
	if len(documentIDs) == 0 {
		return dto.ErrEvidenceRequired
	}
	seen := make(map[string]struct{}, len(documentIDs))
	for _, id := range documentIDs {
		if _, err := uuid.Parse(id); err != nil {
			return dto.ErrInvalidEvidence
		}
		if _, ok := seen[id]; ok {
			return dto.ErrInvalidEvidence
		}
		seen[id] = struct{}{}
	}
	return nil
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func notesWithinLimit(values ...string) bool {
	for _, value := range values {
		if utf8.RuneCountInString(value) > 2000 {
			return false
		}
	}
	return true
}

func (s *permohonanService) Create(ctx context.Context, req dto.PermohonanCreateRequest, userId string) (dto.PermohonanResponse, error) {
	creator, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrCreatePermohonan
	}

	ulpUnit, err := s.creationULP(ctx, creator.Role, creator.Unit, req)
	if err != nil {
		return dto.PermohonanResponse{}, err
	}

	requestDate := time.Now()
	year := requestDate.Year()

	prefix := fmt.Sprintf("PBPD-%d-", year)
	count, err := s.permohonanRepository.CountByNoPermohonanPrefix(ctx, s.db, prefix)
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrCreatePermohonan
	}
	noPermohonan := fmt.Sprintf("%s%04d", prefix, count+1)

	rules, err := s.slaRuleRepository.ListByJenis(ctx, s.db, req.JenisSambungan)
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrCreatePermohonan
	}

	permohonan := entities.Permohonan{
		NoPermohonan:    noPermohonan,
		JenisPermohonan: req.JenisPermohonan,
		JenisSambungan:  req.JenisSambungan,
		UlpUnit:         ulpUnit,
		PelangganNama:   req.PelangganNama,
		PelangganAlamat: req.PelangganAlamat,
		PelangganNoHp:   req.PelangganNoHp,
		RequestDate:     requestDate,
		CurrentStage:    2,
		Status:          string(workflow.Active),
		CreatedBy:       creator.ID,
	}

	activities, result, err := entities.InitializeWorkflow(permohonan, rules, time.Now())
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrCreatePermohonan
	}
	permohonan.CurrentStage = result.CurrentStage

	log := entities.ActivityLog{
		Actor:        creator.ID,
		Action:       "permohonan_created",
		WorkflowNode: ptr(string(workflow.Permohonan)),
	}

	var created entities.Permohonan
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var txErr error
		created, txErr = s.permohonanRepository.Create(ctx, tx, permohonan, activities, log)
		return txErr
	})
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrCreatePermohonan
	}

	created.WorkflowNodes = activities
	return toPermohonanResponse(created, creator.Role, creator.Unit, time.Now())
}

func (s *permohonanService) GetById(ctx context.Context, id string, userId string) (dto.PermohonanResponse, error) {
	permohonan, err := s.permohonanRepository.GetById(ctx, s.db, id)
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrPermohonanNotFound
	}

	requester, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrPermohonanNotFound
	}

	response, err := toPermohonanResponse(permohonan, requester.Role, requester.Unit, time.Now())
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrGetPermohonanById
	}
	return response, nil
}

func (s *permohonanService) List(ctx context.Context, filter *query.PermohonanFilter, userId string) ([]query.Permohonan, int64, error) {
	requester, err := s.userRepository.GetUserById(ctx, s.db, userId)
	if err != nil {
		return nil, 0, err
	}
	if filter.Scope == "mine" {
		filter.CurrentRole = requester.Role
		filter.CurrentUnit = requester.Unit
	}

	results, total, err := s.permohonanRepository.List(ctx, s.db, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range results {
		p := entities.Permohonan{
			JenisSambungan:      results[i].JenisSambungan,
			UlpUnit:             results[i].UlpUnit,
			KebutuhanTiang:      results[i].KebutuhanTiang,
			NpsDelegationStatus: results[i].NpsDelegationStatus,
			PerluPdkb:           results[i].PerluPdkb,
			WorkflowNodes:       results[i].WorkflowNodes,
		}
		evaluated, evaluateErr := workflow.Evaluate(p.WorkflowSnapshot())
		if evaluateErr != nil {
			return nil, 0, evaluateErr
		}
		results[i].CurrentStage = evaluated.CurrentStage
		results[i].Status = string(evaluated.Status)
		results[i].AvailableActions, err = availableActions(requester.Role, requester.Unit, p)
		if err != nil {
			return nil, 0, err
		}
	}
	return results, total, nil
}

func (s *permohonanService) GetActivities(ctx context.Context, id string) ([]dto.WorkflowNodeResponse, error) {
	p, err := s.permohonanRepository.GetById(ctx, s.db, id)
	if err != nil {
		return nil, dto.ErrPermohonanNotFound
	}
	nodes, err := s.permohonanRepository.ListWorkflowNodes(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	p.WorkflowNodes = nodes
	evaluated, err := workflow.Evaluate(p.WorkflowSnapshot())
	if err != nil {
		return nil, err
	}
	return workflowNodeResponses(nodes, evaluated, time.Now()), nil
}

func (s *permohonanService) GetLogs(ctx context.Context, id string) ([]dto.ActivityLogResponse, error) {
	if _, err := s.permohonanRepository.GetById(ctx, s.db, id); err != nil {
		return nil, dto.ErrPermohonanNotFound
	}
	logs, err := s.permohonanRepository.ListActivityLogs(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	responses := make([]dto.ActivityLogResponse, 0, len(logs))
	for _, log := range logs {
		responses = append(responses, dto.ActivityLogResponse{
			ID: log.ID.String(), WorkflowNode: log.WorkflowNode, ActivityNumber: log.ActivityNumber,
			Actor: log.Actor.String(), Action: log.Action, Detail: log.Detail, CreatedAt: log.CreatedAt.Format(time.RFC3339),
		})
	}
	return responses, nil
}

func (s *permohonanService) creationULP(ctx context.Context, role, callerUnit string, req dto.PermohonanCreateRequest) (string, error) {
	switch req.JenisSambungan {
	case rbac.JenisSambunganJTR, rbac.JenisSambunganJTMGardu:
		if role != rbac.RolePelayananPelanggan {
			return "", dto.ErrCreateForbidden
		}
		if req.UlpUnit != nil || strings.TrimSpace(callerUnit) == "" {
			return "", dto.ErrInvalidULPUnit
		}
		return strings.TrimSpace(callerUnit), nil
	case rbac.JenisSambunganPlgTmKurang5, rbac.JenisSambunganPlgTmLebih5:
		if role != rbac.RoleNps {
			return "", dto.ErrCreateForbidden
		}
		if req.UlpUnit == nil || strings.TrimSpace(*req.UlpUnit) == "" {
			return "", dto.ErrInvalidULPUnit
		}
		unit := strings.TrimSpace(*req.UlpUnit)
		// There is no unit-master table yet. A ULP is configured when at least one
		// ULP-scoped operational account is assigned to it.
		exists, err := s.userRepository.ExistsByUnitAndRoles(ctx, s.db, unit, []string{rbac.RolePelayananPelanggan, rbac.RoleTeknik})
		if err != nil {
			return "", dto.ErrCreatePermohonan
		}
		if !exists {
			return "", dto.ErrInvalidULPUnit
		}
		return unit, nil
	default:
		return "", dto.ErrCreatePermohonan
	}
}

func toPermohonanResponse(p entities.Permohonan, role, unit string, now time.Time) (dto.PermohonanResponse, error) {
	evaluated, err := workflow.Evaluate(p.WorkflowSnapshot())
	if err != nil {
		return dto.PermohonanResponse{}, err
	}
	actions, err := availableActions(role, unit, p)
	if err != nil {
		return dto.PermohonanResponse{}, err
	}
	return dto.PermohonanResponse{
		ID:                  p.ID.String(),
		NoPermohonan:        p.NoPermohonan,
		JenisPermohonan:     p.JenisPermohonan,
		JenisSambungan:      p.JenisSambungan,
		UlpUnit:             p.UlpUnit,
		PelangganNama:       p.PelangganNama,
		PelangganAlamat:     p.PelangganAlamat,
		PelangganNoHp:       p.PelangganNoHp,
		RequestDate:         p.RequestDate.Format("2006-01-02"),
		CurrentStage:        evaluated.CurrentStage,
		Status:              string(evaluated.Status),
		KebutuhanTiang:      p.KebutuhanTiang,
		NpsDelegationStatus: p.NpsDelegationStatus,
		PerluPdkb:           p.PerluPdkb,
		CreatedBy:           p.CreatedBy.String(),
		WorkflowNodes:       workflowNodeResponses(p.WorkflowNodes, evaluated, now),
		AvailableActions:    actions,
	}, nil
}

func workflowNodeResponses(nodes []entities.PermohonanActivity, evaluated workflow.Result, now time.Time) []dto.WorkflowNodeResponse {
	responses := make([]dto.WorkflowNodeResponse, 0, len(nodes))
	byCode := make(map[workflow.Code]entities.PermohonanActivity, len(nodes))
	for _, node := range nodes {
		byCode[workflow.Code(node.WorkflowNode)] = node
	}
	for _, definition := range workflow.Definitions() {
		node, ok := byCode[definition.Code]
		if !ok {
			continue
		}
		status := evaluated.Nodes[workflow.Code(node.WorkflowNode)]
		var deadline, completedBy, completedAt *string
		if node.SlaDeadline != nil {
			value := node.SlaDeadline.Format("2006-01-02")
			deadline = &value
		}
		if node.CompletedBy != nil {
			value := node.CompletedBy.String()
			completedBy = &value
		}
		if node.CompletedAt != nil {
			value := node.CompletedAt.Format(time.RFC3339)
			completedAt = &value
		}
		payload := json.RawMessage(node.Payload)
		if !json.Valid(payload) {
			payload = json.RawMessage(`{}`)
		}
		responses = append(responses, dto.WorkflowNodeResponse{
			WorkflowNode: node.WorkflowNode, ActivityNumber: node.ActivityNumber, StageNumber: node.StageNumber,
			Status: string(status), SlaDeadline: deadline, SlaStatus: slaStatus(node, status, now), Payload: payload,
			CompletedBy: completedBy, CompletedAt: completedAt,
		})
	}
	return responses
}

func slaStatus(node entities.PermohonanActivity, status workflow.Status, now time.Time) string {
	if node.SlaDeadline == nil || status == workflow.Skipped {
		return "none"
	}
	reference := now
	if node.CompletedAt != nil {
		reference = *node.CompletedAt
	}
	deadlineDate := time.Date(node.SlaDeadline.Year(), node.SlaDeadline.Month(), node.SlaDeadline.Day(), 0, 0, 0, 0, node.SlaDeadline.Location())
	referenceDate := time.Date(reference.Year(), reference.Month(), reference.Day(), 0, 0, 0, 0, node.SlaDeadline.Location())
	if referenceDate.After(deadlineDate) {
		return "overdue"
	}
	if status == workflow.Completed {
		return "on_time"
	}
	dueSoon := referenceDate.AddDate(0, 0, 2)
	if !deadlineDate.After(dueSoon) {
		return "due_soon"
	}
	return "on_time"
}

func availableActions(role, unit string, p entities.Permohonan) ([]dto.AvailableAction, error) {
	definitions, err := rbac.AvailableActions(role, unit, p.UlpUnit, p.WorkflowSnapshot())
	if err != nil {
		return nil, err
	}
	responses := make([]dto.AvailableAction, 0, len(definitions))
	for _, definition := range definitions {
		responses = append(responses, dto.AvailableAction{
			WorkflowNode: string(definition.Code), ActivityNumber: definition.ActivityNumber,
			StageNumber: definition.Stage, Method: "POST", Path: actionPath(definition.Code),
		})
	}
	return responses, nil
}

func actionPath(code workflow.Code) string {
	paths := map[workflow.Code]string{
		workflow.Permohonan: "/api/permohonan", workflow.Survei: "/api/permohonan/{id}/survei",
		workflow.RAB: "/api/permohonan/{id}/rab-kko-kkf", workflow.KebutuhanTiang: "/api/permohonan/{id}/rab-kko-kkf",
		workflow.Perluasan: "/api/permohonan/{id}/permohonan-perluasan", workflow.NPS: "/api/permohonan/{id}/permohonan-perluasan",
		workflow.WOTiang: "/api/permohonan/{id}/wo-vendor/tiang", workflow.WOKonstruksi: "/api/permohonan/{id}/wo-vendor/konstruksi",
		workflow.WOAPP: "/api/permohonan/{id}/wo-vendor/app", workflow.Reservasi: "/api/permohonan/{id}/reservasi-material",
		workflow.Tera: "/api/permohonan/{id}/reservasi-material", workflow.PKVendor: "/api/permohonan/{id}/pk-vendor",
		workflow.WOPDKB: "/api/permohonan/{id}/wo-pdkb", workflow.PemasanganTiang: "/api/permohonan/{id}/pelaksanaan-konstruksi",
		workflow.Konstruksi: "/api/permohonan/{id}/pelaksanaan-konstruksi", workflow.DokumentasiPDKB: "/api/permohonan/{id}/pdkb-dokumentasi",
		workflow.Energize: "/api/permohonan/{id}/energize-jaringan", workflow.SRAPP: "/api/permohonan/{id}/pemasangan-sr-app",
		workflow.PDL: "/api/permohonan/{id}/closing", workflow.AIL: "/api/permohonan/{id}/closing", workflow.Selesai: "/api/permohonan/{id}/closing",
	}
	return paths[code]
}

func ptr(value string) *string { return &value }
