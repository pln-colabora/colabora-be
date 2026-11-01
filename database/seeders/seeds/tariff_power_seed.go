package seeds

import (
	"strconv"
	"strings"

	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"gorm.io/gorm"
)

// TariffPowerStep is one exact power step from "Tangga Daya PLN.xlsx".
type TariffPowerStep struct {
	Tarif         string
	GolonganTarif string
	DayaMin       int64
	DayaMax       *int64
	DayaLabel     string
	Pembatas      *string
}

// powerLimiters maps every exact power step (VA) to the limiter shown in the
// official spreadsheet. Values from 210.000 VA upward are the spreadsheet's
// rounded daya/(1,73 x 20.000) results; casing is preserved as displayed.
var powerLimiters = []struct {
	daya     int64
	pembatas string
}{
	{220, "1X1"}, {450, "1x2"}, {900, "1x4"}, {1300, "1x6"}, {2200, "1x10"}, {3500, "1x16"},
	{3900, "3x6"}, {4400, "1x20"}, {5500, "1x25"}, {6600, "3x10"}, {7700, "1x35"}, {10600, "3x16"},
	{11000, "1x50"}, {13200, "3x20"}, {16500, "3x25"}, {23000, "3x35"}, {33000, "3x50"}, {41500, "3x63"},
	{53000, "3x80"}, {66000, "3x100"}, {82500, "3x125"}, {105000, "3x160"}, {131000, "3x200"},
	{147000, "3x225"}, {164000, "3x250"}, {197000, "3x300"},
	{210000, "6"}, {240000, "7"}, {345000, "10"}, {555000, "16"}, {690000, "20"}, {865000, "25"},
	{1110000, "32"}, {1210000, "35"}, {1385000, "40"}, {1730000, "50"}, {2180000, "63"}, {2770000, "80"},
	{3465000, "100"}, {4330000, "125"}, {5540000, "160"}, {6930000, "200"}, {8660000, "250"},
	{10380000, "300"}, {11000000, "318"}, {12000000, "347"}, {13800000, "399"},
}

const (
	tariffI4Min   int64 = 30000000
	tariffI4Label       = "> 30.000.000 VA"
)

// powerRange returns every power step from min to max inclusive, in table order.
func powerRange(min, max int64) []int64 {
	steps := make([]int64, 0)
	for _, step := range powerLimiters {
		if step.daya >= min && step.daya <= max {
			steps = append(steps, step.daya)
		}
	}
	return steps
}

func tarifForGolongan(golongan string) string {
	switch golongan[0] {
	case 'R':
		return rbac.TarifRumahTangga
	case 'B':
		return rbac.TarifBisnis
	case 'S':
		return rbac.TarifSosial
	case 'P':
		return rbac.TarifPemerintah
	case 'I':
		return rbac.TarifIndustri
	}
	return ""
}

// FormatDayaLabel renders a power in Indonesian thousands notation, e.g. "13.800.000 VA".
func FormatDayaLabel(daya int64) string {
	digits := strconv.FormatInt(daya, 10)
	var b strings.Builder
	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(digit)
	}
	return b.String() + " VA"
}

// TariffPowerCatalogue lists every active golongan/power step. tarif is
// derived from the golongan's leading letter.
func TariffPowerCatalogue() []TariffPowerStep {
	r3 := powerRange(6600, 197000)
	b3 := powerRange(210000, 13800000)
	s2 := powerRange(450, 197000)
	golongan := []struct {
		code  string
		steps []int64
	}{
		{"R1", []int64{450, 900, 1300, 2200}},
		{"R2", []int64{3500, 3900, 4400, 5500}},
		{"R3", r3},
		{"B1", powerRange(450, 5500)},
		{"B2", r3},
		{"B3", b3},
		{"S1", []int64{220}},
		{"S2", s2},
		{"S3", b3},
		{"P1", s2},
		{"P2", b3},
		{"I1", powerRange(450, 13200)},
		{"I2", powerRange(16500, 197000)},
		{"I3", b3},
	}
	limiters := make(map[int64]string, len(powerLimiters))
	for _, step := range powerLimiters {
		limiters[step.daya] = step.pembatas
	}

	catalogue := make([]TariffPowerStep, 0)
	for _, group := range golongan {
		for _, daya := range group.steps {
			max := daya
			pembatas := limiters[daya]
			catalogue = append(catalogue, TariffPowerStep{
				Tarif: tarifForGolongan(group.code), GolonganTarif: group.code,
				DayaMin: daya, DayaMax: &max, DayaLabel: FormatDayaLabel(daya), Pembatas: &pembatas,
			})
		}
	}
	return append(catalogue, TariffPowerStep{
		Tarif: rbac.TarifIndustri, GolonganTarif: "I4", DayaMin: tariffI4Min, DayaLabel: tariffI4Label,
	})
}

func ListTariffPowerSeeder(db *gorm.DB) error {
	return SyncTariffPowerCatalogue(db)
}

// SyncTariffPowerCatalogue upserts the exact power steps for every connection
// type and deactivates any other row (legacy "s.d." bands and hyphenated
// golongan such as "R-1"). Rows are never deleted, so it is safe to re-run.
func SyncTariffPowerCatalogue(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		keep := make([]string, 0)
		for _, connection := range rbac.AllJenisSambungan {
			for _, step := range TariffPowerCatalogue() {
				query := tx.Where("tarif = ? AND golongan_tarif = ? AND jenis_sambungan = ? AND daya_min = ?", step.Tarif, step.GolonganTarif, connection, step.DayaMin)
				if step.DayaMax != nil {
					query = query.Where("daya_max = ?", *step.DayaMax)
				} else {
					query = query.Where("daya_max IS NULL")
				}
				var existing entities.TariffPowerOption
				found := query.Limit(1).Find(&existing)
				if found.Error != nil {
					return found.Error
				}
				if found.RowsAffected == 0 {
					existing = entities.TariffPowerOption{
						Tarif: step.Tarif, GolonganTarif: step.GolonganTarif, JenisSambungan: connection,
						DayaMin: step.DayaMin, DayaMax: step.DayaMax, Label: step.DayaLabel,
						DayaLabel: step.DayaLabel, Pembatas: step.Pembatas, Active: true,
					}
					if err := tx.Create(&existing).Error; err != nil {
						return err
					}
				} else if err := tx.Model(&existing).Updates(map[string]any{
					"label": step.DayaLabel, "daya_label": step.DayaLabel, "pembatas": step.Pembatas, "active": true,
				}).Error; err != nil {
					return err
				}
				keep = append(keep, existing.ID.String())
			}
		}
		return tx.Model(&entities.TariffPowerOption{}).Where("id NOT IN ?", keep).Update("active", false).Error
	})
}
