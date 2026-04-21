package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/tychoish/fun/mdwn"
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

func (SingingInfo) ColumnNames() []mdwn.Column {
	return []mdwn.Column{
		{Name: "Date"},
		{Name: "Name", Elastic: true},
		{Name: "Location", MaxWidth: 40},
		{Name: "Lessons", RightAlign: true},
		{Name: "Leaders", RightAlign: true},
	}
}

func (r SingingInfo) RowValues() []string {
	return []string{
		r.SingingDate.Time().Format(time.DateOnly),
		strings.ReplaceAll(r.SingingName, "\\n", "; "),
		r.SingingLocation,
		strconv.FormatInt(r.NumberOfLessons, 10),
		strconv.FormatInt(r.NumberOfLeaders, 10),
	}
}

func (r SingingInfo) LineItem() *mdwn.Builder {
	var mb mdwn.Builder
	mb.BulletListItem(r.MenuFormat())
	return &mb
}

func (SongDetail) ColumnNames() []mdwn.Column {
	return []mdwn.Column{{Name: "Page"}, {Name: "Title"}, {Name: "Keys"}, {Name: "Meter"}}
}

func (r SongDetail) RowValues() []string {
	return []string{r.PageNum, r.SongTitle, r.Keys, r.SongMeter}
}

func (r SongDetail) LineItem() *mdwn.Builder {
	var mb mdwn.Builder
	mb.BulletListItem(r.MenuFormat())
	return &mb
}

func (LeaderProfile) ColumnNames() []mdwn.Column {
	return []mdwn.Column{{Name: "Name"}, {Name: "Lessons", RightAlign: true}, {Name: "Singings", RightAlign: true}, {Name: "First", RightAlign: true}, {Name: "Last", RightAlign: true}}
}

func (r LeaderProfile) RowValues() []string {
	return []string{
		r.Name,
		strconv.FormatInt(r.LessonCount, 10),
		strconv.FormatInt(r.SingingCount, 10),
		strconv.FormatInt(r.FirstYear, 10),
		strconv.FormatInt(r.LastYear, 10),
	}
}

func (r LeaderProfile) LineItem() *mdwn.Builder {
	var mb mdwn.Builder
	mb.BulletListItem(r.MenuFormat())
	return &mb
}
