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
		md := mdwn.MakeBuilder(len(p.Name) + 20)
		md.Concat("Most led songs for **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		mdtb := mdwn.MakeBuilder(4096)
		var ec erc.Collector
		var lastLineLength int
		for record := range erc.HandleUntil(conn.MostLedSongs(ctx, p.Name, p.Limit), ec.Push) {
			line := record.LineItem()
			lastLineLength = line.Len()

			if mdtb.Len() >= 4000 || mdtb.Len()+lastLineLength >= 4000 {
				if !yield(mdtb, nil) {
					return
				}
				mdtb = mdwn.MakeBuilder(4096)
				lastLineLength = 0
			}
			(&mdtb.Mutable).Push(&line.Mutable)
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
			return
		}

		flush(mdtb, yield)
	}
}

func Songs(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		md := mdwn.MakeBuilder(len(p.Name) + 20)
		md.Concat("Top leaders for song **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.TopLeadersOfSong(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 20)
		md.Concat("Lessons for singing **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.SingingLessons(ctx, p.Name), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 30)
		md.Concat("Singing buddies for **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.SingingBuddies(ctx, p.Name, cmp.Or(p.Limit, 24)), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 30)
		md.Concat("Singing strangers for **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.SingingStrangers(ctx, p.Name, cmp.Or(p.Limit, 24)), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 30)
		md.Concat("Popular songs as observed by **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.PopularAsObserved(ctx, p.Name, cmp.Or(p.Limit, 25)), ec.Push)) {
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
		md := mdwn.MakeBuilder(256)
		md.Concat("Globally popular songs")
		if len(p.Years) > 0 {
			md.PushString(" (")
			for y := range irt.Slice(p.Years) {
				md.PushInt(y)
				md.PushString(", ")
			}
			md.PushString(")")
		} else {
			md.PushString(" (all time)")
		}
		md.PushString(":")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.GloballyPopularForYears(ctx, cmp.Or(p.Limit, 20), p.Years...), ec.Push)) {
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

		md := mdwn.MakeBuilder(len(p.Name) + 30)
		md.Concat("Locally popular songs (**", p.Name, "**):")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.LocallyPopular(ctx, cmp.Or(p.Limit, 20), localities...), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 40)
		md.Concat("Songs never sung at a singing **", p.Name, "** attended:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(irt.Limit2(conn.NeverSung(ctx, p.Name), cmp.Or(p.Limit, 20)), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 25)
		md.Concat("Songs **", p.Name, "** has never led:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(irt.Limit2(conn.NeverLed(ctx, p.Name, cmp.Or(p.Limit, 20)), cmp.Or(p.Limit, 20)), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 30)
		md.Concat("Unfamiliar hits for **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.TheUnfamilarHits(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
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
		md := mdwn.MakeBuilder(32)
		md.Concat("Leaders by connectedness:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.AllLeaderConnectedness(ctx, cmp.Or(p.Limit, 20)), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 25)
		md.Concat("Singing idols for **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.LeaderFootsteps(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
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
		md := mdwn.MakeBuilder(256)
		md.Concat("Top leaders")
		if len(p.Years) > 0 {
			md.PushString(" (")
			for y := range irt.Slice(p.Years) {
				md.PushInt(y)
				md.PushString(", ")
			}
			md.PushString(")")
		}
		md.PushString(":")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(irt.Convert(erc.HandleUntil(conn.TopLeadersByLeads(ctx, cmp.Or(p.Limit, 20), p.Years...), ec.Push), models.TopLeadersWrapper(&atomic.Int64{}))) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 20)
		md.Concat("Lead history for **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.LeaderLeadHistory(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 25)
		md.Concat("Singings attended by **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.LeaderSingingsAttended(ctx, p.Name, cmp.Or(p.Limit, 0)), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 25)
		md.Concat("Favorite keys for **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.LeaderFavoriteKey(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
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

		md := mdwn.MakeBuilder(256)
		md.Concat("Debut leaders for **")
		md.PushInt(year)
		md.PushString("**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(irt.Convert(erc.HandleUntil(conn.NewLeadersByYear(ctx, year, cmp.Or(p.Limit, 20)), ec.Push), models.WrapLeaderSongRank("Leads"))) {
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
		md := mdwn.MakeBuilder(256)
		md.Concat("Songs by key")
		if len(p.Years) > 0 {
			md.PushString(" (")
			for y := range irt.Slice(p.Years) {
				md.PushInt(y)
				md.PushString(", ")
			}
			md.PushString(")")
		} else {
			md.PushString(" (all time)")
		}
		md.PushString(":")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(irt.Convert(erc.HandleUntil(conn.SongsByKey(ctx, p.Years...), ec.Push), models.WrapSongByKey)) {
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
		md := mdwn.MakeBuilder(32)
		md.Concat("Leaders by top-20 leads:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(irt.Convert(erc.HandleUntil(conn.LeadersByTop20Leads(ctx, cmp.Or(p.Limit, 20)), ec.Push), models.WrapLeaderSongRank("Top-20 Leads"))) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 30)
		md.Concat("Singings per year for **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.LeaderSingingsPerYear(ctx, p.Name), ec.Push)) {
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
		md := mdwn.MakeBuilder(len(p.Name) + 25)
		md.Concat("Leaders in key **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(irt.Convert(
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
		md := mdwn.MakeBuilder(len(p.Name) + 30)
		md.Concat("Popular songs in key **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		var ec erc.Collector
		for md, err := range renderLineItems(erc.HandleUntil(conn.PopularSongsByKey(ctx, p.Name, cmp.Or(p.Limit, 20)), ec.Push)) {
			if !yield(md, err) {
				return
			}
		}
		if !ec.Ok() {
			yield(nil, ec.Resolve())
		}
	}
}
