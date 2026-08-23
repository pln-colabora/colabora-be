package service

import (
	"context"
	"fmt"
	"time"

	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	userRepository "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"gorm.io/gorm"
)

type PermohonanService interface {
	Create(ctx context.Context, req dto.PermohonanCreateRequest, userId string) (dto.PermohonanResponse, error)
	GetById(ctx context.Context, id string, userId string) (dto.PermohonanResponse, error)
	List(ctx context.Context, filter *query.PermohonanFilter, userId string) ([]query.Permohonan, int64, error)
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

	if creator.Role != rbac.RolePelayananPelanggan {
		return dto.PermohonanResponse{}, dto.ErrOnlyPelayananPelangganCanCreate
	}

	requestDate := time.Now()
	year := requestDate.Year()

	prefix := fmt.Sprintf("PBPD-%d-", year)
	count, err := s.permohonanRepository.CountByNoPermohonanPrefix(ctx, s.db, prefix)
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrCreatePermohonan
	}
	noPermohonan := fmt.Sprintf("%s%04d", prefix, count+1)

	activity1Rule, err := s.slaRuleRepository.GetByActivityAndJenis(ctx, s.db, 1, req.JenisSambungan)
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrCreatePermohonan
	}
	activity2Rule, err := s.slaRuleRepository.GetByActivityAndJenis(ctx, s.db, 2, req.JenisSambungan)
	if err != nil {
		return dto.PermohonanResponse{}, dto.ErrCreatePermohonan
	}

	permohonan := entities.Permohonan{
		NoPermohonan:    noPermohonan,
		JenisPermohonan: req.JenisPermohonan,
		JenisSambungan:  req.JenisSambungan,
		UlpUnit:         creator.Unit,
		PelangganNama:   req.PelangganNama,
		PelangganAlamat: req.PelangganAlamat,
		PelangganNoHp:   req.PelangganNoHp,
		RequestDate:     requestDate,
		CurrentStage:    2,
		Status:          "in_progress",
		CreatedBy:       creator.ID,
	}

	now := time.Now()
	activities := []entities.PermohonanActivity{
		{
			ActivityNumber: 1,
			StageNumber:    rbac.ActivityStageMap[1],
			Status:         "done",
			SlaDeadline:    requestDate.AddDate(0, 0, int(activity1Rule.OffsetDays)),
			CompletedBy:    &creator.ID,
			CompletedAt:    &now,
		},
		{
			ActivityNumber: 2,
			StageNumber:    rbac.ActivityStageMap[2],
			Status:         "in_progress",
			SlaDeadline:    requestDate.AddDate(0, 0, int(activity2Rule.OffsetDays)),
		},
	}

	log := entities.ActivityLog{
		Actor:  creator.ID,
		Action: "permohonan_created",
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

	return toPermohonanResponse(created, rbac.OwnsStage(creator.Role, creator.Unit, created.CurrentStage, created.JenisSambungan, created.UlpUnit, created.OwnerFnOverride)), nil
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

	canAct := rbac.OwnsStage(requester.Role, requester.Unit, permohonan.CurrentStage, permohonan.JenisSambungan, permohonan.UlpUnit, permohonan.OwnerFnOverride)

	return toPermohonanResponse(permohonan, canAct), nil
}

func (s *permohonanService) List(ctx context.Context, filter *query.PermohonanFilter, userId string) ([]query.Permohonan, int64, error) {
	if filter.Scope == "mine" {
		requester, err := s.userRepository.GetUserById(ctx, s.db, userId)
		if err != nil {
			return nil, 0, err
		}
		filter.CurrentRole = requester.Role
		filter.CurrentUnit = requester.Unit
	}

	return s.permohonanRepository.List(ctx, s.db, filter)
}

func toPermohonanResponse(p entities.Permohonan, canAct bool) dto.PermohonanResponse {
	return dto.PermohonanResponse{
		ID:              p.ID.String(),
		NoPermohonan:    p.NoPermohonan,
		JenisPermohonan: p.JenisPermohonan,
		JenisSambungan:  p.JenisSambungan,
		UlpUnit:         p.UlpUnit,
		PelangganNama:   p.PelangganNama,
		PelangganAlamat: p.PelangganAlamat,
		PelangganNoHp:   p.PelangganNoHp,
		RequestDate:     p.RequestDate.Format("2006-01-02"),
		CurrentStage:    p.CurrentStage,
		Status:          p.Status,
		KebutuhanTiang:  p.KebutuhanTiang,
		NpsKeputusan:    p.NpsKeputusan,
		PerluPdkb:       p.PerluPdkb,
		OwnerFnOverride: p.OwnerFnOverride,
		CreatedBy:       p.CreatedBy.String(),
		CanAct:          canAct,
	}
}
