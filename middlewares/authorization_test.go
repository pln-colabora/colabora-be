package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	permohonanQuery "github.com/pln-colabora/colabora-be/modules/permohonan/query"
	permohonanRepo "github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	userRepo "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/constants"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/samber/do"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

type fakeUserRepository struct {
	user entities.User
	err  error
}

func (f *fakeUserRepository) Register(ctx context.Context, tx *gorm.DB, user entities.User) (entities.User, error) {
	return entities.User{}, nil
}
func (f *fakeUserRepository) GetUserById(ctx context.Context, tx *gorm.DB, userId string) (entities.User, error) {
	return f.user, f.err
}
func (f *fakeUserRepository) GetUserByEmail(ctx context.Context, tx *gorm.DB, email string) (entities.User, error) {
	return entities.User{}, nil
}
func (f *fakeUserRepository) CheckEmail(ctx context.Context, tx *gorm.DB, email string) (entities.User, bool, error) {
	return entities.User{}, false, nil
}
func (f *fakeUserRepository) Update(ctx context.Context, tx *gorm.DB, user entities.User) (entities.User, error) {
	return entities.User{}, nil
}
func (f *fakeUserRepository) Delete(ctx context.Context, tx *gorm.DB, userId string) error {
	return nil
}
func (f *fakeUserRepository) ExistsByUnitAndRoles(ctx context.Context, tx *gorm.DB, unit string, roles []string) (bool, error) {
	return true, nil
}

type fakePermohonanRepository struct {
	permohonan entities.Permohonan
	err        error
}

func (f *fakePermohonanRepository) Create(ctx context.Context, tx *gorm.DB, permohonan entities.Permohonan, activities []entities.PermohonanActivity, log entities.ActivityLog) (entities.Permohonan, error) {
	return entities.Permohonan{}, nil
}
func (f *fakePermohonanRepository) GetById(ctx context.Context, tx *gorm.DB, id string) (entities.Permohonan, error) {
	return f.permohonan, f.err
}
func (f *fakePermohonanRepository) List(ctx context.Context, tx *gorm.DB, filter *permohonanQuery.PermohonanFilter) ([]permohonanQuery.Permohonan, int64, error) {
	return nil, 0, nil
}
func (f *fakePermohonanRepository) CountByNoPermohonanPrefix(ctx context.Context, tx *gorm.DB, prefix string) (int64, error) {
	return 0, nil
}
func (f *fakePermohonanRepository) ListWorkflowNodes(ctx context.Context, tx *gorm.DB, permohonanID string) ([]entities.PermohonanActivity, error) {
	return nil, nil
}
func (f *fakePermohonanRepository) ListActivityLogs(ctx context.Context, tx *gorm.DB, permohonanID string) ([]entities.ActivityLog, error) {
	return nil, nil
}

func newTestInjector(user entities.User, userErr error, permohonan entities.Permohonan, permohonanErr error) *do.Injector {
	injector := do.New()

	do.ProvideNamed(injector, constants.DB, func(i *do.Injector) (*gorm.DB, error) {
		return &gorm.DB{}, nil
	})
	do.Provide(injector, func(i *do.Injector) (userRepo.UserRepository, error) {
		return &fakeUserRepository{user: user, err: userErr}, nil
	})
	do.Provide(injector, func(i *do.Injector) (permohonanRepo.PermohonanRepository, error) {
		return &fakePermohonanRepository{permohonan: permohonan, err: permohonanErr}, nil
	})

	return injector
}

func TestRequireActivityOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)

	permohonan := entities.Permohonan{
		ID:             uuid.New(),
		JenisSambungan: rbac.JenisSambunganJTR,
		UlpUnit:        "ULP Taman",
		CurrentStage:   2,
	}

	t.Run("owning role reaches the handler", func(t *testing.T) {
		user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Taman"}
		injector := newTestInjector(user, nil, permohonan, nil)

		router := gin.New()
		router.GET("/permohonan/:id", func(ctx *gin.Context) {
			ctx.Set("user_id", user.ID.String())
			ctx.Next()
		}, RequireActivityOwner(2, injector), func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/permohonan/"+permohonan.ID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("non-owning role is rejected with 403", func(t *testing.T) {
		user := entities.User{ID: uuid.New(), Role: rbac.RolePelayananPelanggan, Unit: "ULP Taman"}
		injector := newTestInjector(user, nil, permohonan, nil)

		router := gin.New()
		router.GET("/permohonan/:id", func(ctx *gin.Context) {
			ctx.Set("user_id", user.ID.String())
			ctx.Next()
		}, RequireActivityOwner(2, injector), func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/permohonan/"+permohonan.ID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("permohonan lookup failure returns 404", func(t *testing.T) {
		user := entities.User{ID: uuid.New(), Role: rbac.RoleTeknik, Unit: "ULP Taman"}
		injector := newTestInjector(user, nil, entities.Permohonan{}, gorm.ErrRecordNotFound)

		router := gin.New()
		router.GET("/permohonan/:id", func(ctx *gin.Context) {
			ctx.Set("user_id", user.ID.String())
			ctx.Next()
		}, RequireActivityOwner(2, injector), func(ctx *gin.Context) {
			ctx.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/permohonan/"+uuid.NewString(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}
