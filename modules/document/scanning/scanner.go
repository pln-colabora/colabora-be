package scanning

import "context"

const (
	StatusNotScanned = "not_scanned"
	StatusClean      = "clean"
	StatusInfected   = "infected"
)

// Scanner is the integration boundary for inspecting untrusted upload bytes.
// Implementations must not put filenames or file contents in returned errors.
type Scanner interface {
	Scan(context.Context, []byte) (string, error)
}

type DisabledScanner struct{}

func (DisabledScanner) Scan(context.Context, []byte) (string, error) {
	return StatusNotScanned, nil
}
