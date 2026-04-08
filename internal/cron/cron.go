// Package cron provides a minimal cron expression parser.
//
// Supported format: "minute hour day-of-month month day-of-week"
// Each field supports: * (any), number, range (1-5), step (*/5, 1-30/2), list (1,3,5).
package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// NextAfter returns the next time after t that matches the cron expression.
// Returns zero time if no match is found within 4 years (to cover leap cycles).
func NextAfter(expr string, t time.Time) (time.Time, error) {
	fields, err := parse(expr)
	if err != nil {
		return time.Time{}, err
	}

	// Start from the next minute boundary.
	t = t.Truncate(time.Minute).Add(time.Minute)

	// Search across at least one leap cycle so Feb 29 schedules remain reachable.
	limit := t.AddDate(4, 0, 0)
	for t.Before(limit) {
		if fields[3].has(int(t.Month())) && fields[2].has(t.Day()) && fields[4].has(int(t.Weekday())) &&
			fields[1].has(t.Hour()) && fields[0].has(t.Minute()) {
			return t, nil
		}

		// Skip ahead intelligently: if month doesn't match, jump to next month.
		if !fields[3].has(int(t.Month())) {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
			continue
		}
		// If day doesn't match, jump to next day.
		if !fields[2].has(t.Day()) || !fields[4].has(int(t.Weekday())) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}
		// If hour doesn't match, jump to next hour.
		if !fields[1].has(t.Hour()) {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
			continue
		}
		// Otherwise advance one minute.
		t = t.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("no match within 4 years")
}

// Validate checks whether a cron expression is syntactically valid.
func Validate(expr string) error {
	_, err := parse(expr)
	return err
}

// bitset is a simple fixed-size bit set for values 0..59.
type bitset [1]uint64

func (b *bitset) set(v int)     { b[0] |= 1 << uint(v) }
func (b *bitset) has(v int) bool { return b[0]&(1<<uint(v)) != 0 }

func parse(expr string) ([5]bitset, error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return [5]bitset{}, fmt.Errorf("cron: expected 5 fields, got %d", len(parts))
	}
	ranges := [5][2]int{
		{0, 59}, // minute
		{0, 23}, // hour
		{1, 31}, // day
		{1, 12}, // month
		{0, 6},  // weekday
	}
	var fields [5]bitset
	for i, part := range parts {
		bs, err := parseField(part, ranges[i][0], ranges[i][1])
		if err != nil {
			return [5]bitset{}, fmt.Errorf("cron field %d (%s): %w", i, part, err)
		}
		fields[i] = bs
	}
	return fields, nil
}

func parseField(field string, lo, hi int) (bitset, error) {
	var bs bitset
	for _, token := range strings.Split(field, ",") {
		step := 1
		if idx := strings.Index(token, "/"); idx >= 0 {
			s, err := strconv.Atoi(token[idx+1:])
			if err != nil || s <= 0 {
				return bs, fmt.Errorf("invalid step %q", token)
			}
			step = s
			token = token[:idx]
		}
		switch {
		case token == "*":
			for v := lo; v <= hi; v += step {
				bs.set(v)
			}
		case strings.Contains(token, "-"):
			parts := strings.SplitN(token, "-", 2)
			a, err1 := strconv.Atoi(parts[0])
			b, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil || a < lo || b > hi || a > b {
				return bs, fmt.Errorf("invalid range %q", token)
			}
			for v := a; v <= b; v += step {
				bs.set(v)
			}
		default:
			v, err := strconv.Atoi(token)
			if err != nil || v < lo || v > hi {
				return bs, fmt.Errorf("invalid value %q (range %d–%d)", token, lo, hi)
			}
			bs.set(v)
		}
	}
	return bs, nil
}
