package models

import (
	"fmt"
	"strings"
)

func MenuFormat[T interface{ MenuFormat() string }](in T) string { return in.MenuFormat() }

func (l LeaderProfile) MenuFormat() string {
	return fmt.Sprintf("%s (%d-%d) -- %d lesson(s) [%d unique] at %d singing(s)",
		l.Name, l.FirstYear, l.LastYear, l.LessonCount, l.UniqueLessonCount, l.SingingCount,
	)
}

func (info SingingInfo) MenuFormat() string {
	return fmt.Sprintf("%s -- %s (%s)",
		info.SingingDate.Time().Format("2006-01-02"),
		strings.ReplaceAll(info.SingingName, "\\n", "; "),
		info.SingingLocation,
	)
}

func (s SongDetail) MenuFormat() string {
	return fmt.Sprintf("pg %s -- %s", s.PageNum, s.SongTitle)
}
