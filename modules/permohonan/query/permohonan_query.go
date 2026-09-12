package query

import (
	"github.com/Caknoooo/go-pagination"
	"github.com/pln-colabora/colabora-be/database/entities"
	"github.com/pln-colabora/colabora-be/modules/permohonan/dto"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"gorm.io/gorm"
)

// Permohonan is the dashboard-row projection returned by the paginated list query —
// separate from dto.PermohonanResponse, matching how modules/user/query/user_query.go
// keeps its list projection separate from dto.UserResponse.
type Permohonan struct {
	ID                  string                        `json:"id"`
	NoPermohonan        string                        `json:"no_permohonan"`
	JenisPermohonan     string                        `json:"jenis_permohonan"`
	JenisSambungan      string                        `json:"jenis_sambungan"`
	UlpUnit             string                        `json:"ulp_unit"`
	PelangganNama       string                        `json:"pelanggan_nama"`
	PelangganAlamat     string                        `json:"pelanggan_alamat"`
	PelangganNoHp       string                        `json:"pelanggan_no_hp"`
	RequestDate         string                        `json:"request_date"`
	CurrentStage        int16                         `json:"current_stage"`
	Status              string                        `json:"status"`
	SlaDeadline         *string                       `json:"sla_deadline"`
	SlaStatus           string                        `json:"sla_status"`
	KebutuhanTiang      *bool                         `json:"kebutuhan_tiang"`
	NpsDelegationStatus *string                       `json:"nps_delegation_status"`
	PerluPdkb           *bool                         `json:"perlu_pdkb"`
	CreatedBy           string                        `json:"created_by"`
	CreatedAt           string                        `json:"created_at"`
	UpdatedAt           string                        `json:"updated_at"`
	WorkflowNodes       []entities.PermohonanActivity `gorm:"-" json:"-"`
	AvailableActions    []dto.AvailableAction         `gorm:"-" json:"available_actions"`
}

// sla bucket thresholds — "due soon" isn't specified anywhere in the source docs, 2 days
// is a reasonable default and easy to change here if it turns out to be wrong.
const slaDueSoonDays = 2

type PermohonanFilter struct {
	pagination.BaseFilter

	Ulp            string `json:"ulp" form:"ulp"`
	JenisSambungan string `json:"jenis_sambungan" form:"jenis_sambungan"`
	Stage          int16  `json:"stage" form:"stage"`
	Status         string `json:"status" form:"status"`
	Sla            string `json:"sla" form:"sla"`
	Scope          string `json:"scope" form:"scope"`

	// Set by the controller from the authenticated user, never bound from the query string.
	CurrentRole   string `form:"-"`
	CurrentUnit   string `form:"-"`
	CurrentUserID string `form:"-" json:"-"`
}

func (f *PermohonanFilter) ApplyFilters(query *gorm.DB) *gorm.DB {
	query = rbac.ApplyReadScope(query, f.CurrentRole, f.CurrentUnit, f.CurrentUserID)
	if f.Ulp != "" {
		query = query.Where("ulp_unit = ?", f.Ulp)
	}
	if f.JenisSambungan != "" {
		query = query.Where("jenis_sambungan = ?", f.JenisSambungan)
	}
	if f.Stage > 0 {
		query = query.Where("current_stage = ?", f.Stage)
	}
	if f.Status != "" {
		query = query.Where("status = ?", f.Status)
	}

	if f.Sla != "" {
		switch f.Sla {
		case "overdue":
			query = query.Where("EXISTS (SELECT 1 FROM permohonan_activities sla_node WHERE sla_node.permohonan_id = permohonan.id AND sla_node.status IN ('available', 'in_progress') AND sla_node.sla_deadline < CURRENT_DATE)")
		case "duesoon":
			query = query.Where(
				"EXISTS (SELECT 1 FROM permohonan_activities sla_node WHERE sla_node.permohonan_id = permohonan.id AND sla_node.status IN ('available', 'in_progress') AND sla_node.sla_deadline >= CURRENT_DATE AND sla_node.sla_deadline <= CURRENT_DATE + CAST(? AS integer))",
				slaDueSoonDays,
			)
		case "ontime":
			query = query.Where("EXISTS (SELECT 1 FROM permohonan_activities sla_node WHERE sla_node.permohonan_id = permohonan.id AND sla_node.status IN ('available', 'in_progress') AND sla_node.sla_deadline > CURRENT_DATE + CAST(? AS integer))", slaDueSoonDays)
		}
	}

	if f.Scope == "mine" {
		query = rbac.ApplyScopeMine(query, f.CurrentRole, f.CurrentUnit)
	}

	return query
}

func (f *PermohonanFilter) GetTableName() string { return "permohonan" }
func (f *PermohonanFilter) GetSearchFields() []string {
	return []string{"no_permohonan", "pelanggan_nama"}
}
func (f *PermohonanFilter) GetDefaultSort() string                      { return "created_at desc" }
func (f *PermohonanFilter) GetIncludes() []string                       { return f.Includes }
func (f *PermohonanFilter) GetPagination() pagination.PaginationRequest { return f.Pagination }

func (f *PermohonanFilter) Validate() {
	var validIncludes []string
	allowedIncludes := f.GetAllowedIncludes()
	for _, include := range f.Includes {
		if allowedIncludes[include] {
			validIncludes = append(validIncludes, include)
		}
	}
	f.Includes = validIncludes
}

func (f *PermohonanFilter) GetAllowedIncludes() map[string]bool { return map[string]bool{} }
