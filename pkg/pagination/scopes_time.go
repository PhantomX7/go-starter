package pagination

import (
	"time"

	"gorm.io/gorm"
)

// parseDate parses a YYYY-MM-DD date in the configured timezone.
func (p *Pagination) parseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, p.options.Timezone)
}

// parseDateTime parses a YYYY-MM-DD HH:MM:SS timestamp in the configured timezone.
func (p *Pagination) parseDateTime(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", s, p.options.Timezone)
}

// buildDateScope builds date filter scope.
//
// Day ranges use a half-open interval [start, nextDay) rather than a closed
// BETWEEN ending at 23:59:59.999999999. Closed ranges with sub-microsecond
// precision are unsafe against Postgres `timestamptz` (μs-only) and MySQL
// `DATETIME` (μs at best): the database truncates or rounds the literal
// before comparison, so rows stored in the last microsecond of a day may be
// silently excluded or pulled in from the next day. Half-open intervals are
// precision-independent.
func (p *Pagination) buildDateScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	switch op.Operator {
	case OperatorEquals:
		t, err := p.parseDate(op.Values[0])
		if err != nil {
			return nil
		}
		nextDay := t.AddDate(0, 0, 1) // calendar-correct across DST transitions
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" >= ? AND "+field+" < ?", t, nextDay)
		}

	case OperatorBetween:
		start, err := p.parseDate(op.Values[0])
		if err != nil {
			return nil
		}
		end, err := p.parseDate(op.Values[1])
		if err != nil {
			return nil
		}
		nextDay := end.AddDate(0, 0, 1)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" >= ? AND "+field+" < ?", start, nextDay)
		}

	case OperatorGte:
		t, err := p.parseDate(op.Values[0])
		if err != nil {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" >= ?", t)
		}

	case OperatorLte:
		t, err := p.parseDate(op.Values[0])
		if err != nil {
			return nil
		}
		nextDay := t.AddDate(0, 0, 1) // calendar-correct across DST transitions
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" < ?", nextDay)
		}
	case OperatorIsNull, OperatorIsNotNull:
		return nullScope(field, op.Operator)
	}
	return nil
}

// buildDateTimeScope builds datetime filter scope.
//
// The input grammar "YYYY-MM-DD HH:MM:SS" is 1-second resolution, but the
// underlying column can store fractional seconds (Postgres timestamptz keeps
// microseconds, MySQL DATETIME(6) keeps microseconds). A closed range like
// "BETWEEN '…23:59:59' AND '…23:59:59'" silently drops any row whose
// fractional component is non-zero — the user thinks they asked "all rows in
// that second" but only got rows stored at exactly *.000000. The same
// precision trap that buildDateScope avoids for day boundaries applies at
// every second boundary here, so the same fix applies: half-open
// [t, t + 1s). This works identically on Postgres, MySQL, and SQLite.
func (p *Pagination) buildDateTimeScope(field string, op FilterOperation) func(*gorm.DB) *gorm.DB {
	switch op.Operator {
	case OperatorEquals:
		t, err := p.parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		nextSec := t.Add(time.Second)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" >= ? AND "+field+" < ?", t, nextSec)
		}

	case OperatorBetween:
		start, err := p.parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		end, err := p.parseDateTime(op.Values[1])
		if err != nil {
			return nil
		}
		endExclusive := end.Add(time.Second)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" >= ? AND "+field+" < ?", start, endExclusive)
		}

	case OperatorGte:
		t, err := p.parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		return func(db *gorm.DB) *gorm.DB { return db.Where(field+" >= ?", t) }

	case OperatorLte:
		t, err := p.parseDateTime(op.Values[0])
		if err != nil {
			return nil
		}
		endExclusive := t.Add(time.Second)
		return func(db *gorm.DB) *gorm.DB {
			return db.Where(field+" < ?", endExclusive)
		}
	case OperatorIsNull, OperatorIsNotNull:
		return nullScope(field, op.Operator)
	}
	return nil
}
