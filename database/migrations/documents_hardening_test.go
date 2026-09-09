package migrations

import (
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestDocumentHardeningMigrationPostgres(t *testing.T) {
	dsn := os.Getenv("COLABORA_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set COLABORA_TEST_POSTGRES_DSN for isolated PostgreSQL migration verification")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	tx := db.Begin()
	require.NoError(t, tx.Error)
	defer tx.Rollback()
	schema := "test_documents_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, tx.Exec("CREATE SCHEMA "+schema).Error)
	require.NoError(t, tx.Exec("SET LOCAL search_path TO "+schema+", public").Error)
	require.NoError(t, tx.Exec(`CREATE TABLE documents (id uuid PRIMARY KEY, type varchar(50) NOT NULL, file_path varchar(500) NOT NULL, uploaded_by uuid NOT NULL, permohonan_id uuid, created_at timestamptz, updated_at timestamptz)`).Error)
	require.NoError(t, tx.Exec(`INSERT INTO documents(id,type,file_path,uploaded_by,created_at) VALUES (gen_random_uuid(),'evidence','documents/legacy',gen_random_uuid(),now())`).Error)
	require.NoError(t, UpHardenDocuments(tx))
	for _, column := range []string{"original_filename", "mime_type", "size_bytes", "checksum_sha256", "scan_status", "revision", "supersedes_id", "superseded_by_id"} {
		require.True(t, tx.Migrator().HasColumn("documents", column), column)
	}
	var value string
	require.NoError(t, tx.Raw(`SELECT source FROM documents LIMIT 1`).Scan(&value).Error)
	require.Equal(t, "imported", value)
	var constraints int64
	require.NoError(t, tx.Raw(`SELECT count(*) FROM pg_constraint WHERE conrelid = 'documents'::regclass AND conname IN ('documents_size_nonnegative','documents_revision_positive','documents_checksum_valid','documents_source_valid','documents_classification_valid','documents_scan_status_valid','documents_supersedes_fk','documents_superseded_by_fk')`).Scan(&constraints).Error)
	require.EqualValues(t, 8, constraints)
	require.NoError(t, DownHardenDocuments(tx))
	require.False(t, tx.Migrator().HasColumn("documents", "mime_type"))
	require.NoError(t, UpHardenDocuments(tx))
}
