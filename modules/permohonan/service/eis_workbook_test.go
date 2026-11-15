package service

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
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

func TestEISXLSXImportCanonicalizesMatchingULP(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE users (id TEXT PRIMARY KEY, name TEXT, email TEXT, role TEXT, unit TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE permohonan_number_counters (request_year INTEGER PRIMARY KEY, last_number INTEGER NOT NULL)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE permohonan (
		id TEXT PRIMARY KEY, no_permohonan TEXT, jenis_permohonan TEXT, jenis_sambungan TEXT,
		jenis_pembayaran TEXT, tarif TEXT, daya_lama INTEGER, daya_baru INTEGER,
		golongan_tarif TEXT, golongan_tarif_lama TEXT, pembatas_baru TEXT, pembatas_lama TEXT,
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

	created, err := s.ImportEIS(context.Background(), actor.ID.String(), ".xlsx", bytes.NewReader(eisXLSX(t, "ULP TAMAN")))
	require.NoError(t, err)
	require.Len(t, created, 1)
	require.Equal(t, "ULP Taman", created[0].UlpUnit)
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
