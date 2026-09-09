package service

import (
	"context"
	"github.com/google/uuid"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/modules/permohonan/repository"
	userRepository "github.com/pln-colabora/colabora-be/modules/user/repository"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestVendorAssignmentAuthorizationAndAtomicAudit(t *testing.T) {
	for _, scenario := range []string{"tiang", "konstruksi", "sr-app", "wrong assigner", "wrong vendor role", "PLG SRAPP", "before NPS", "audit failure"} {
		t.Run(scenario, func(t *testing.T) {
			db := phase4IntegrationDB(t)
			require.NoError(t, db.Exec("CREATE TABLE vendor_assignments (permohonan_id TEXT, vendor_role TEXT, vendor_id TEXT, assigned_by TEXT, created_at DATETIME, updated_at DATETIME, PRIMARY KEY(permohonan_id, vendor_role))").Error)
			p := delegatedRequest(t, rbac.JenisSambunganJTR, true)
			actor := entities.User{ID: uuid.New(), Role: rbac.RolePerencanaan, Unit: "UP3"}
			vendor := entities.User{ID: uuid.New(), Role: rbac.RoleVendorTiang, Unit: "vendor"}
			switch scenario {
			case "konstruksi":
				actor.Role = rbac.RoleKonstruksi
				vendor.Role = rbac.RoleVendorKonstruksi
			case "sr-app":
				actor.Role = rbac.RoleTransaksiEnergi
				vendor.Role = rbac.RoleVendorSrApp
			case "wrong assigner":
				actor.Role = rbac.RoleSuperUser
			case "PLG SRAPP":
				p.JenisSambungan = rbac.JenisSambunganPlgTmLebih5
				actor.Role = rbac.RoleTransaksiEnergi
				vendor.Role = rbac.RoleVendorSrApp
			case "before NPS":
				p = phase4Request(t, rbac.JenisSambunganJTR)
			}
			req := dto.VendorAssignmentRequest{VendorID: vendor.ID.String(), VendorRole: vendor.Role}
			if scenario == "wrong vendor role" {
				vendor.Role = rbac.RoleNps
			}
			require.NoError(t, db.Create(&actor).Error)
			require.NoError(t, db.Create(&vendor).Error)
			require.NoError(t, db.Omit("WorkflowNodes").Create(&p).Error)
			for i := range p.WorkflowNodes {
				p.WorkflowNodes[i].PermohonanID = p.ID
			}
			require.NoError(t, db.Create(&p.WorkflowNodes).Error)
			if scenario == "audit failure" {
				require.NoError(t, db.Exec("CREATE TRIGGER fail_assign_audit BEFORE INSERT ON activity_logs BEGIN SELECT RAISE(ABORT, 'synthetic failure'); END").Error)
			}
			s := &permohonanService{db: db, permohonanRepository: repository.NewPermohonanRepository(db), userRepository: userRepository.NewUserRepository(db), vendorAssignmentRepository: repository.NewVendorAssignmentRepository()}
			_, err := s.AssignVendor(context.Background(), p.ID.String(), actor.ID.String(), req)
			success := scenario == "tiang" || scenario == "konstruksi" || scenario == "sr-app"
			if success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			var count, logs int64
			require.NoError(t, db.Table("vendor_assignments").Count(&count).Error)
			require.NoError(t, db.Table("activity_logs").Count(&logs).Error)
			if success {
				require.EqualValues(t, 1, count)
				require.EqualValues(t, 1, logs)
				_, err = s.AssignVendor(context.Background(), p.ID.String(), actor.ID.String(), req)
				require.ErrorIs(t, err, workflow.ErrNotActionable)
			} else {
				require.Zero(t, count)
				require.Zero(t, logs)
			}
		})
	}
}
