package entities

// PermohonanNumberCounter stores the last allocated request number per year.
// It is updated atomically inside the same transaction that creates a request.
type PermohonanNumberCounter struct {
	RequestYear int `gorm:"primaryKey;column:request_year"`
	LastNumber  int `gorm:"not null"`
}

func (PermohonanNumberCounter) TableName() string { return "permohonan_number_counters" }
