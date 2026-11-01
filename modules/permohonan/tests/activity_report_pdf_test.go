package tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/pdftext"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/stretchr/testify/require"
)

func (h tariffHarness) exportActivity(t *testing.T, id, node string, user entities.User) []byte {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/permohonan/"+id+"/activities/"+node+"/export", nil)
	request.Header.Set("X-Test-User", user.ID.String())
	recorder := httptest.NewRecorder()
	h.engine.ServeHTTP(recorder, request)
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	require.Equal(t, "application/pdf", recorder.Header().Get("Content-Type"))
	content := recorder.Body.Bytes()
	require.Equal(t, "%PDF", string(content[:4]))
	// COLABORA_PDF_SAMPLE_DIR keeps generated samples for manual review.
	if dir := os.Getenv("COLABORA_PDF_SAMPLE_DIR"); dir != "" {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "laporan_"+node+".pdf"), content, 0o644))
	}
	return content
}

func requireActivityReport(t *testing.T, content []byte, want ...string) {
	t.Helper()
	for _, text := range append([]string{
		"LAPORAN AKTIVITAS", "UP3 SBY BARAT", "Nomor Permohonan", "Pelanggan Uji",
		"Dokumen ini dihasilkan otomatis oleh aplikasi Colabora pada ", "Tidak memerlukan tanda tangan.", "Halaman 1 dari 1",
	}, want...) {
		require.True(t, pdftext.Contains(content, text), "PDF should contain %q", text)
	}
	for _, text := range []string{"PEJABAT BERWENANG", "Paraf", "document_ids", "Jawaban Activity"} {
		require.False(t, pdftext.Contains(content, text), "PDF must not contain %q", text)
	}
}

func TestExportActivityUsesReportLayout(t *testing.T) {
	h := newTariffHarness(t)
	code, result := h.do(t, http.MethodPost, "/api/permohonan", h.frontage, createBody(rbac.JenisPermohonanPasangBaru, map[string]any{
		"tarif": rbac.TarifBisnis, "golongan_tarif": "B3", "daya_baru": 210000, "pembatas_baru": "6",
	}))
	require.Equal(t, http.StatusCreated, code, result.Error)
	created := decodeObject(t, result.Data)
	id, number := created["id"].(string), created["no_permohonan"].(string)

	requireActivityReport(t, h.exportActivity(t, id, "permohonan", h.frontage),
		number, "Permohonan PB/PD", "Bisnis / B3 / 210.000 VA", "Selesai", "Petugas Pelayanan (Pelayanan Pelanggan)")

	advanceToPriority(t, h.db, id)
	code, result = h.do(t, http.MethodPost, "/api/permohonan/"+id+"/verifikasi-prioritas", h.planner,
		map[string]any{"priority": "High", "notes": "Pelanggan prioritas"})
	require.Equal(t, http.StatusOK, code, result.Error)
	requireActivityReport(t, h.exportActivity(t, id, "verifikasi_prioritas", h.planner),
		number, "Verifikasi Prioritas Pekerjaan", "Data Aktivitas", "Prioritas", "High", "Catatan", "Pelanggan prioritas",
		"3 - Perencanaan Perluasan", "Petugas Perencanaan (Perencanaan)", "Tidak ada lampiran.")
}
