package msgui

import (
	"context"
	"iter"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/reportui"
)

type Messenger func(context.Context, *db.Connection, models.Params) iter.Seq2[*mdwn.Builder, error]

// return func(yield func(*mdwn.Builder, error) bool) {
// }

func MostLed(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		md := mdwn.MakeBuilder(len(p.Name) + 20)
		md.Concat("Most led songs for **", p.Name, "**:")
		if !yield(md, nil) {
			return
		}

		mdtb := mdwn.MakeBuilder(4096)
		var ec erc.Collector
		reportui.WriteSongTable(mdtb, erc.HandleUntil(conn.MostLedSongs(ctx, p.Name, p.Limit), ec.Push))
		if !ec.Ok() {
			yield(nil, ec.Resolve())
			return
		}

		flush(mdtb, yield)
	}
}

func Songs(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func Singings(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func Buddies(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func Strangers(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func PopularAsObserved(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func PopularInYears(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func PopularLocally(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func NeverSung(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func NeverLed(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func UnfamilarHits(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func Connectedness(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func LeaderRoleModels(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func TopLeaders(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func LeaderShare(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func LeaderLeadHistory(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func LeaderSingings(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func LeaderFavoriteKey(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func LeaderDebutsByYear(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func SongsByKey(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func Top20Leaders(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func LeaderSingingsPerYear(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func LeadersByKey(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}

func PopularSongsByKey(ctx context.Context, conn *db.Connection, p models.Params) iter.Seq2[*mdwn.Builder, error] {
	return irt.Two[*mdwn.Builder, error](nil, nil)
}
