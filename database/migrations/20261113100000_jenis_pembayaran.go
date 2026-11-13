package migrations

import (
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func init() {
	database.RegisterMigration("20261113100000_jenis_pembayaran", func(db *gorm.DB) error {
		return db.Migrator().AddColumn(&entities.Permohonan{}, "JenisPembayaran")
	}, func(db *gorm.DB) error {
		return db.Migrator().DropColumn(&entities.Permohonan{}, "JenisPembayaran")
	})
}
