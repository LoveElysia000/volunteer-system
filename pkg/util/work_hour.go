package util

import (
	"errors"
	"math"
	"time"

	gmysql "github.com/go-sql-driver/mysql"
)

// RoundHours keeps hour precision to 2 decimals.
func RoundHours(v float64) float64 {
	return math.Round(v*100) / 100
}

// CalcGrantedHours calculates granted hours from check-in/out and duration cap.
func CalcGrantedHours(duration float64, checkIn, checkOut time.Time) float64 {
	if checkOut.Before(checkIn) {
		return 0
	}

	elapsed := checkOut.Sub(checkIn).Hours()
	if elapsed < 0 {
		elapsed = 0
	}

	hours := elapsed
	if duration > 0 && hours > duration {
		hours = duration
	}

	return RoundHours(hours)
}

// IsDuplicateEntryErr reports whether err is a MySQL duplicate-key error.
func IsDuplicateEntryErr(err error) bool {
	if err == nil {
		return false
	}

	var mysqlErr *gmysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1062
	}
	return false
}
