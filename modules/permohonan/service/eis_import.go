package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/extrame/xls"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/validation"
	"github.com/pln-colabora/colabora-be/pkg/businessday"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrEISDraft = errors.New("draft is not available or already continued")

type EISDraftService interface {
	ImportEIS(context.Context, string, io.ReadSeeker) ([]dto.PermohonanResponse, error)
	EditEISDraft(context.Context, string, string, dto.EISDraftEditRequest) (dto.PermohonanResponse, error)
	ContinueEISDraft(context.Context, string, string, []*multipart.FileHeader) (dto.PermohonanResponse, error)
}

func (s *permohonanService) ImportEIS(ctx context.Context, userID string, reader io.ReadSeeker) ([]dto.PermohonanResponse, error) {
	actor, err := s.userRepository.GetUserById(ctx, s.db, userID)
	if err != nil {
		return nil, err
	}
	if actor.Role != rbac.RolePelayananPelanggan && actor.Role != rbac.RoleNps {
		return nil, dto.ErrCreateForbidden
	}
	book, err := xls.OpenReader(reader, "utf-8")
	if err != nil || book.NumSheets() == 0 {
		return nil, fmt.Errorf("invalid EIS XLS workbook")
	}
	sheet := book.GetSheet(0)
	if sheet == nil || sheet.MaxRow < 1 {
		return nil, fmt.Errorf("EIS sheet has no data")
	}
	header := sheet.Row(0)
	if header == nil {
		return nil, fmt.Errorf("EIS header is missing")
	}
	allowed := []string{"NAMA", "NOTELP_PEMOHON", "TGLBAYAR", "TOTALBIAYA", "JENIS_TRANSAKSI", "NAMAUP", "ALAMAT", "TARIF_LAMA", "DAYA_LAMA", "TARIF", "DAYA", "PAKET_SAR"}
	positions := map[string]int{}
	for col := 0; col < header.LastCol(); col++ {
		name := strings.ToUpper(strings.TrimSpace(header.Col(col)))
		for _, key := range allowed {
			if name == key {
				positions[key] = col
			}
		}
	}
	for _, key := range allowed {
		if _, ok := positions[key]; !ok {
			return nil, fmt.Errorf("EIS header %s is missing", key)
		}
	}
	if sheet.MaxRow > 5000 {
		return nil, fmt.Errorf("EIS import exceeds 5000 rows")
	}
	requests := make([]dto.PermohonanCreateRequest, 0, sheet.MaxRow)
	seen := map[string]bool{}
	var problems []string
	for index := 1; index <= int(sheet.MaxRow); index++ {
		row := sheet.Row(index)
		if row == nil {
			continue
		}
		values := map[string]string{}
		allBlank := true
		for _, key := range allowed {
			values[key] = strings.TrimSpace(row.Col(positions[key]))
			if values[key] != "" {
				allBlank = false
			}
		}
		if allBlank {
			continue
		}
		req, rowErr := mapEISRow(values)
		if rowErr != nil {
			problems = append(problems, fmt.Sprintf("row %d: %v", index+1, rowErr))
			continue
		}
		if req.PelangganNama == "" || req.PelangganNoHp == "" || req.PelangganAlamat == "" || req.UlpUnit == nil {
			problems = append(problems, fmt.Sprintf("row %d: customer name, phone, address, and NAMAUP are required", index+1))
			continue
		}
		if actor.Role == rbac.RolePelayananPelanggan && *req.UlpUnit != actor.Unit {
			problems = append(problems, fmt.Sprintf("row %d: NAMAUP is outside caller ULP", index+1))
			continue
		}
		parts := make([]string, 0, len(allowed))
		for _, field := range allowed {
			parts = append(parts, strings.ToUpper(strings.TrimSpace(values[field])))
		}
		key := strings.Join(parts, "\x00")
		if seen[key] {
			problems = append(problems, fmt.Sprintf("row %d: duplicate in file", index+1))
			continue
		}
		seen[key] = true
		requests = append(requests, req)
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("EIS sheet has no nonempty data rows")
	}
	created := make([]entities.Permohonan, 0, len(requests))
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize EIS imports so duplicate checks also hold across concurrent files.
		if tx.Dialector.Name() == "postgres" {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(72452501)").Error; err != nil {
				return err
			}
		}
		for index, req := range requests {
			payment, _ := time.Parse("2006-01-02", req.TanggalPembayaran)
			var duplicate int64
			if err := tx.Model(&entities.Permohonan{}).Where("pelanggan_nama = ? AND pelanggan_no_hp = ? AND pelanggan_alamat = ? AND tanggal_pembayaran = ? AND nominal_pembayaran = ? AND jenis_permohonan = ? AND ulp_unit = ? AND jenis_pembayaran = ? AND golongan_tarif IS NOT DISTINCT FROM ? AND golongan_tarif_lama IS NOT DISTINCT FROM ? AND daya_baru IS NOT DISTINCT FROM ? AND daya_lama IS NOT DISTINCT FROM ?", req.PelangganNama, req.PelangganNoHp, req.PelangganAlamat, payment, *req.NominalPembayaran, req.JenisPermohonan, *req.UlpUnit, req.JenisPembayaran, req.GolonganTarif, req.GolonganTarifLama, req.DayaBaru, req.DayaLama).Count(&duplicate).Error; err != nil {
				return err
			}
			if duplicate > 0 {
				return fmt.Errorf("row %d: duplicate permohonan", index+2)
			}
			number, err := s.permohonanRepository.AllocateNoPermohonan(ctx, tx, time.Now().Year())
			if err != nil {
				return err
			}
			p := entities.Permohonan{NoPermohonan: number, Status: string(workflow.Draft), CurrentStage: 1,
				JenisPermohonan: req.JenisPermohonan, JenisPembayaran: &req.JenisPembayaran,
				UlpUnit: *req.UlpUnit, PelangganNama: req.PelangganNama, PelangganNoHp: req.PelangganNoHp,
				PelangganAlamat: req.PelangganAlamat, GolonganTarif: req.GolonganTarif,
				GolonganTarifLama: req.GolonganTarifLama, DayaLama: req.DayaLama, DayaBaru: req.DayaBaru,
				TanggalPembayaran: &payment, NominalPembayaran: req.NominalPembayaran,
				RequestDate: time.Now(), SlaBasis: businessday.PaymentBasis, CreatedBy: actor.ID}
			if err := tx.Create(&p).Error; err != nil {
				return err
			}
			created = append(created, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	responses := make([]dto.PermohonanResponse, 0, len(created))
	for _, p := range created {
		response, _ := toPermohonanResponse(p, actor.Role, actor.Unit, time.Now())
		responses = append(responses, response)
	}
	return responses, nil
}

func (s *permohonanService) EditEISDraft(ctx context.Context, id, userID string, req dto.EISDraftEditRequest) (dto.PermohonanResponse, error) {
	actor, err := s.userRepository.GetUserById(ctx, s.db, userID)
	if err != nil {
		return dto.PermohonanResponse{}, err
	}
	var p entities.Permohonan
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND created_by = ? AND status = ?", id, actor.ID, workflow.Draft).Take(&p).Error; err != nil {
			return ErrEISDraft
		}
		updates := map[string]any{"jenis_permohonan": req.JenisPermohonan, "jenis_sambungan": req.JenisSambungan,
			"jenis_pembayaran": req.JenisPembayaran, "tarif": req.Tarif, "daya_lama": req.DayaLama, "daya_baru": req.DayaBaru,
			"golongan_tarif": req.GolonganTarif, "golongan_tarif_lama": req.GolonganTarifLama,
			"pembatas_baru": req.PembatasBaru, "pembatas_lama": req.PembatasLama,
			"ulp_unit": req.UlpUnit, "pelanggan_nama": req.PelangganNama,
			"pelanggan_alamat": req.PelangganAlamat, "pelanggan_no_hp": req.PelangganNoHp,
			"tanggal_pembayaran": req.TanggalPembayaran, "nominal_pembayaran": req.NominalPembayaran}
		if req.PelangganNama == "" || req.PelangganAlamat == "" || req.PelangganNoHp == "" || req.UlpUnit == "" {
			return fmt.Errorf("name, address, phone and ULP are required")
		}
		if actor.Role == rbac.RolePelayananPelanggan && req.UlpUnit != actor.Unit {
			return dto.ErrInvalidULPUnit
		}
		if req.TanggalPembayaran != "" {
			if _, err := time.Parse("2006-01-02", req.TanggalPembayaran); err != nil {
				return err
			}
		} else {
			updates["tanggal_pembayaran"] = nil
		}
		if err := tx.Model(&p).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Take(&p).Error
	})
	if err != nil {
		return dto.PermohonanResponse{}, err
	}
	return toPermohonanResponse(p, actor.Role, actor.Unit, time.Now())
}

