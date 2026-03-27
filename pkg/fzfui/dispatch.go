package fzfui

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
)

type MinutesAppOperation int

func AllMinutesAppOperations() iter.Seq[MinutesAppOperation] {
	return irt.Keep(irt.Convert(irt.Range(int(MinutesAppOpUnknown), int(MinutesAppOpExit)), toOp), isOk)
}

func AllMinutesAppCommanders() iter.Seq[*cmdr.Commander] {
	return irt.Convert(AllMinutesAppOperations(), toCmdr)
}

const (
	MinutesAppOpUnknown MinutesAppOperation = iota
	MinutesAppOpLeaders
	MinutesAppOpSongs
	MinutesAppOpSingings
	MinutesAppOpBuddies
	MinutesAppOpStrangers
	MinutesAppOpPopularInOnesExperience
	MinutesAppOpPopularInYears
	MinutesAppOpLocallyPopular
	MinutesAppOpRetry
	MinutesAppOpNeverSung
	MinutesAppOpNeverLed
	MinutesAppOpUnfamilarHits
	MinutesAppOpConnectedness
	MinutesAppOpLeaderFootsteps
	MinutesAppOpInvalid
	MinutesAppOpExit = 181
)

func (mao MinutesAppOperation) GetInfo() irt.KV[string, string] {
	switch mao {
	case MinutesAppOpUnknown:
		return irt.MakeKV("unknown", "operation is not defined (zero)")
	case MinutesAppOpLeaders:
		return irt.MakeKV("leaders", "learn more about the leaders of a song")
	case MinutesAppOpSongs:
		return irt.MakeKV("songs", "learn more about a particular song and its most frequent leaders")
	case MinutesAppOpSingings:
		return irt.MakeKV("singings", "more info about a specific singing")
	case MinutesAppOpBuddies:
		return irt.MakeKV("buddies", "for a singer, find who their most-frequent co-attenders are")
	case MinutesAppOpStrangers:
		return irt.MakeKV("strangers", "for a singer, find out who they've never been at a singing with (but most of their buddies have!)")
	case MinutesAppOpPopularInOnesExperience:
		return irt.MakeKV("popular-in-ones-experience", "rank order the popularity of songs at the singings one singer has attended")
	case MinutesAppOpLocallyPopular:
		return irt.MakeKV("popular-locally", "rank order the popularity of songs in a given region or locality")
	case MinutesAppOpPopularInYears:
		return irt.MakeKV("popular", "the most popular song for a year (or negative, without that year)")
	case MinutesAppOpNeverSung:
		return irt.MakeKV("never-sung", "all songs a given singer has never sung (on the record).")
	case MinutesAppOpNeverLed:
		return irt.MakeKV("never-led", "all songs a given leader has never led (on the record)")
	case MinutesAppOpRetry:
		return irt.MakeKV("retry", "(restart) select an 'odem' application operation")
	case MinutesAppOpUnfamilarHits:
		return irt.MakeKV("unfamilar-hits", "otherwise popular songs which a singer has less exposure to")
	case MinutesAppOpConnectedness:
		return irt.MakeKV("connectedness", "rank all singers by their connectedness ratio (fraction of the community they've sung with)")
	case MinutesAppOpLeaderFootsteps:
		return irt.MakeKV("leader-footsteps", "for each song a singer has led, show the most frequent other leader of that song")
	case MinutesAppOpExit:
		return irt.MakeKV("exit", "181")
	case MinutesAppOpInvalid:
		return irt.MakeKV("invalid", fmt.Sprintf("invalid operation %d", mao))
	default:
		return irt.MakeKV("undefined", fmt.Sprint(mao))
	}
}

