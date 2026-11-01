package businessday

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func parsed(s string) time.Time { value, _ := time.ParseInLocation("2006-01-02", s, WIB); return value }
func TestPaymentDeadlineCalendar(t *testing.T) {
	for _, test := range []struct {
		payment string
		offset  int
		want    string
	}{{"2026-10-24", 0, "2026-10-26"}, {"2026-10-24", 1, "2026-10-27"}, {"2026-05-29", 1, "2026-06-02"}, {"2026-12-24", 0, "2026-12-24"}, {"2026-12-24", 1, "2026-12-28"}, {"2026-12-31", 1, "2027-01-04"}, {"2027-03-09", 1, "2027-03-12"}} {
		date, err := Deadline(parsed(test.payment), test.offset)
		require.NoError(t, err)
		require.Equal(t, test.want, date.Format("2006-01-02"))
	}
	_, err := Deadline(parsed("2027-12-31"), 1)
	require.Error(t, err)
	_, err = Deadline(parsed("2025-12-31"), 0)
	require.Error(t, err)
}
func TestRemainingWorkingDaysAndWIB(t *testing.T) {
	for _, test := range []struct {
		deadline, now string
		want          int
	}{{"2026-10-23", "2026-10-24", 0}, {"2026-10-23", "2026-10-25", 0}, {"2026-10-23", "2026-10-26", -1}, {"2026-10-22", "2026-10-25", -1}, {"2026-06-02", "2026-05-29", 1}} {
		days, err := Remaining(parsed(test.deadline), parsed(test.now))
		require.NoError(t, err)
		require.Equal(t, test.want, days)
	}
	utc := time.Date(2026, 10, 25, 18, 0, 0, 0, time.UTC)
	days, err := Remaining(parsed("2026-10-23"), utc)
	require.NoError(t, err)
	require.Equal(t, -1, days)
	_, err = Remaining(parsed("2026-10-23"), parsed("2028-01-01"))
	require.Error(t, err)
}
