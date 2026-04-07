package msgui

import (
	"context"
	"iter"

	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/reportui"
)

type Messenger func(context.Context, *db.Connection, reportui.Params) iter.Seq2[*strut.Mutable, error]

func MostLed(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func Songs(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func Singings(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func Buddies(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func Strangers(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func PopularAsObserved(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func PopularInYears(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func PopularLocally(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func NeverSung(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func NeverLed(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func UnfamilarHits(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func Connectedness(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func LeaderRoleModels(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func TopLeaders(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func LeaderShare(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func LeaderLeadHistory(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func LeaderSingings(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func LeaderFavoriteKey(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func LeaderDebutsByYear(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func SongsByKey(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func Top20Leaders(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func LeaderSingingsPerYear(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func LeadersByKey(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}

func PopularSongsByKey(ctx context.Context, conn *db.Connection, p reportui.Params) iter.Seq2[*strut.Mutable, error] {
	return irt.Two[*strut.Mutable, error](nil, nil)
}
