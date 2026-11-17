package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	userRepo "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestImportedDraftCanBeEditedWithoutStartingWorkflow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, email TEXT, role TEXT, unit TEXT)").Error)
	require.NoError(t, db.Exec(`CREATE TABLE permohonan (id TEXT PRIMARY KEY, no_permohonan TEXT, status TEXT, current_stage INTEGER,
		jenis_permohonan TEXT, jenis_sambungan TEXT, jenis_pembayaran TEXT, tarif TEXT, daya_lama INTEGER, daya_baru INTEGER,
		golongan_tarif TEXT, golongan_tarif_lama TEXT, pembatas_baru TEXT, pembatas_lama TEXT, eis_no_agenda TEXT, ulp_unit TEXT,
		pelanggan_nama TEXT, pelanggan_alamat TEXT, pelanggan_no_hp TEXT, request_date DATETIME,
		tanggal_pembayaran DATETIME, nominal_pembayaran INTEGER, created_by TEXT, created_at DATETIME, updated_at DATETIME)`).Error)
	actor := entities.User{ID: uuid.New(), Name: "Synthetic Operator", Email: "operator@example.invalid", Role: rbac.RolePelayananPelanggan, Unit: "ULP Test"}
	require.NoError(t, db.Exec("INSERT INTO users (id,name,email,role,unit) VALUES (?,?,?,?,?)", actor.ID.String(), actor.Name, actor.Email, actor.Role, actor.Unit).Error)
	payment := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	amount, power := int64(1000), int64(2200)
	method, group := "pascabayar", "R1"
	p := entities.Permohonan{NoPermohonan: "PBPD-2026-1", Status: "draft", CurrentStage: 1,
		JenisPermohonan: rbac.JenisPermohonanPasangBaru, JenisPembayaran: &method,
		UlpUnit: actor.Unit, PelangganNama: "Synthetic Customer", PelangganAlamat: "Synthetic Address",
		PelangganNoHp: "080000000000", RequestDate: payment, TanggalPembayaran: &payment,
		NominalPembayaran: &amount, GolonganTarif: &group, DayaBaru: &power, CreatedBy: actor.ID}
	p.ID = uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO permohonan (id,no_permohonan,status,current_stage,jenis_permohonan,jenis_pembayaran,
		ulp_unit,pelanggan_nama,pelanggan_alamat,pelanggan_no_hp,request_date,tanggal_pembayaran,nominal_pembayaran,
		golongan_tarif,daya_baru,created_by) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, p.ID.String(), p.NoPermohonan,
		p.Status, p.CurrentStage, p.JenisPermohonan, method, p.UlpUnit, p.PelangganNama, p.PelangganAlamat,
		p.PelangganNoHp, payment, payment, amount, group, power, actor.ID.String()).Error)
	s := &permohonanService{db: db, userRepository: userRepo.NewUserRepository(db)}
	category := rbac.TarifRumahTangga
	result, err := s.EditEISDraft(context.Background(), p.ID.String(), actor.ID.String(), dto.EISDraftEditRequest{
		JenisPermohonan: p.JenisPermohonan, JenisSambungan: rbac.JenisSambunganJTR,
		JenisPembayaran: method, Tarif: &category, DayaBaru: &power, GolonganTarif: &group,
		UlpUnit: actor.Unit, PelangganNama: p.PelangganNama, PelangganAlamat: p.PelangganAlamat,
		PelangganNoHp: p.PelangganNoHp, TanggalPembayaran: "2026-11-14", NominalPembayaran: &amount,
	})
	require.NoError(t, err)
	require.Equal(t, "draft", result.Status)
	require.Empty(t, result.WorkflowNodes)
	otherID := uuid.New()
	require.NoError(t, db.Exec("INSERT INTO users (id,name,email,role,unit) VALUES (?,?,?,?,?)", otherID.String(), "Other Operator", "other@example.invalid", actor.Role, actor.Unit).Error)
	_, err = s.EditEISDraft(context.Background(), p.ID.String(), otherID.String(), dto.EISDraftEditRequest{})
	require.ErrorIs(t, err, ErrEISDraft)
	_, err = s.ContinueEISDraft(context.Background(), p.ID.String(), actor.ID.String(), nil)
	require.ErrorIs(t, err, dto.ErrEvidenceRequired)
	var saved entities.Permohonan
	require.NoError(t, db.Take(&saved, "id = ?", p.ID).Error)
	require.Equal(t, "draft", saved.Status)
}

func TestDeleteEISDraftOnlyRemovesOwnUnrelatedDraft(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     string
		otherUser  bool
		related    string
		wantErr    error
		wantExists bool
	}{
		{name: "own draft", status: "draft"},
		{name: "another creator", status: "draft", otherUser: true, wantErr: ErrEISDraftNotFound, wantExists: true},
		{name: "continued draft", status: "in_progress", wantErr: ErrEISDraftConflict, wantExists: true},
		{name: "workflow node", status: "draft", related: "permohonan_activities", wantErr: ErrEISDraftConflict, wantExists: true},
		{name: "activity log", status: "draft", related: "activity_logs", wantErr: ErrEISDraftConflict, wantExists: true},
		{name: "attached document", status: "draft", related: "documents", wantErr: ErrEISDraftConflict, wantExists: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, db.Exec("CREATE TABLE permohonan (id TEXT PRIMARY KEY, created_by TEXT, status TEXT)").Error)
			for _, table := range []string{"permohonan_activities", "activity_logs", "documents", "document_evidence", "vendor_assignments", "activity_revisions", "pdkb_schedules"} {
				require.NoError(t, db.Exec("CREATE TABLE "+table+" (permohonan_id TEXT)").Error)
			}
			requestID, ownerID := uuid.New(), uuid.New()
			require.NoError(t, db.Exec("INSERT INTO permohonan (id, created_by, status) VALUES (?, ?, ?)", requestID, ownerID, tc.status).Error)
			if tc.related != "" {
				require.NoError(t, db.Exec("INSERT INTO "+tc.related+" (permohonan_id) VALUES (?)", requestID).Error)
			}
			callerID := ownerID
			if tc.otherUser {
				callerID = uuid.New()
			}
			s := &permohonanService{db: db}
			err = s.DeleteEISDraft(context.Background(), requestID.String(), callerID.String())
			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.wantErr)
			}
			var remaining int64
			require.NoError(t, db.Table("permohonan").Where("id = ?", requestID).Count(&remaining).Error)
			if tc.wantExists {
				require.EqualValues(t, 1, remaining)
			} else {
				require.Zero(t, remaining)
				require.ErrorIs(t, s.DeleteEISDraft(context.Background(), requestID.String(), ownerID.String()), ErrEISDraftNotFound)
			}
		})
	}
}
