package query

import (
	"github.com/Caknoooo/go-pagination"
	"github.com/pln-colabora/colabora-be/pkg/rbac"
	"gorm.io/gorm"
)

type User struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	TelpNumber string `json:"telp_number"`
	Role       string `json:"role"`
	Unit       string `json:"unit"`
	ImageUrl   string `json:"image_url"`
	IsVerified bool   `json:"is_verified"`
}

type UserFilter struct {
	pagination.BaseFilter
}

func (f *UserFilter) ApplyFilters(query *gorm.DB) *gorm.DB {
	// Apply your filters here
	return query
}

func (f *UserFilter) GetTableName() string {
	return "users"
}

func (f *UserFilter) GetSearchFields() []string {
	return []string{"name"}
}

func (f *UserFilter) GetDefaultSort() string {
	return "id asc"
}

func (f *UserFilter) GetIncludes() []string {
	return f.Includes
}

func (f *UserFilter) GetPagination() pagination.PaginationRequest {
	return f.Pagination
}

func (f *UserFilter) Validate() {
	var validIncludes []string
	allowedIncludes := f.GetAllowedIncludes()
	for _, include := range f.Includes {
		if allowedIncludes[include] {
			validIncludes = append(validIncludes, include)
		}
	}
	f.Includes = validIncludes
}

func (f *UserFilter) GetAllowedIncludes() map[string]bool {
	return map[string]bool{}
}

type Vendor struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	TelpNumber string `json:"telp_number"`
	Role       string `json:"role"`
	Unit       string `json:"unit"`
}

type VendorFilter struct {
	pagination.BaseFilter
	Role string `json:"role" form:"role"`
}

func (f *VendorFilter) ApplyFilters(query *gorm.DB) *gorm.DB {
	query = query.Where("role IN ?", rbac.VendorRoles())
	if f.Role != "" {
		query = query.Where("role = ?", f.Role)
	}
	return query
}

func (f *VendorFilter) GetTableName() string {
	return "users"
}

func (f *VendorFilter) GetSearchFields() []string {
	return []string{"name"}
}

func (f *VendorFilter) GetDefaultSort() string {
	return "id asc"
}

func (f *VendorFilter) GetIncludes() []string {
	return f.Includes
}

func (f *VendorFilter) GetPagination() pagination.PaginationRequest {
	return f.Pagination
}

func (f *VendorFilter) Validate() {
	var validIncludes []string
	allowedIncludes := f.GetAllowedIncludes()
	for _, include := range f.Includes {
		if allowedIncludes[include] {
			validIncludes = append(validIncludes, include)
		}
	}
	f.Includes = validIncludes
}

func (f *VendorFilter) GetAllowedIncludes() map[string]bool {
	return map[string]bool{}
}
