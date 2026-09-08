package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pln-colabora/colabora-be/database/entities"
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
}

type permohonanService struct {
	permohonanRepository repository.PermohonanRepository
	slaRuleRepository    repository.SLARuleRepository
	userRepository       userRepository.UserRepository
	db                   *gorm.DB
}

func NewPermohonanService(
	permohonanRepo repository.PermohonanRepository,
	slaRuleRepo repository.SLARuleRepository,
	userRepo userRepository.UserRepository,
	db *gorm.DB,
) PermohonanService {
	return &permohonanService{
		permohonanRepository: permohonanRepo,
		slaRuleRepository:    slaRuleRepo,
		userRepository:       userRepo,
		db:                   db,
	}
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
