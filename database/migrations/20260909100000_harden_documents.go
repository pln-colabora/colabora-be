package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20260909100000_harden_documents", UpHardenDocuments, DownHardenDocuments)
}

func UpHardenDocuments(db *gorm.DB) error {
	statements := []string{
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS original_filename varchar(255)`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS mime_type varchar(100)`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS size_bytes bigint`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS checksum_sha256 char(64)`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS source varchar(20)`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS classification varchar(20)`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS scan_status varchar(20)`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS scan_checked_at timestamp with time zone`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS revision smallint`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS supersedes_id uuid`,
		`ALTER TABLE documents ADD COLUMN IF NOT EXISTS superseded_by_id uuid`,
		`UPDATE documents SET original_filename = COALESCE(original_filename, 'legacy-upload'), mime_type = COALESCE(mime_type, 'application/octet-stream'), size_bytes = COALESCE(size_bytes, 0), checksum_sha256 = COALESCE(checksum_sha256, repeat('0', 64)), source = COALESCE(source, 'imported'), classification = COALESCE(classification, 'restricted'), scan_status = COALESCE(scan_status, 'not_scanned'), revision = COALESCE(revision, 1)`,
		`ALTER TABLE documents ALTER COLUMN original_filename SET NOT NULL, ALTER COLUMN mime_type SET NOT NULL, ALTER COLUMN size_bytes SET NOT NULL, ALTER COLUMN checksum_sha256 SET NOT NULL, ALTER COLUMN source SET NOT NULL, ALTER COLUMN source SET DEFAULT 'uploaded', ALTER COLUMN classification SET NOT NULL, ALTER COLUMN classification SET DEFAULT 'restricted', ALTER COLUMN scan_status SET NOT NULL, ALTER COLUMN scan_status SET DEFAULT 'not_scanned', ALTER COLUMN revision SET NOT NULL, ALTER COLUMN revision SET DEFAULT 1`,
		`CREATE INDEX IF NOT EXISTS idx_documents_supersedes_id ON documents(supersedes_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_documents_superseded_by_id ON documents(superseded_by_id) WHERE superseded_by_id IS NOT NULL`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'documents_size_nonnegative' AND conrelid = 'documents'::regclass) THEN ALTER TABLE documents ADD CONSTRAINT documents_size_nonnegative CHECK (size_bytes >= 0); END IF; END $$`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'documents_revision_positive' AND conrelid = 'documents'::regclass) THEN ALTER TABLE documents ADD CONSTRAINT documents_revision_positive CHECK (revision > 0); END IF; END $$`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'documents_checksum_valid' AND conrelid = 'documents'::regclass) THEN ALTER TABLE documents ADD CONSTRAINT documents_checksum_valid CHECK (checksum_sha256 ~ '^[0-9a-f]{64}$'); END IF; END $$`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'documents_source_valid' AND conrelid = 'documents'::regclass) THEN ALTER TABLE documents ADD CONSTRAINT documents_source_valid CHECK (source IN ('uploaded','generated','imported','scanned')); END IF; END $$`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'documents_classification_valid' AND conrelid = 'documents'::regclass) THEN ALTER TABLE documents ADD CONSTRAINT documents_classification_valid CHECK (classification IN ('internal','restricted','public')); END IF; END $$`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'documents_scan_status_valid' AND conrelid = 'documents'::regclass) THEN ALTER TABLE documents ADD CONSTRAINT documents_scan_status_valid CHECK (scan_status IN ('not_scanned','clean','infected')); END IF; END $$`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'documents_supersedes_fk' AND conrelid = 'documents'::regclass) THEN ALTER TABLE documents ADD CONSTRAINT documents_supersedes_fk FOREIGN KEY (supersedes_id) REFERENCES documents(id) ON DELETE SET NULL; END IF; END $$`,
		`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'documents_superseded_by_fk' AND conrelid = 'documents'::regclass) THEN ALTER TABLE documents ADD CONSTRAINT documents_superseded_by_fk FOREIGN KEY (superseded_by_id) REFERENCES documents(id) ON DELETE SET NULL; END IF; END $$`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func DownHardenDocuments(db *gorm.DB) error {
	statements := []string{
		`ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_superseded_by_fk, DROP CONSTRAINT IF EXISTS documents_supersedes_fk, DROP CONSTRAINT IF EXISTS documents_scan_status_valid, DROP CONSTRAINT IF EXISTS documents_classification_valid, DROP CONSTRAINT IF EXISTS documents_source_valid, DROP CONSTRAINT IF EXISTS documents_checksum_valid, DROP CONSTRAINT IF EXISTS documents_revision_positive, DROP CONSTRAINT IF EXISTS documents_size_nonnegative`,
		`DROP INDEX IF EXISTS idx_documents_superseded_by_id`,
		`DROP INDEX IF EXISTS idx_documents_supersedes_id`,
		`ALTER TABLE documents DROP COLUMN IF EXISTS superseded_by_id, DROP COLUMN IF EXISTS supersedes_id, DROP COLUMN IF EXISTS revision, DROP COLUMN IF EXISTS scan_checked_at, DROP COLUMN IF EXISTS scan_status, DROP COLUMN IF EXISTS classification, DROP COLUMN IF EXISTS source, DROP COLUMN IF EXISTS checksum_sha256, DROP COLUMN IF EXISTS size_bytes, DROP COLUMN IF EXISTS mime_type, DROP COLUMN IF EXISTS original_filename`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
