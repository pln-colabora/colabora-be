// Package workflow evaluates the COLABORA dependency graph without I/O.
package workflow

type Code string

const (
	Permohonan      Code = "permohonan"
	Survei          Code = "survei"
	RAB             Code = "rab_kko_kkf"
	KebutuhanTiang  Code = "kebutuhan_tiang"
	Perluasan       Code = "permohonan_perluasan"
	NPS             Code = "nps_delegation"
	WOTiang         Code = "wo_tiang"
	WOKonstruksi    Code = "wo_konstruksi"
	WOAPP           Code = "wo_app"
	Reservasi       Code = "reservasi_material"
	Tera            Code = "tera_app"
	WOPDKB          Code = "wo_pdkb"
	PKVendor        Code = "pk_vendor"
	PemasanganTiang Code = "pemasangan_tiang"
	Konstruksi      Code = "pelaksanaan_konstruksi"
	DokumentasiPDKB Code = "pdkb_documentation"
	Energize        Code = "energize_jaringan"
	SRAPP           Code = "pemasangan_sr_app"
	PDL             Code = "entri_mutasi_pdl"
	AIL             Code = "arsip_ail"
	Selesai         Code = "selesai"
)

type Condition string

const (
	Always        Condition = "always"
	PolesRequired Condition = "poles_required"
	PDKBRequired  Condition = "pdkb_required"
)

// Definition is returned as a detached copy. Nil activity/SLA references mean
// there is no numbered activity or independent SLA rule respectively.
type Definition struct {
	Code              Code
	ActivityNumber    *int16
	SLAActivityNumber *int16
	Stage             int16
	OwnerJTRJTM       string
	OwnerPLGTM        string
	Applicability     Condition
	Prerequisites     []Code // ALL must be completed or legitimately skipped.
}

func number(n int16) *int16 {
	if n == 0 {
		return nil
	}
	return &n
}
func def(c Code, activity, stage int16, owner, tm string, condition Condition, deps ...Code) Definition {
	if tm == "" {
		tm = owner
	}
	sla := activity
	if activity == 5 {
		sla = 0
	}
	return Definition{c, number(activity), number(sla), stage, owner, tm, condition, deps}
}

// Topological order is deterministic, including within presentation stages.
var definitions = []Definition{
	def(Permohonan, 1, 1, "pelayanan-pelanggan", "nps", Always),
	def(Survei, 2, 2, "teknik", "perencanaan", Always, Permohonan),
	def(RAB, 3, 3, "teknik", "perencanaan", Always, Survei),
	def(KebutuhanTiang, 0, 3, "teknik", "perencanaan", Always, RAB),
	def(Perluasan, 4, 3, "nps", "", Always, RAB, KebutuhanTiang),
	def(NPS, 5, 3, "nps", "", Always, Perluasan),
	def(WOTiang, 6, 4, "perencanaan", "", PolesRequired, NPS),
	def(WOKonstruksi, 7, 4, "konstruksi", "", Always, NPS),
	def(WOAPP, 8, 4, "transaksi-energi", "", Always, NPS),
	def(Reservasi, 9, 4, "transaksi-energi", "", Always, WOAPP),
	def(Tera, 10, 4, "transaksi-energi", "", Always, Reservasi),
	def(WOPDKB, 0, 4, "konstruksi", "", PDKBRequired, WOKonstruksi),
	def(PKVendor, 0, 4, "konstruksi", "", Always, WOKonstruksi, WOPDKB),
	def(PemasanganTiang, 11, 5, "vendor-tiang", "", PolesRequired, WOTiang),
	def(Konstruksi, 12, 5, "vendor-konstruksi", "", Always, WOKonstruksi, PKVendor, WOPDKB),
	def(DokumentasiPDKB, 0, 5, "pdkb", "", PDKBRequired, Konstruksi),
	def(Energize, 13, 6, "teknik", "jaringan", Always, Konstruksi, PemasanganTiang, DokumentasiPDKB),
	def(SRAPP, 14, 6, "vendor-sr-app", "vendor-konstruksi", Always, Konstruksi, Tera),
	def(PDL, 15, 7, "pelayanan-pelanggan", "", Always, Energize, SRAPP),
	def(AIL, 16, 7, "pelayanan-pelanggan", "", Always, PDL),
	def(Selesai, 17, 7, "pelayanan-pelanggan", "", Always, AIL),
}

func clone(d Definition) Definition {
	if d.ActivityNumber != nil {
		d.ActivityNumber = number(*d.ActivityNumber)
	}
	if d.SLAActivityNumber != nil {
		d.SLAActivityNumber = number(*d.SLAActivityNumber)
	}
	d.Prerequisites = append([]Code(nil), d.Prerequisites...)
	return d
}
func Definitions() []Definition {
	out := make([]Definition, len(definitions))
	for i, d := range definitions {
		out[i] = clone(d)
	}
	return out
}
func Lookup(code Code) (Definition, bool) {
	for _, d := range definitions {
		if d.Code == code {
			return clone(d), true
		}
	}
	return Definition{}, false
}

// CodeForActivity maps a numbered legacy activity to its canonical primary node.
// Decisions and supporting nodes intentionally have no numeric representation.
func CodeForActivity(activity int16) (Code, bool) {
	for _, d := range definitions {
		if d.ActivityNumber != nil && *d.ActivityNumber == activity {
			return d.Code, true
		}
	}
	return "", false
}

func ValidConnection(connection string) bool {
	switch connection {
	case "JTR", "JTM/Gardu", "PLG TM <5 GWNG", "PLG TM >5 GWNG":
		return true
	}
	return false
}
func Owner(code Code, connection string) (string, bool) {
	d, ok := Lookup(code)
	if !ok || !ValidConnection(connection) {
		return "", false
	}
	if connection == "JTR" || connection == "JTM/Gardu" {
		return d.OwnerJTRJTM, true
	}
	return d.OwnerPLGTM, true
}
