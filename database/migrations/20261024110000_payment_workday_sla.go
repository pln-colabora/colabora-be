package migrations

import (
	"fmt"
	"github.com/pln-colabora/colabora-be/database"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/businessday"
	"gorm.io/gorm"
	"strings"
)

func init() {
	database.RegisterMigration("20261024110000_payment_workday_sla", upPaymentWorkdaySLA, downPaymentWorkdaySLA)
}
func upPaymentWorkdaySLA(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Adding the marker never recalculates historical deadlines.
		if err := tx.AutoMigrate(&entities.Permohonan{}, &entities.PermohonanActivity{}); err != nil {
			return err
		}
		quoted := []string{}
		for _, day := range businessday.HolidayDates() {
			quoted = append(quoted, "'"+day+"'::date")
		}
		return tx.Exec(fmt.Sprintf(`CREATE OR REPLACE FUNCTION colabora_sla_remaining(deadline date,basis text,reference_date date)
   RETURNS integer LANGUAGE SQL STABLE AS $$
   SELECT CASE WHEN deadline IS NULL THEN NULL
    WHEN basis <> 'payment_workdays_v1' THEN deadline-reference_date
    WHEN LEAST(deadline,reference_date) < DATE '2026-01-01' OR GREATEST(deadline,reference_date) > DATE '2027-12-31' THEN NULL
    ELSE (CASE WHEN deadline < reference_date THEN -1 ELSE 1 END) *
     (SELECT COUNT(*)::integer FROM generate_series(LEAST(deadline,reference_date)+1,GREATEST(deadline,reference_date),INTERVAL '1 day') d
      WHERE EXTRACT(ISODOW FROM d) < 6 AND d::date <> ALL(ARRAY[%s])) END
   $$`, strings.Join(quoted, ","))).Error
	})
}
func downPaymentWorkdaySLA(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&entities.Permohonan{}).Where("sla_basis = ?", businessday.PaymentBasis).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("cannot remove SLA basis while payment workday requests exist")
		}
		if err := tx.Exec("DROP FUNCTION IF EXISTS colabora_sla_remaining(date,text,date)").Error; err != nil {
			return err
		}
		if err := tx.Migrator().DropColumn(&entities.PermohonanActivity{}, "sla_basis"); err != nil {
			return err
		}
		return tx.Migrator().DropColumn(&entities.Permohonan{}, "sla_basis")
	})
}
