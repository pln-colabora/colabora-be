package service

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	permohonanRepo "github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	userRepo "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func eisXLSX(t *testing.T, unit string) []byte {
	t.Helper()
	book := excelize.NewFile()
	defer book.Close()
	header := []interface{}{"NAMA", "NOTELP_PEMOHON", "TGLBAYAR", "TOTALBIAYA", "JENIS_TRANSAKSI", "NAMAUP", "ALAMAT", "TARIF_LAMA", "DAYA_LAMA", "TARIF", "DAYA", "PAKET_SAR"}
	data := []interface{}{"Synthetic Customer", "080000000000", 46337.5, 100000, "PASANG BARU", unit, "Synthetic Address", "", "", "R1", 2200, "Pascabayar"}
	require.NoError(t, book.SetSheetRow("Sheet1", "A1", &header))
	require.NoError(t, book.SetSheetRow("Sheet1", "A2", &data))
	var out bytes.Buffer
	_, err := book.WriteTo(&out)
	require.NoError(t, err)
	return out.Bytes()
}

func eisImportService(t *testing.T) (*permohonanService, *gorm.DB, entities.User) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, email TEXT, role TEXT, unit TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE permohonan_number_counters (request_year INTEGER PRIMARY KEY, last_number INTEGER NOT NULL)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE permohonan (
		id TEXT PRIMARY KEY, no_permohonan TEXT, jenis_permohonan TEXT, jenis_sambungan TEXT,
		jenis_pembayaran TEXT, tarif TEXT, daya_lama INTEGER, daya_baru INTEGER,
		golongan_tarif TEXT, golongan_tarif_lama TEXT, pembatas_baru TEXT, pembatas_lama TEXT, eis_no_agenda TEXT,
		priority TEXT, ulp_unit TEXT, pelanggan_nama TEXT, pelanggan_alamat TEXT,
		pelanggan_no_hp TEXT, request_date DATETIME, tanggal_pembayaran DATETIME,
		nominal_pembayaran INTEGER, sla_basis TEXT, current_stage INTEGER, status TEXT,
		kebutuhan_tiang BOOLEAN, nps_delegation_status TEXT, perlu_pdkb BOOLEAN,
		legacy_priority_exempt BOOLEAN, energize_status TEXT, created_by TEXT,
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	actor := entities.User{ID: uuid.New(), Name: "Synthetic Operator", Email: "operator@example.invalid", Role: rbac.RolePelayananPelanggan, Unit: "ULP Taman"}
	require.NoError(t, db.Exec("INSERT INTO users (id,name,email,role,unit) VALUES (?,?,?,?,?)", actor.ID.String(), actor.Name, actor.Email, actor.Role, actor.Unit).Error)
	s := &permohonanService{db: db, userRepository: userRepo.NewUserRepository(db), permohonanRepository: permohonanRepo.NewPermohonanRepository(db)}
	return s, db, actor
}

