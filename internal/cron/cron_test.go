package cron

import (
	"testing"
	"time"
)

func TestNextAfter(t *testing.T) {
	base := time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		expr string
		want time.Time
	}{
		{"every minute", "* * * * *", time.Date(2026, 4, 7, 10, 31, 0, 0, time.UTC)},
		{"every 5 min", "*/5 * * * *", time.Date(2026, 4, 7, 10, 35, 0, 0, time.UTC)},
		{"daily at 9", "0 9 * * *", time.Date(2026, 4, 8, 9, 0, 0, 0, time.UTC)},
		{"daily at 12", "0 12 * * *", time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)},
		{"weekday 9am", "0 9 * * 1-5", time.Date(2026, 4, 8, 9, 0, 0, 0, time.UTC)}, // Apr 7 is Tuesday, next is Wed
		{"first of month", "0 9 1 * *", time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)},
		{"leap day", "0 9 29 2 *", time.Date(2028, 2, 29, 9, 0, 0, 0, time.UTC)}, // 2026 is not a leap year
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NextAfter(tt.expr, base)
			if err != nil {
				t.Fatalf("NextAfter(%q): %v", tt.expr, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("NextAfter(%q) = %v, want %v", tt.expr, got, tt.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	valid := []string{"* * * * *", "0 9 * * *", "*/5 * * * *", "0 9 1-15 * 1-5", "30 8,12,18 * * *"}
	for _, expr := range valid {
		if err := Validate(expr); err != nil {
			t.Errorf("Validate(%q) unexpected error: %v", expr, err)
		}
	}

	invalid := []string{"", "* *", "60 * * * *", "* 25 * * *", "* * 0 * *", "* * * 13 *", "* * * * 7"}
	for _, expr := range invalid {
		if err := Validate(expr); err == nil {
			t.Errorf("Validate(%q) expected error", expr)
		}
	}
}
