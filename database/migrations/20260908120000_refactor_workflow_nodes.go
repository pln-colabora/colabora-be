package migrations

import (
	"fmt"

	"github.com/pln-colabora/colabora-be/database"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260908120000_refactor_workflow_nodes", UpRefactorWorkflowNodes, DownRefactorWorkflowNodes)
}

// UpRefactorWorkflowNodes converts the old activity-number persistence into the
// canonical workflow-node graph. It preserves numbered activity history where it
// exists, while adding decision/support nodes as locked records.
func UpRefactorWorkflowNodes(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		m := tx.Migrator()
		if m.HasColumn("permohonan", "nps_keputusan") && !m.HasColumn("permohonan", "nps_delegation_status") {
			if err := tx.Exec("ALTER TABLE permohonan RENAME COLUMN nps_keputusan TO nps_delegation_status").Error; err != nil {
				return err
			}
		}
		if m.HasColumn("permohonan", "owner_fn_override") {
			if err := tx.Exec("ALTER TABLE permohonan DROP COLUMN owner_fn_override").Error; err != nil {
				return err
			}
		}

		if !m.HasColumn("permohonan_activities", "workflow_node") {
			if err := tx.Exec("ALTER TABLE permohonan_activities ADD COLUMN workflow_node varchar(50)").Error; err != nil {
				return err
			}
			if err := tx.Exec(`UPDATE permohonan_activities SET workflow_node = CASE activity_number
				WHEN 1 THEN 'permohonan' WHEN 2 THEN 'survei' WHEN 3 THEN 'rab_kko_kkf'
				WHEN 4 THEN 'permohonan_perluasan' WHEN 5 THEN 'nps_delegation'
				WHEN 6 THEN 'wo_tiang' WHEN 7 THEN 'wo_konstruksi' WHEN 8 THEN 'wo_app'
				WHEN 9 THEN 'reservasi_material' WHEN 10 THEN 'tera_app'
				WHEN 11 THEN 'pemasangan_tiang' WHEN 12 THEN 'pelaksanaan_konstruksi'
				WHEN 13 THEN 'energize_jaringan' WHEN 14 THEN 'pemasangan_sr_app'
				WHEN 15 THEN 'entri_mutasi_pdl' WHEN 16 THEN 'arsip_ail' WHEN 17 THEN 'selesai'
			END`).Error; err != nil {
				return err
			}
		}
		var unmappedActivities int64
		if err := tx.Model(&struct{ ID string }{}).Table("permohonan_activities").Where("workflow_node IS NULL OR workflow_node = ''").Count(&unmappedActivities).Error; err != nil {
			return err
		}
		if unmappedActivities > 0 {
			return fmt.Errorf("cannot map %d legacy activity rows to canonical workflow nodes", unmappedActivities)
		}
		if err := tx.Exec("ALTER TABLE permohonan_activities ALTER COLUMN workflow_node SET NOT NULL").Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE permohonan_activities SET status = CASE status WHEN 'done' THEN 'completed' WHEN 'not_started' THEN 'locked' ELSE status END").Error; err != nil {
			return err
		}
		if err := tx.Exec("ALTER TABLE permohonan_activities ALTER COLUMN activity_number DROP NOT NULL").Error; err != nil {
			return err
		}
		if err := tx.Exec("ALTER TABLE permohonan_activities ALTER COLUMN sla_deadline DROP NOT NULL").Error; err != nil {
			return err
		}

		if !m.HasColumn("activity_logs", "workflow_node") {
			if err := tx.Exec("ALTER TABLE activity_logs ADD COLUMN workflow_node varchar(50)").Error; err != nil {
				return err
			}
			if err := tx.Exec(`UPDATE activity_logs SET workflow_node = CASE activity_number
				WHEN 1 THEN 'permohonan' WHEN 2 THEN 'survei' WHEN 3 THEN 'rab_kko_kkf'
				WHEN 4 THEN 'permohonan_perluasan' WHEN 5 THEN 'nps_delegation'
				WHEN 6 THEN 'wo_tiang' WHEN 7 THEN 'wo_konstruksi' WHEN 8 THEN 'wo_app'
				WHEN 9 THEN 'reservasi_material' WHEN 10 THEN 'tera_app'
				WHEN 11 THEN 'pemasangan_tiang' WHEN 12 THEN 'pelaksanaan_konstruksi'
				WHEN 13 THEN 'energize_jaringan' WHEN 14 THEN 'pemasangan_sr_app'
				WHEN 15 THEN 'entri_mutasi_pdl' WHEN 16 THEN 'arsip_ail' WHEN 17 THEN 'selesai'
			END`).Error; err != nil {
				return err
			}
		}

		if err := tx.Exec(`CREATE TABLE IF NOT EXISTS document_evidence (
			id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
			document_id uuid NOT NULL,
			permohonan_id uuid NOT NULL,
			workflow_node varchar(50) NOT NULL,
			attached_by uuid NOT NULL,
			created_at timestamp with time zone,
			updated_at timestamp with time zone
		)`).Error; err != nil {
			return err
		}
		if m.HasColumn("documents", "activity_number") {
			if err := tx.Exec(`INSERT INTO document_evidence (document_id, permohonan_id, workflow_node, attached_by, created_at, updated_at)
				SELECT id, permohonan_id, CASE activity_number
					WHEN 1 THEN 'permohonan' WHEN 2 THEN 'survei' WHEN 3 THEN 'rab_kko_kkf'
					WHEN 4 THEN 'permohonan_perluasan' WHEN 5 THEN 'nps_delegation'
					WHEN 6 THEN 'wo_tiang' WHEN 7 THEN 'wo_konstruksi' WHEN 8 THEN 'wo_app'
					WHEN 9 THEN 'reservasi_material' WHEN 10 THEN 'tera_app'
					WHEN 11 THEN 'pemasangan_tiang' WHEN 12 THEN 'pelaksanaan_konstruksi'
					WHEN 13 THEN 'energize_jaringan' WHEN 14 THEN 'pemasangan_sr_app'
					WHEN 15 THEN 'entri_mutasi_pdl' WHEN 16 THEN 'arsip_ail' WHEN 17 THEN 'selesai'
					END, uploaded_by, created_at, updated_at
				FROM documents WHERE permohonan_id IS NOT NULL AND activity_number IS NOT NULL
				ON CONFLICT DO NOTHING`).Error; err != nil {
				return err
			}
			if err := tx.Exec("ALTER TABLE documents DROP COLUMN activity_number").Error; err != nil {
				return err
			}
		}

		if err := tx.Exec(`INSERT INTO permohonan_activities
			(id, permohonan_id, workflow_node, activity_number, stage_number, status, sla_deadline, payload, completed_by, completed_at, created_at, updated_at)
			SELECT uuid_generate_v4(), p.id, n.workflow_node, n.activity_number, n.stage_number,
				CASE n.workflow_node WHEN 'permohonan' THEN 'completed' WHEN 'survei' THEN 'available' ELSE 'locked' END,
				CASE WHEN n.sla_activity IS NULL THEN NULL ELSE p.request_date + COALESCE(sr.offset_days, 0)::integer END,
				'{}'::jsonb,
				CASE WHEN n.workflow_node = 'permohonan' THEN p.created_by ELSE NULL END,
				CASE WHEN n.workflow_node = 'permohonan' THEN p.created_at ELSE NULL END,
				NOW(), NOW()
			FROM permohonan p
			CROSS JOIN (VALUES
				('permohonan', 1::smallint, 1::smallint, 1::smallint),
				('survei', 2::smallint, 2::smallint, 2::smallint),
				('rab_kko_kkf', 3::smallint, 3::smallint, 3::smallint),
				('kebutuhan_tiang', NULL::smallint, 3::smallint, NULL::smallint),
				('permohonan_perluasan', 4::smallint, 3::smallint, 4::smallint),
				('nps_delegation', 5::smallint, 3::smallint, NULL::smallint),
				('wo_tiang', 6::smallint, 4::smallint, 6::smallint),
				('wo_konstruksi', 7::smallint, 4::smallint, 7::smallint),
				('wo_app', 8::smallint, 4::smallint, 8::smallint),
				('reservasi_material', 9::smallint, 4::smallint, 9::smallint),
				('tera_app', 10::smallint, 4::smallint, 10::smallint),
				('wo_pdkb', NULL::smallint, 4::smallint, NULL::smallint),
				('pk_vendor', NULL::smallint, 4::smallint, NULL::smallint),
				('pemasangan_tiang', 11::smallint, 5::smallint, 11::smallint),
				('pelaksanaan_konstruksi', 12::smallint, 5::smallint, 12::smallint),
				('pdkb_documentation', NULL::smallint, 5::smallint, NULL::smallint),
				('energize_jaringan', 13::smallint, 6::smallint, 13::smallint),
				('pemasangan_sr_app', 14::smallint, 6::smallint, 14::smallint),
				('entri_mutasi_pdl', 15::smallint, 7::smallint, 15::smallint),
				('arsip_ail', 16::smallint, 7::smallint, 16::smallint),
				('selesai', 17::smallint, 7::smallint, 17::smallint)
			) AS n(workflow_node, activity_number, stage_number, sla_activity)
			LEFT JOIN sla_rules sr ON sr.jenis_sambungan = p.jenis_sambungan AND sr.activity_number = n.sla_activity
			WHERE NOT EXISTS (
				SELECT 1 FROM permohonan_activities pa
				WHERE pa.permohonan_id = p.id AND pa.workflow_node = n.workflow_node
			)`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE permohonan_activities pa
			SET sla_deadline = p.request_date + COALESCE(sr.offset_days, 0)::integer
			FROM permohonan p, sla_rules sr
			WHERE pa.permohonan_id = p.id
				AND sr.jenis_sambungan = p.jenis_sambungan
				AND sr.activity_number = pa.activity_number
				AND pa.activity_number IS NOT NULL
				AND pa.sla_deadline IS NULL`).Error; err != nil {
			return err
		}

		statements := []string{
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_permohonan_workflow_node ON permohonan_activities (permohonan_id, workflow_node)",
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_id_permohonan ON documents (id, permohonan_id)",
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_document_evidence_document_node ON document_evidence (document_id, workflow_node)",
			"CREATE INDEX IF NOT EXISTS idx_document_evidence_permohonan_node ON document_evidence (permohonan_id, workflow_node)",
		}
		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("workflow-node index: %w", err)
			}
		}
		if err := tx.Exec(`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_document_evidence_document_permohonan') THEN
				ALTER TABLE document_evidence ADD CONSTRAINT fk_document_evidence_document_permohonan
				FOREIGN KEY (document_id, permohonan_id) REFERENCES documents (id, permohonan_id) ON DELETE CASCADE;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_document_evidence_workflow_node') THEN
				ALTER TABLE document_evidence ADD CONSTRAINT fk_document_evidence_workflow_node
				FOREIGN KEY (permohonan_id, workflow_node) REFERENCES permohonan_activities (permohonan_id, workflow_node) ON DELETE CASCADE;
			END IF;
		END $$`).Error; err != nil {
			return fmt.Errorf("workflow-node foreign keys: %w", err)
		}
		return nil
	})
}

// DownRefactorWorkflowNodes intentionally discards node-only associations. A
// rollback is a development recovery operation; legacy activity rows are retained.
func DownRefactorWorkflowNodes(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		m := tx.Migrator()
		if m.HasTable("document_evidence") {
			if err := tx.Exec("DROP TABLE document_evidence").Error; err != nil {
				return err
			}
		}
		if m.HasColumn("documents", "activity_number") == false {
			if err := tx.Exec("ALTER TABLE documents ADD COLUMN activity_number smallint").Error; err != nil {
				return err
			}
		}
		if m.HasColumn("activity_logs", "workflow_node") {
			if err := tx.Exec("ALTER TABLE activity_logs DROP COLUMN workflow_node").Error; err != nil {
				return err
			}
		}
		if m.HasColumn("permohonan_activities", "workflow_node") {
			if err := tx.Exec("DROP INDEX IF EXISTS idx_permohonan_workflow_node").Error; err != nil {
				return err
			}
			if err := tx.Exec("ALTER TABLE permohonan_activities DROP COLUMN workflow_node").Error; err != nil {
				return err
			}
		}
		if !m.HasColumn("permohonan", "owner_fn_override") {
			if err := tx.Exec("ALTER TABLE permohonan ADD COLUMN owner_fn_override varchar(50) NOT NULL DEFAULT ''").Error; err != nil {
				return err
			}
		}
		if m.HasColumn("permohonan", "nps_delegation_status") && !m.HasColumn("permohonan", "nps_keputusan") {
			if err := tx.Exec("ALTER TABLE permohonan RENAME COLUMN nps_delegation_status TO nps_keputusan").Error; err != nil {
				return err
			}
		}
		return nil
	})
}
