package msgui

import (
	"cmp"
	"context"
	"fmt"
	"iter"
	"strings"
	"sync/atomic"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/mdwn"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/models"
)

type Messenger func(context.Context, *db.Connection, models.Params) iter.Seq2[*mdwn.Builder, error]

func MostLed(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 20)
		header.Concat("Most led songs for **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.MostLedSongs(ctx, p.Name, p.Limit), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func SongsByWord(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		results, err := erc.FromIteratorUntil(conn.SongsByWord(ctx, p.Name, cmp.Or(p.Limit, 20)))
		if err != nil {
			yield(nil, err)
			return
		}
		if len(results) == 0 {
			yield(nil, fmt.Errorf("no songs found containing %q", p.Name))
			return
		}

		header := mdwn.MakeBuilder(len(p.Name) + 32)
		header.Concat("Songs containing **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, irt.Slice(results)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func SongLyrics(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		sl, err := conn.SongLyrics(ctx, p.Name)
		if err != nil {
			yield(nil, err)
			return
		}

		msg := mdwn.MakeBuilder(256 + len(sl.Text) + 16)
		msg.Bold(sl.PageNum, " — ", sl.SongTitle).Line()
		msg.KV("Music", sl.MusicAttribution)
		msg.KV("Words", sl.WordsAttribution)
		msg.KV("Meter", sl.SongMeter)
		msg.KV("Key", sl.Keys)
		msg.PushString("\n\n")
		msg.Paragraph(sl.Text)
		yield(msg, nil)
	}
}

