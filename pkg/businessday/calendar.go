// Package businessday defines the approved SLA calendar: Monday–Friday in WIB,
// excluding national holidays. Collective leave remains a working day.
package businessday

import (
	"fmt"
	"sort"
	"time"
)

const PaymentBasis = "payment_workdays_v1"

var WIB = time.FixedZone("WIB", 7*60*60)

// Sources: SKB 1497/2/5 (2025) and SKB 1205/3/2 (2026).
// https://www.kemenkopmk.go.id/node/5862
// https://www.kemenkopmk.go.id/node/6444
var nationalHolidays = map[string]bool{
	"2026-01-01": true, "2026-01-16": true, "2026-02-17": true, "2026-03-19": true, "2026-03-21": true, "2026-03-22": true,
	"2026-04-03": true, "2026-04-05": true, "2026-05-01": true, "2026-05-14": true, "2026-05-27": true, "2026-05-31": true,
	"2026-06-01": true, "2026-06-16": true, "2026-08-17": true, "2026-08-25": true, "2026-12-25": true,
	"2027-01-01": true, "2027-01-05": true, "2027-02-06": true, "2027-03-08": true, "2027-03-10": true, "2027-03-11": true,
	"2027-03-26": true, "2027-03-28": true, "2027-05-01": true, "2027-05-06": true, "2027-05-17": true, "2027-05-20": true,
	"2027-06-01": true, "2027-06-06": true, "2027-08-15": true, "2027-08-17": true, "2027-12-25": true, "2027-12-26": true,
}

func HolidayDates() []string {
	dates := make([]string, 0, len(nationalHolidays))
	for date := range nationalHolidays {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	return dates
}
func Date(value time.Time) time.Time {
	value = value.In(WIB)
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, WIB)
}
func Supported(value time.Time) error {
	if year := value.In(WIB).Year(); year < 2026 || year > 2027 {
		return fmt.Errorf("kalender SLA tahun %d belum tersedia (tersedia: 2026–2027)", year)
	}
	return nil
}
func IsWorking(value time.Time) (bool, error) {
	if err := Supported(value); err != nil {
		return false, err
	}
	value = value.In(WIB)
	return value.Weekday() != time.Saturday && value.Weekday() != time.Sunday && !nationalHolidays[value.Format("2006-01-02")], nil
}

// Deadline normalizes a non-working payment date to H/G, then adds the offset.
func Deadline(payment time.Time, offset int) (time.Time, error) {
	if offset < 0 {
		return time.Time{}, fmt.Errorf("invalid SLA offset")
	}
	date := Date(payment)
	for {
		working, err := IsWorking(date)
		if err != nil {
			return time.Time{}, err
		}
		if working {
			break
		}
		date = date.AddDate(0, 0, 1)
	}
	for offset > 0 {
		date = date.AddDate(0, 0, 1)
		working, err := IsWorking(date)
		if err != nil {
			return time.Time{}, err
		}
		if working {
			offset--
		}
	}
	// PostgreSQL DATE is a civil date. Store UTC midnight with the same date
	// components so pgx does not shift WIB midnight into the previous UTC day.
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC), nil
}

// Remaining counts (earlier date, later date], so overdue does not grow on weekends.
func Remaining(deadline, reference time.Time) (int, error) {
	start, end := Date(reference), Date(deadline)
	sign := 1
	if end.Before(start) {
		start, end = end, start
		sign = -1
	}
	if err := Supported(start); err != nil {
		return 0, err
	}
	if err := Supported(end); err != nil {
		return 0, err
	}
	days := 0
	for date := start.AddDate(0, 0, 1); !date.After(end); date = date.AddDate(0, 0, 1) {
		working, err := IsWorking(date)
		if err != nil {
			return 0, err
		}
		if working {
			days++
		}
	}
	return sign * days, nil
}
