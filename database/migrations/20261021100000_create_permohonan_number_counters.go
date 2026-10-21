package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261021100000_create_permohonan_number_counters", UpCreatePermohonanNumberCounters, DownCreatePermohonanNumberCounters)
}

func UpCreatePermohonanNumberCounters(db *gorm.DB) error {
	if err := db.AutoMigrate(&entities.PermohonanNumberCounter{}); err != nil {
		return err
	}

	// Backfill from the largest valid number already persisted for each year.
	// Existing request numbers are never rewritten, and gaps are intentionally
	// preserved rather than reused.
	return db.Exec(`
		INSERT INTO permohonan_number_counters (request_year, last_number)
		SELECT
			CAST(substring(no_permohonan FROM '^PBPD-([0-9]{4})-[0-9]+$') AS integer),
			MAX(CAST(substring(no_permohonan FROM '^PBPD-[0-9]{4}-([0-9]+)$') AS integer))
		FROM permohonan
		WHERE no_permohonan ~ '^PBPD-[0-9]{4}-[0-9]+$'
		GROUP BY substring(no_permohonan FROM '^PBPD-([0-9]{4})-[0-9]+$')
		ON CONFLICT (request_year) DO UPDATE
		SET last_number = GREATEST(permohonan_number_counters.last_number, EXCLUDED.last_number)
	`).Error
}

func DownCreatePermohonanNumberCounters(db *gorm.DB) error {
	return db.Migrator().DropTable(&entities.PermohonanNumberCounter{})
}