func (mao MinutesAppOperation) Dispatch() MinutesAppOperationHandler {
	return func(ctx context.Context, conn *db.Connection, args ...string) error {
		switch mao {
		case MinutesAppOpLeaders:
			return leaderAction(ctx, conn, args)
		case MinutesAppOpSongs:
			return songAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpSingings:
			return singingAction(ctx, conn)
		case MinutesAppOpBuddies:
			return singerBuddiesAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpStrangers:
			return singerStrangersAction(ctx, conn, "")
		case MinutesAppOpPopularInOnesExperience:
			return popularInOnesExperienceAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpNeverSung:
			return neverSungAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpNeverLed:
			return neverLedAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpLocallyPopular:
			return locallyPopularAction(ctx, conn, irt.Collect(irt.Convert(irt.Slice(args), models.NewSingingLocality))...)
		case MinutesAppOpPopularInYears:
			return popularInYearsAction(ctx, conn, strings.Join(args, ","))
		case MinutesAppOpUnfamilarHits:
			return unfamilarHitsAction(ctx, conn, strings.Join(args, ","))
		case MinutesAppOpConnectedness:
			return singersByConnectednessAction(ctx, conn)
		case MinutesAppOpLeaderFootsteps:
			return leaderFootstepsAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpExit:
			grip.Info("goodbye!")
			return nil
		case MinutesAppOpRetry:
			return selectMinutesAppAction(ctx, conn, strings.Join(args, "-"))
		case MinutesAppOpInvalid, MinutesAppOpUnknown:
			return ers.New("invalid/undefined operation")
		default:
			return fmt.Errorf("unknown operation at %d (%s)", mao, mao)
		}
	}
}

func NewMinutesAppOperation(arg string) MinutesAppOperation {
	switch arg {
	case "leaders", "leader", "singer", "person":
		return MinutesAppOpLeaders
	case "song", "tune", "hymn", "songs":
		return MinutesAppOpSongs
	case "singing", "allday", "convention":
		return MinutesAppOpSingings
	case "buddies", "connections", "neighbors":
		return MinutesAppOpBuddies
	case "strangers", "enemies", "never-neighbors":
		return MinutesAppOpStrangers
	case "exit", "return", "abort":
		return MinutesAppOpExit
	case "retry", "restart":
		return MinutesAppOpRetry
	case "prevalent", "popular-in-ones-experience":
		return MinutesAppOpPopularInOnesExperience
	case "never-sung", "unknown":
		return MinutesAppOpNeverSung
	case "never-led", "neverled":
		return MinutesAppOpNeverLed
	case "locally-popular", "localpop", "locally":
		return MinutesAppOpLocallyPopular
	case "unfamilar-hits", "unsung-hits":
		return MinutesAppOpUnfamilarHits
	case "connectedness", "connected", "network":
		return MinutesAppOpConnectedness
	case "popular-for-years", "popular-in-years":
		return MinutesAppOpPopularInYears
	case "leader-footsteps", "footsteps":
		return MinutesAppOpLeaderFootsteps
	default:
		return MinutesAppOpInvalid
	}
}

func (mao MinutesAppOperation) String() string { return mao.GetInfo().Key }

func (mao MinutesAppOperation) Validate() error {
	return ers.Whenf(!mao.Ok(), "invalid OperationID %s %d", mao, mao)
}

func (mao MinutesAppOperation) Ok() bool {
	return (mao > MinutesAppOpUnknown && mao < MinutesAppOpInvalid) || mao == MinutesAppOpExit
}

func (mao MinutesAppOperation) Commander() *cmdr.Commander {
	info := mao.GetInfo()
	return cmdr.MakeCommander().SetName(info.Key).SetUsage(info.Value).With(infra.DBOperationSpec(mao.Dispatch().Op).Add)
}

type MinutesAppOperationHandler func(context.Context, *db.Connection, ...string) error

func (maoh MinutesAppOperationHandler) Handle(ctx context.Context, conn *db.Connection, args ...string) error {
	return maoh(ctx, conn, args...)
}

func (maoh MinutesAppOperationHandler) Op(ctx context.Context, conn *db.Connection, args []string) error {
	return maoh(ctx, conn, args...)
}

func isOk[T interface{ Ok() bool }](in T) bool      { return in.Ok() }
func toOp(in int) MinutesAppOperation               { return MinutesAppOperation(in) }
func toCmdr(in MinutesAppOperation) *cmdr.Commander { return in.Commander() }
