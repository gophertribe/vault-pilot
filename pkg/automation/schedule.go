package automation

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// NextRun computes the next run time for a schedule definition.
func NextRun(kind, expr, tz string, from time.Time) (*time.Time, error) {
	location := time.UTC
	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return nil, fmt.Errorf("invalid timezone %q: %w", tz, err)
		}
		location = loc
	}
	localFrom := from.In(location)

	switch strings.ToLower(kind) {
	case "interval":
		d, err := time.ParseDuration(expr)
		if err != nil {
			return nil, fmt.Errorf("invalid interval expression %q: %w", expr, err)
		}
		if d <= 0 {
			return nil, fmt.Errorf("interval must be > 0")
		}
		next := localFrom.Add(d)
		utc := next.UTC()
		return &utc, nil
	case "oneshot":
		t, err := time.Parse(time.RFC3339, expr)
		if err != nil {
			return nil, fmt.Errorf("invalid oneshot expression %q: %w", expr, err)
		}
		if !t.After(from) {
			return nil, nil
		}
		utc := t.UTC()
		return &utc, nil
	case "cron":
		next, err := nextCron(expr, localFrom)
		if err != nil {
			return nil, err
		}
		if next == nil {
			return nil, nil
		}
		utc := next.UTC()
		return &utc, nil
	default:
		return nil, fmt.Errorf("unsupported schedule kind %q", kind)
	}
}

func nextCron(expr string, from time.Time) (*time.Time, error) {
	expr = strings.TrimSpace(expr)
	switch expr {
	case "@hourly":
		next := from.Truncate(time.Hour).Add(time.Hour)
		return &next, nil
	case "@daily":
		next := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location()).Add(24 * time.Hour)
		return &next, nil
	case "@weekly":
		weekdayOffset := (7 - int(from.Weekday())) % 7
		if weekdayOffset == 0 {
			weekdayOffset = 7
		}
		next := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location()).AddDate(0, 0, weekdayOffset)
		return &next, nil
	}

	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid cron expression %q (expected 5 fields)", expr)
	}

	minute, err := parseCronField(parts[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}
	hour, err := parseCronField(parts[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}
	dayOfMonth, err := parseCronField(parts[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-month field: %w", err)
	}
	month, err := parseCronField(parts[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}
	dayOfWeek, err := parseCronField(parts[4], 0, 7)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field: %w", err)
	}
	if dayOfWeek[7] {
		dayOfWeek[0] = true
	}

	domWildcard := parts[2] == "*"
	dowWildcard := parts[4] == "*"

	candidate := from.Truncate(time.Minute).Add(time.Minute)
	limit := candidate.AddDate(2, 0, 0)
	for !candidate.After(limit) {
		if !month[int(candidate.Month())] {
			candidate = time.Date(candidate.Year(), candidate.Month(), 1, 0, 0, 0, 0, candidate.Location()).AddDate(0, 1, 0)
			continue
		}
		if !hour[candidate.Hour()] {
			candidate = candidate.Add(time.Minute)
			continue
		}
		if !minute[candidate.Minute()] {
			candidate = candidate.Add(time.Minute)
			continue
		}

		domMatch := dayOfMonth[candidate.Day()]
		dowMatch := dayOfWeek[int(candidate.Weekday())]
		dayMatches := false
		if domWildcard && dowWildcard {
			dayMatches = true
		} else if domWildcard {
			dayMatches = dowMatch
		} else if dowWildcard {
			dayMatches = domMatch
		} else {
			dayMatches = domMatch || dowMatch
		}

		if dayMatches {
			return &candidate, nil
		}
		candidate = candidate.Add(time.Minute)
	}
	return nil, nil
}

func parseCronField(field string, min, max int) (map[int]bool, error) {
	allowed := make(map[int]bool, max-min+1)
	if field == "*" {
		for i := min; i <= max; i++ {
			allowed[i] = true
		}
		return allowed, nil
	}

	items := strings.Split(field, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			return nil, fmt.Errorf("empty token")
		}
		step := 1
		rangePart := item
		if strings.Contains(item, "/") {
			parts := strings.Split(item, "/")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid step expression %q", item)
			}
			rangePart = parts[0]
			s, err := strconv.Atoi(parts[1])
			if err != nil || s <= 0 {
				return nil, fmt.Errorf("invalid step in %q", item)
			}
			step = s
		}

		if rangePart == "*" {
			for i := min; i <= max; i += step {
				allowed[i] = true
			}
			continue
		}

		start, end, err := parseRange(rangePart, min, max)
		if err != nil {
			return nil, err
		}
		for i := start; i <= end; i += step {
			allowed[i] = true
		}
	}
	if len(allowed) == 0 {
		return nil, fmt.Errorf("no values selected")
	}
	return allowed, nil
}

func parseRange(part string, min, max int) (int, int, error) {
	if strings.Contains(part, "-") {
		r := strings.Split(part, "-")
		if len(r) != 2 {
			return 0, 0, fmt.Errorf("invalid range %q", part)
		}
		start, err := strconv.Atoi(r[0])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range start %q", part)
		}
		end, err := strconv.Atoi(r[1])
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range end %q", part)
		}
		if start > end || start < min || end > max {
			return 0, 0, fmt.Errorf("range out of bounds %q", part)
		}
		return start, end, nil
	}

	v, err := strconv.Atoi(part)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid value %q", part)
	}
	if v < min || v > max {
		return 0, 0, fmt.Errorf("value %d out of bounds [%d,%d]", v, min, max)
	}
	return v, v, nil
}