func TestEISXLSXImportCanonicalizesMatchingULP(t *testing.T) {
	s, db, actor := eisImportService(t)

	result, err := s.ImportEIS(context.Background(), actor.ID.String(), ".xlsx", bytes.NewReader(eisXLSX(t, "ULP TAMAN")))
	require.NoError(t, err)
	require.Len(t, result.Created, 1)
	require.Empty(t, result.Skipped)
	require.Equal(t, "ULP Taman", result.Created[0].UlpUnit)
	var saved entities.Permohonan
	require.NoError(t, db.First(&saved).Error)
	require.Equal(t, actor.Unit, saved.UlpUnit)
	require.Equal(t, "draft", saved.Status)
	require.Equal(t, "2026-11-11", saved.TanggalPembayaran.Format("2006-01-02"))

	_, err = s.ImportEIS(context.Background(), actor.ID.String(), ".xlsx", bytes.NewReader(eisXLSX(t, "ULP Lain")))
	require.ErrorContains(t, err, "row 2: NAMAUP is outside caller ULP")
	var count int64
	require.NoError(t, db.Model(&entities.Permohonan{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestEISXLSXImportFallsBackToHPPhone(t *testing.T) {
	s, db, actor := eisImportService(t)
	workbook := func(phone, hp string) []byte {
		book := excelize.NewFile()
		defer book.Close()
		header := []interface{}{"NAMA", "NOTELP_PEMOHON", "NOTELP_HP_PEMOHON", "TGLBAYAR", "TOTALBIAYA", "JENIS_TRANSAKSI", "NAMAUP", "ALAMAT", "TARIF_LAMA", "DAYA_LAMA", "TARIF", "DAYA", "PAKET_SAR"}
		data := []interface{}{"Synthetic Customer", phone, hp, 46337.5, 100000, "PASANG BARU", "ULP Taman", "Synthetic Address", "", "", "R1", 2200, "Prabayar (Prepaid) Satu Arah"}
		require.NoError(t, book.SetSheetRow("Sheet1", "A1", &header))
		require.NoError(t, book.SetSheetRow("Sheet1", "A2", &data))
		var out bytes.Buffer
		_, err := book.WriteTo(&out)
		require.NoError(t, err)
		return out.Bytes()
	}

	_, err := s.ImportEIS(context.Background(), actor.ID.String(), ".xlsx", bytes.NewReader(workbook("", "")))
	require.EqualError(t, err, "row 2: NOTELP_PEMOHON or NOTELP_HP_PEMOHON is required")

	result, err := s.ImportEIS(context.Background(), actor.ID.String(), ".xlsx", bytes.NewReader(workbook("", "081111111111")))
	require.NoError(t, err)
	require.Len(t, result.Created, 1)
	var saved entities.Permohonan
	require.NoError(t, db.First(&saved).Error)
	require.Equal(t, "081111111111", saved.PelangganNoHp)
	require.NotNil(t, saved.JenisPembayaran)
	require.Equal(t, "prabayar", *saved.JenisPembayaran)
}

// eisAgendaXLSX writes rows that share every mapped customer, payment, tariff
// and power value and differ only in NOAGENDA, like one customer applying for
// several connections on the same day.
func eisAgendaXLSX(t *testing.T, withAgenda bool, agendas ...string) []byte {
	t.Helper()
	book := excelize.NewFile()
	defer book.Close()
	header := []interface{}{"NAMA", "NOTELP_PEMOHON", "TGLBAYAR", "TOTALBIAYA", "JENIS_TRANSAKSI", "NAMAUP", "ALAMAT", "TARIF_LAMA", "DAYA_LAMA", "TARIF", "DAYA", "PAKET_SAR"}
	if withAgenda {
		header = append(header, "NOAGENDA")
	}
	require.NoError(t, book.SetSheetRow("Sheet1", "A1", &header))
	for index, agenda := range agendas {
		data := []interface{}{"Synthetic Developer", "080000000000", 46337.5, 100000, "PASANG BARU", "ULP Taman", "Synthetic Address", "", "", "R1MT", 900, "Prabayar (Prepaid) Satu Arah"}
		if withAgenda {
			data = append(data, agenda)
		}
		cell, err := excelize.CoordinatesToCellName(1, index+2)
		require.NoError(t, err)
		require.NoError(t, book.SetSheetRow("Sheet1", cell, &data))
	}
	var out bytes.Buffer
	_, err := book.WriteTo(&out)
	require.NoError(t, err)
	return out.Bytes()
}

func TestEISImportUsesNOAGENDAForDuplicates(t *testing.T) {
	s, db, actor := eisImportService(t)
	ctx := context.Background()

	// Identical details with different NOAGENDA are separate applications.
	first, err := s.ImportEIS(ctx, actor.ID.String(), ".xlsx", bytes.NewReader(eisAgendaXLSX(t, true, "A-001", "A-002")))
	require.NoError(t, err)
	require.Len(t, first.Created, 2)
	require.Empty(t, first.Skipped)
	var agendas []string
	require.NoError(t, db.Model(&entities.Permohonan{}).Order("eis_no_agenda").Pluck("eis_no_agenda", &agendas).Error)
	require.Equal(t, []string{"A-001", "A-002"}, agendas)

	// Re-importing skips known NOAGENDA rows, reports them, and imports the new one.
	second, err := s.ImportEIS(ctx, actor.ID.String(), ".xlsx", bytes.NewReader(eisAgendaXLSX(t, true, "A-001", "A-003", "A-002")))
	require.NoError(t, err)
	require.Len(t, second.Created, 1)
	require.Equal(t, []dto.EISImportSkippedRow{
		{Row: 2, NoAgenda: "A-001", ExistingNoPermohonan: first.Created[0].NoPermohonan, Reason: "NOAGENDA sudah pernah di-import"},
		{Row: 4, NoAgenda: "A-002", ExistingNoPermohonan: first.Created[1].NoPermohonan, Reason: "NOAGENDA sudah pernah di-import"},
	}, second.Skipped)

	// Everything already imported is a successful no-op, not an error.
	third, err := s.ImportEIS(ctx, actor.ID.String(), ".xlsx", bytes.NewReader(eisAgendaXLSX(t, true, "A-003")))
	require.NoError(t, err)
	require.Empty(t, third.Created)
	require.Len(t, third.Skipped, 1)

	var count int64
	require.NoError(t, db.Model(&entities.Permohonan{}).Count(&count).Error)
	require.Equal(t, int64(3), count)
}

func TestEISImportWithoutNOAGENDAFallsBackToDetails(t *testing.T) {
	s, db, actor := eisImportService(t)
	ctx := context.Background()

	legacy, err := s.ImportEIS(ctx, actor.ID.String(), ".xlsx", bytes.NewReader(eisAgendaXLSX(t, false, "")))
	require.NoError(t, err)
	require.Len(t, legacy.Created, 1)
	var saved entities.Permohonan
	require.NoError(t, db.First(&saved).Error)
	require.Nil(t, saved.EISNoAgenda)

	// A file without NOAGENDA matches on details.
	again, err := s.ImportEIS(ctx, actor.ID.String(), ".xlsx", bytes.NewReader(eisAgendaXLSX(t, false, "")))
	require.NoError(t, err)
	require.Empty(t, again.Created)
	require.Equal(t, []dto.EISImportSkippedRow{{Row: 2, ExistingNoPermohonan: legacy.Created[0].NoPermohonan, Reason: "data sama dengan permohonan yang sudah ada"}}, again.Skipped)

	// A draft imported before NOAGENDA was stored still blocks the same details
	// once, so legacy imports are not duplicated by a newer file.
	withAgenda, err := s.ImportEIS(ctx, actor.ID.String(), ".xlsx", bytes.NewReader(eisAgendaXLSX(t, true, "A-100")))
	require.NoError(t, err)
	require.Empty(t, withAgenda.Created)
	require.Equal(t, legacy.Created[0].NoPermohonan, withAgenda.Skipped[0].ExistingNoPermohonan)
}

func TestEISXLSXRejectsMoreThan5000DataRows(t *testing.T) {
	book := excelize.NewFile()
	defer book.Close()
	require.NoError(t, book.SetCellValue("Sheet1", "A1", "NAMA"))
	require.NoError(t, book.SetCellValue("Sheet1", "A5002", "extra"))
	var out bytes.Buffer
	_, err := book.WriteTo(&out)
	require.NoError(t, err)
	_, err = readEISRows(bytes.NewReader(out.Bytes()), ".xlsx")
	require.ErrorContains(t, err, "exceeds 5000 rows")
}

func TestEISXLSXRejectsInvalidWorkbook(t *testing.T) {
	_, err := readEISRows(strings.NewReader("not a workbook"), ".xlsx")
	require.ErrorContains(t, err, "invalid EIS XLSX workbook")
}
