package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/query"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestAccessAndListShareAssignmentBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	for _, sql := range []string{
		"CREATE TABLE users (id TEXT PRIMARY KEY, role TEXT, unit TEXT)",
		"CREATE TABLE permohonan (id TEXT PRIMARY KEY, ulp_unit TEXT)",
		"CREATE TABLE vendor_assignments (permohonan_id TEXT, vendor_role TEXT, vendor_id TEXT)",
	} {
		require.NoError(t, db.Exec(sql).Error)
	}
	own, other := uuid.NewString(), uuid.NewString()
	require.NoError(t, db.Exec("INSERT INTO permohonan VALUES (?, ?), (?, ?)", own, "ULP A", other, "ULP B").Error)
	for _, tc := range []struct {
		role, unit string
		assigned   bool
		count      int64
	}{
		{rbac.RoleVendorTiang, "vendor", true, 1},
		{rbac.RoleVendorTiang, "vendor", false, 0},
		{rbac.RoleVendorKonstruksi, "vendor", true, 1},
		{rbac.RoleVendorSrApp, "vendor", true, 1},
		{rbac.RoleTeknik, "ULP A", false, 1},
		{rbac.RolePelayananPelanggan, "ULP A", false, 1},
		{rbac.RoleTeknik, "", false, 0},
		{rbac.RoleNps, "UP3", false, 2},
		{rbac.RolePerencanaan, "UP3", false, 2},
		{rbac.RoleKonstruksi, "UP3", false, 2},
		{rbac.RoleTransaksiEnergi, "UP3", false, 2},
		{rbac.RoleJaringan, "UP3", false, 2},
		{rbac.RolePdkb, "UP3", false, 2},
		{rbac.RoleSuperUser, "", false, 2},
		{"unknown", "ULP A", false, 0},
	} {
		t.Run(tc.role+tc.unit+uuid.NewString(), func(t *testing.T) {
			actorID := uuid.NewString()
			require.NoError(t, db.Exec("INSERT INTO users VALUES (?, ?, ?)", actorID, tc.role, tc.unit).Error)
			if tc.assigned {
				require.NoError(t, db.Exec("INSERT INTO vendor_assignments VALUES (?, ?, ?)", own, tc.role, actorID).Error)
			}
			// ApplyFilters runs before counts and pagination, including scope=all.
			filter := &query.PermohonanFilter{Scope: "all", CurrentRole: tc.role, CurrentUnit: tc.unit, CurrentUserID: actorID}
			var count int64
			require.NoError(t, filter.ApplyFilters(db.Model(&entities.Permohonan{})).Count(&count).Error)
			require.Equal(t, tc.count, count)
			for _, suffix := range []string{"", "/activities", "/logs", "/documents", "/documents/doc-id", "/pelaksanaan-konstruksi"} {
				router := gin.New()
				router.Use(func(c *gin.Context) { c.Set("user_id", actorID) })
				router.Use(RequirePermohonanAccess(db))
				called := false
				method := http.MethodGet
				if suffix == "/pelaksanaan-konstruksi" {
					method = http.MethodPost
				}
				router.Handle(method, "/api/permohonan/:id"+suffix, func(c *gin.Context) { called = true; c.Status(204) })
				for _, id := range []string{own, other, uuid.NewString()} {
					called = false
					recorder := httptest.NewRecorder()
					router.ServeHTTP(recorder, httptest.NewRequest(method, "/api/permohonan/"+id+suffix, nil))
					allowed := (id == own && tc.count > 0) || (id == other && tc.count == 2)
					if allowed {
						require.Equal(t, 204, recorder.Code)
						require.Equal(t, "no-store", recorder.Header().Get("Cache-Control"))
					} else {
						require.Equal(t, 404, recorder.Code)
						require.NotContains(t, recorder.Body.String(), "vendor_assignments")
					}
					require.Equal(t, allowed, called, "denied requests must not reach data or presign handlers")
				}
			}
		})
	}
}
