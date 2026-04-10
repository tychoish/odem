package models

import (
	"fmt"
	"time"
)

type DateTime time.Time

func (d *DateTime) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("DateTime: expected string, got %T", src)
	}
	if normalized, ok := dateExceptions[s]; ok {
		s = normalized
	}
	t, err := time.Parse("January 2, 2006", s)
	if err != nil {
		return fmt.Errorf("DateTime.Scan: %w", err)
	}
	*d = DateTime(t)
	return nil
}

func (d DateTime) Time() time.Time { return time.Time(d) }
func (d DateTime) String() string  { return time.Time(d).Format(time.DateOnly) }
