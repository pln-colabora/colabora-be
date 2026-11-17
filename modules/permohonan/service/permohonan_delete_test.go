package service

import (
	"context"
	"mime/multipart"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type deletedKeysRecorder struct{ keys []string }

func (*deletedKeysRecorder) UploadForWorkflow(context.Context, *gorm.DB, string, string, string, []*multipart.FileHeader) ([]string, error) {
	return nil, nil
}
func (r *deletedKeysRecorder) DeleteStoredObject(_ context.Context, key string) error {
	r.keys = append(r.keys, key)
	return nil
}

func TestDeletePermohonanOnlySuperUser(t *testing.T) {
	db := phase4IntegrationDB(t)
	require.NoError(t, db.AutoMigrate(&entities.ActivityRevision{}))
	require.NoError(t, db.Exec(`CREATE TABLE pdkb_schedules (id TEXT PRIMARY KEY, schedule_date DATE NOT NULL UNIQUE, title TEXT, permohonan_id TEXT UNIQUE, created_by TEXT, updated_by TEXT, created_at DATETIME, updated_at DATETIME)`).Error)
	p := closingReady(t, rbac.JenisSambunganJTR)
	require.NoError(t, db.Omit("WorkflowNodes").Create(&p).Error)
	actor := entities.User{ID: uuid.New(), Role: rbac.RoleAdmin}
	require.NoError(t, db.Create(&entities.Document{ID: uuid.New(), UploadedBy: actor.ID, Source: "uploaded", Type: "evidence", PermohonanID: &p.ID, FilePath: "synthetic/a.pdf"}).Error)
	require.NoError(t, db.Create(&entities.ActivityRevision{ID: uuid.New(), PermohonanID: p.ID, WorkflowNode: "survei", Revision: 1, Payload: "{}", DocumentIDs: "[]", RecordedBy: actor.ID}).Error)
	booking := entities.PDKBSchedule{ID: uuid.New(), ScheduleDate: time.Now().AddDate(0, 0, 3), Title: "WO", PermohonanID: &p.ID, CreatedBy: actor.ID, UpdatedBy: actor.ID}
	require.NoError(t, db.Create(&booking).Error)

	users := &phase3UserRepository{user: actor}
	files := &deletedKeysRecorder{}
	s := &permohonanService{db: db, userRepository: users, workflowDocumentUploader: files}
	ctx := context.Background()

	for _, role := range []string{rbac.RoleAdmin, rbac.RolePelayananPelanggan, rbac.RoleTeknik} {
		users.user.Role = role
		require.ErrorIs(t, s.DeletePermohonan(ctx, p.ID.String(), actor.ID.String()), rbac.ErrWorkflowForbidden, role)
	}
	var count int64
	require.NoError(t, db.Model(&entities.Permohonan{}).Where("id = ?", p.ID).Count(&count).Error)
	require.EqualValues(t, 1, count)

	users.user.Role = rbac.RoleSuperUser
	require.NoError(t, s.DeletePermohonan(ctx, p.ID.String(), actor.ID.String()))
	require.NoError(t, db.Model(&entities.Permohonan{}).Where("id = ?", p.ID).Count(&count).Error)
	require.Zero(t, count)
	require.NoError(t, db.Model(&entities.ActivityRevision{}).Where("permohonan_id = ?", p.ID).Count(&count).Error)
	require.Zero(t, count)
	var released entities.PDKBSchedule
	require.NoError(t, db.First(&released, "id = ?", booking.ID).Error)
	require.Nil(t, released.PermohonanID)
	require.Equal(t, []string{"synthetic/a.pdf"}, files.keys)
	require.ErrorIs(t, s.DeletePermohonan(ctx, p.ID.String(), actor.ID.String()), gorm.ErrRecordNotFound)
}
