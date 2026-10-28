package migrations

import (
	"fmt"
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/workflow"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261024100000_workflow_updates", upWorkflowUpdates, downWorkflowUpdates)
}

func upWorkflowUpdates(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&entities.Permohonan{}, &entities.PermohonanActivity{}, &entities.DocumentEvidence{}, &entities.ActivityRevision{}, &entities.PDKBSchedule{}); err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE permohonan p SET legacy_priority_exempt = EXISTS
			(SELECT 1 FROM permohonan_activities a WHERE a.permohonan_id=p.id AND a.workflow_node='nps_delegation' AND a.status='completed')`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO permohonan_activities
			(id,permohonan_id,workflow_node,stage_number,status,payload,revision,sla_deadline,created_at,updated_at)
			SELECT uuid_generate_v4(),p.id,'verifikasi_prioritas',3,
			CASE WHEN p.legacy_priority_exempt THEN 'skipped' ELSE 'locked' END,
			CASE WHEN p.legacy_priority_exempt THEN '{"migration_reason":"nps_already_completed"}'::jsonb ELSE '{}'::jsonb END,
			1,(SELECT a.sla_deadline FROM permohonan_activities a WHERE a.permohonan_id=p.id AND a.workflow_node='rab_kko_kkf'),NOW(),NOW()
			FROM permohonan p ON CONFLICT (permohonan_id,workflow_node) DO NOTHING`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE permohonan_activities SET status='skipped' WHERE workflow_node='wo_app' AND status<>'completed'`).Error; err != nil {
			return err
		}
		var requests []entities.Permohonan
		if err := tx.Preload("WorkflowNodes").Find(&requests).Error; err != nil {
			return err
		}
		for _, p := range requests {
			result, err := workflow.Evaluate(p.WorkflowSnapshot())
			if err != nil {
				return err
			}
			if err := tx.Model(&p).Updates(map[string]any{"current_stage": result.CurrentStage, "status": string(result.Status)}).Error; err != nil {
				return err
			}
			for code, status := range result.Nodes {
				if err := tx.Model(&entities.PermohonanActivity{}).Where("permohonan_id=? AND workflow_node=?", p.ID, string(code)).Update("status", string(status)).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func downWorkflowUpdates(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&entities.Permohonan{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("workflow rollback requires an empty request database; preserve revisions and bookings before restoring a backup")
		}
		if err := tx.Where("workflow_node=?", string(workflow.Prioritas)).Delete(&entities.PermohonanActivity{}).Error; err != nil {
			return err
		}
		if err := tx.Migrator().DropTable(&entities.ActivityRevision{}, &entities.PDKBSchedule{}); err != nil {
			return err
		}
		for _, column := range []string{"legacy_priority_exempt", "energize_status"} {
			if err := tx.Migrator().DropColumn(&entities.Permohonan{}, column); err != nil {
				return err
			}
		}
		if err := tx.Migrator().DropColumn(&entities.PermohonanActivity{}, "revision"); err != nil {
			return err
		}
		return tx.Migrator().DropColumn(&entities.DocumentEvidence{}, "is_current")
	})
}