func (s *permohonanService) ContinueEISDraft(ctx context.Context, id, userID string, files []*multipart.FileHeader) (dto.PermohonanResponse, error) {
	if len(files) == 0 {
		return dto.PermohonanResponse{}, dto.ErrEvidenceRequired
	}
	actor, err := s.userRepository.GetUserById(ctx, s.db, userID)
	if err != nil {
		return dto.PermohonanResponse{}, err
	}
	var created entities.Permohonan
	var uploadedKeys []string
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND created_by = ? AND status = ?", id, actor.ID, workflow.Draft).Take(&created).Error; err != nil {
			return ErrEISDraft
		}
		req := draftCreateRequest(created)
		if err := validation.NewPermohonanValidation().ValidatePermohonanCreateRequest(req); err != nil {
			return err
		}
		if err := validateDraftForContinue(req); err != nil {
			return err
		}
		working := *s
		working.db = tx
		ulpUnit, err := working.creationULP(ctx, actor.Role, actor.Unit, req)
		if err != nil {
			return err
		}
		if err := working.validateTariffPower(ctx, req); err != nil {
			return err
		}
		payment, err := time.Parse("2006-01-02", req.TanggalPembayaran)
		if err != nil {
			return err
		}
		if err := businessday.Supported(payment); err != nil {
			return err
		}
		rules, err := s.slaRuleRepository.ListByJenis(ctx, tx, req.JenisSambungan)
		if err != nil {
			return err
		}
		created.JenisSambungan, created.UlpUnit = req.JenisSambungan, ulpUnit
		created.Status = string(workflow.Active)
		created.CurrentStage = 2
		created.RequestDate = time.Now()
		created.SlaBasis = businessday.PaymentBasis
		created.TanggalPembayaran = &payment
		nodes, result, err := entities.InitializeWorkflow(created, rules, time.Now())
		if err != nil {
			return err
		}
		created.CurrentStage = result.CurrentStage
		if err := tx.Model(&created).Select("jenis_sambungan", "ulp_unit", "status", "current_stage", "request_date", "sla_basis").Updates(&created).Error; err != nil {
			return err
		}
		if err := tx.Create(&nodes).Error; err != nil {
			return err
		}
		if err := tx.Create(&entities.ActivityLog{PermohonanID: created.ID, Actor: actor.ID, Action: "permohonan_created", WorkflowNode: ptr(string(workflow.Permohonan))}).Error; err != nil {
			return err
		}
		if s.workflowDocumentUploader == nil {
			return dto.ErrCreatePermohonan
		}
		uploadedKeys, err = s.workflowDocumentUploader.UploadForWorkflow(ctx, tx, userID, created.ID.String(), string(workflow.Permohonan), files)
		if err != nil {
			return err
		}
		created.WorkflowNodes = nodes
		if s.workflowNotifier != nil {
			return s.workflowNotifier.QueueInitialAvailable(ctx, tx, created, result)
		}
		return nil
	})
	if err != nil {
		for _, key := range uploadedKeys {
			_ = s.workflowDocumentUploader.DeleteStoredObject(ctx, key)
		}
		return dto.PermohonanResponse{}, err
	}
	return toPermohonanResponse(created, actor.Role, actor.Unit, time.Now())
}