func Songs(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 20)
		header.Concat("Top leaders for song **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.TopLeadersOfSong(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func Singings(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 20)
		header.Concat("Lessons for singing **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.SingingLessons(ctx, p.Name, models.FirstValidYear(p.Years)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func Buddies(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 30)
		header.Concat("Singing buddies for **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.SingingBuddies(ctx, p.Name, cmp.Or(p.Limit, 24)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func Strangers(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 30)
		header.Concat("Singing strangers for **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.SingingStrangers(ctx, p.Name, cmp.Or(p.Limit, 24)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func ActiveStrangers(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 35)
		header.Concat("Active singing strangers for **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.ActiveSingingStrangers(ctx, p.Name, cmp.Or(p.Limit, 24)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func PopularAsObserved(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 30)
		header.Concat("Popular songs as observed by **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.PopularAsObserved(ctx, p.Name, cmp.Or(p.Limit, 25)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func PopularInYears(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(256)
		header.Concat("Globally popular songs")
		if len(p.Years) > 0 {
			header.PushString(" (")
			for y := range irt.Slice(p.Years) {
				header.PushInt(y)
				header.PushString(", ")
			}
			header.PushString(")")
		} else {
			header.PushString(" (all time)")
		}
		header.PushString(":")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.GloballyPopularForYears(ctx, cmp.Or(p.Limit, 20), p.Years...), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func PopularLocally(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		var localities []models.SingingLocality
		for part := range strings.SplitSeq(p.Name, ",") {
			localities = append(localities, models.NewSingingLocality(strings.TrimSpace(part)))
		}

		header := mdwn.MakeBuilder(len(p.Name) + 30)
		header.Concat("Locally popular songs (**", p.Name, "**):")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.LocallyPopular(ctx, cmp.Or(p.Limit, 20), localities...), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func NeverSung(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 40)
		header.Concat("Songs never sung at a singing **", p.Name, "** attended:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(irt.Limit2(conn.NeverSung(ctx, p.Name), cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func NeverLed(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 25)
		header.Concat("Songs **", p.Name, "** has never led:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(irt.Limit2(conn.NeverLed(ctx, p.Name, cmp.Or(p.Limit, 20)), cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func UnfamilarHits(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 30)
		header.Concat("Unfamiliar hits for **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.TheUnfamilarHits(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func Connectedness(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(32)
		header.Concat("Leaders by connectedness:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.AllLeaderConnectedness(ctx, cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func ActiveConnectedness(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(48)
		header.Concat("Active leaders by connectedness:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.ActiveLeaderConnectedness(ctx, cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func LeaderRoleModels(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 25)
		header.Concat("Singing idols for **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.LeaderFootsteps(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func TopLeaders(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(256)
		header.Concat("Top leaders")
		if len(p.Years) > 0 {
			header.PushString(" (")
			for y := range irt.Slice(p.Years) {
				header.PushInt(y)
				header.PushString(", ")
			}
			header.PushString(")")
		}
		header.PushString(":")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, irt.Convert(erc.HandleUntil(conn.TopLeadersByLeads(ctx, cmp.Or(p.Limit, 20), p.Years...), ec.Push), models.TopLeadersWrapper(&atomic.Int64{}))) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func LeaderShare(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		var ec erc.Collector
		share, err := conn.LeaderShareOfLeads(ctx, p.Name, cmp.Or(p.Limit, 20), p.Years...)
		if !ec.PushOk(err) {
			yield(nil, ec.Resolve())
			return
		}

		mdtb := mdwn.MakeBuilder(256)
		mdtb.KV("Leader", p.Name)
		if len(p.Years) > 0 {
			yb := mdwn.MakeBuilder(32)
			for y := range irt.Slice(p.Years) {
				yb.PushInt(y)
				yb.PushString(", ")
			}
			mdtb.KV("Year(s)", yb.String())
		}
		mdtb.KV("Share of Leads", fmt.Sprintf("%.4f%%", stw.DerefZ(share)*100))

		flush(mdtb, yield)
	}
}

func LeaderLeadHistory(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 20)
		header.Concat("Lead history for **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.LeaderLeadHistory(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func LeaderSingings(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 25)
		header.Concat("Singings attended by **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.LeaderSingingsAttended(ctx, p.Name, cmp.Or(p.Limit, 0)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func LeaderFavoriteKey(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 25)
		header.Concat("Favorite keys for **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.LeaderFavoriteKey(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func LeaderDebutsByYear(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		if len(p.Years) == 0 {
			yield(nil, fmt.Errorf("year required"))
			return
		}
		year := p.Years[0]

		header := mdwn.MakeBuilder(256)
		header.Concat("Debut leaders for **")
		header.PushInt(year)
		header.PushString("**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, irt.Convert(erc.HandleUntil(conn.NewLeadersByYear(ctx, year, cmp.Or(p.Limit, 20)), ec.Push), models.WrapLeaderSongRank("Leads"))) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func SongsByKey(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(256)
		header.Concat("Songs by key")
		if len(p.Years) > 0 {
			header.PushString(" (")
			for y := range irt.Slice(p.Years) {
				header.PushInt(y)
				header.PushString(", ")
			}
			header.PushString(")")
		} else {
			header.PushString(" (all time)")
		}
		header.PushString(":")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, irt.Convert(erc.HandleUntil(conn.SongsByKey(ctx, p.Years...), ec.Push), models.WrapSongByKey)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func Top20Leaders(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(32)
		header.Concat("Leaders by top-20 leads:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, irt.Convert(erc.HandleUntil(conn.LeadersByTop20Leads(ctx, cmp.Or(p.Limit, 20)), ec.Push), models.WrapLeaderSongRank("Top-20 Leads"))) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func Top20LeadersActiveInLastYear(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(48)
		header.Concat("Top-20 leaders active in the last year:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, irt.Convert(erc.HandleUntil(conn.Top20LeadersActiveInLastYear(ctx, cmp.Or(p.Limit, 20)), ec.Push), models.WrapLeaderSongRank("Top-20 Leads"))) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func LeaderSingingsPerYear(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 30)
		header.Concat("Singings per year for **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.LeaderSingingsPerYear(ctx, p.Name), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func LeadersByKey(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 25)
		header.Concat("Leaders in key **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, irt.Convert(
			erc.HandleUntil(conn.LeadersByKey(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push),
			models.WrapLeaderSongRank("Count"))) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}

func PopularSongsByKey(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		header := mdwn.MakeBuilder(len(p.Name) + 30)
		header.Concat("Popular songs in key **", p.Name, "**:")

		var ec erc.Collector
		for md, err := range renderWithHeader(header, erc.HandleUntil(conn.PopularSongsByKey(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}
