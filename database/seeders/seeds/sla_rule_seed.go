package seeds

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/pln-colabora/colabora-be/database/entities"
	"gorm.io/gorm"
)

func ListSLARuleSeeder(db *gorm.DB) error {
	jsonFile, err := os.Open("./database/seeders/json/sla_rules.json")
	if err != nil {
		return err
	}

	jsonData, err := io.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	var listSLARule []entities.SLARule
	if err := json.Unmarshal(jsonData, &listSLARule); err != nil {
		return err
	}

	hasTable := db.Migrator().HasTable(&entities.SLARule{})
	if !hasTable {
		if err := db.Migrator().CreateTable(&entities.SLARule{}); err != nil {
			return err
		}
	}

	for _, data := range listSLARule {
		var rule entities.SLARule
		err := db.Where(&entities.SLARule{ActivityNumber: data.ActivityNumber, JenisSambungan: data.JenisSambungan}).First(&rule).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		isData := db.Find(&rule, "activity_number = ? AND jenis_sambungan = ?", data.ActivityNumber, data.JenisSambungan).RowsAffected
		if isData == 0 {
			if err := db.Create(&data).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