func draftCreateRequest(p entities.Permohonan) dto.PermohonanCreateRequest {
	req := dto.PermohonanCreateRequest{JenisPermohonan: p.JenisPermohonan, JenisSambungan: p.JenisSambungan,
		Tarif: p.Tarif, DayaLama: p.DayaLama, DayaBaru: p.DayaBaru, GolonganTarif: p.GolonganTarif,
		GolonganTarifLama: p.GolonganTarifLama, PembatasBaru: p.PembatasBaru, PembatasLama: p.PembatasLama,
		PelangganNama: p.PelangganNama, PelangganAlamat: p.PelangganAlamat, PelangganNoHp: p.PelangganNoHp,
		NominalPembayaran: p.NominalPembayaran}
	if value := dateString(p.TanggalPembayaran); value != nil {
		req.TanggalPembayaran = *value
	}
	if p.JenisPembayaran != nil {
		req.JenisPembayaran = *p.JenisPembayaran
	}
	if p.JenisSambungan == rbac.JenisSambunganPlgTmKurang5 || p.JenisSambungan == rbac.JenisSambunganPlgTmLebih5 {
		req.UlpUnit = &p.UlpUnit
	}
	return req
}

func validateDraftForContinue(req dto.PermohonanCreateRequest) error {
	if len([]rune(strings.TrimSpace(req.PelangganNama))) < 2 || len([]rune(req.PelangganNama)) > 150 ||
		len([]rune(strings.TrimSpace(req.PelangganAlamat))) < 2 || len([]rune(req.PelangganAlamat)) > 255 ||
		len(req.PelangganNoHp) < 8 || len(req.PelangganNoHp) > 20 {
		return fmt.Errorf("invalid customer name, address or phone")
	}
	if req.PelangganNama == "" || req.PelangganNoHp == "" || req.PelangganAlamat == "" || req.TanggalPembayaran == "" || req.NominalPembayaran == nil || *req.NominalPembayaran <= 0 || req.GolonganTarif == nil || req.Tarif == nil || req.DayaBaru == nil {
		return fmt.Errorf("draft has incomplete required fields")
	}
	if req.JenisPembayaran != "pascabayar" && req.JenisPembayaran != "prabayar" {
		return fmt.Errorf("invalid jenis_pembayaran")
	}
	if req.JenisPermohonan != rbac.JenisPermohonanPasangBaru && req.JenisPermohonan != rbac.JenisPermohonanPerubahanDaya {
		return fmt.Errorf("invalid jenis_permohonan")
	}
	if !workflow.ValidConnection(req.JenisSambungan) {
		return fmt.Errorf("invalid jenis_sambungan")
	}
	if req.JenisPermohonan == rbac.JenisPermohonanPasangBaru && req.DayaLama != nil {
		return fmt.Errorf("daya_lama must be empty for pasang baru")
	}
	if req.JenisPermohonan == rbac.JenisPermohonanPerubahanDaya && (req.DayaLama == nil || *req.DayaLama == *req.DayaBaru) {
		return fmt.Errorf("invalid daya_lama for perubahan daya")
	}
	return nil
}
