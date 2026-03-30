package dispatch

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/fzfui"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/reportui"
)

type MinutesAppOperation int

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
	MinutesAppOpTopLeaders
	MinutesAppOpLeaderShare
	MinutesAppOpInvalid
	MinutesAppOpExit = 181
)

func AllMinutesAppOperations() iter.Seq[MinutesAppOperation] {
	return irt.Keep(irt.Convert(irt.Range(int(MinutesAppOpUnknown), int(MinutesAppOpExit)), toOp), isOk)
}

func NewMinutesAppOperation(arg string) MinutesAppOperation {
	switch arg {
	case "leaders", "leader", "singer", "person":
		return MinutesAppOpLeaders
	case "song", "tune", "hymn", "songs":
		return MinutesAppOpSongs
	case "singing", "singings", "allday", "convention":
		return MinutesAppOpSingings
	case "buddies", "buddy", "connections", "neighbors":
		return MinutesAppOpBuddies
	case "strangers", "enemies", "never-neighbors":
		return MinutesAppOpStrangers
	case "exit", "return", "abort":
		return MinutesAppOpExit
	case "retry", "again", "restart", "repeat":
		return MinutesAppOpRetry
	case "prevalent", "popular-in-ones-experience":
		return MinutesAppOpPopularInOnesExperience
	case "never-sung", "unknown":
		return MinutesAppOpNeverSung
	case "never-led", "neverled", "unled":
		return MinutesAppOpNeverLed
	case "locally-popular", "localpop", "locally":
		return MinutesAppOpLocallyPopular
	case "unfamilar-hits", "unsung-hits", "unexpectedly-rare":
		return MinutesAppOpUnfamilarHits
	case "connectedness", "connected", "network":
		return MinutesAppOpConnectedness
	case "popular-for-years", "popular-in-years":
		return MinutesAppOpPopularInYears
	case "leader-footsteps", "footsteps", "giants", "singing-idols":
		return MinutesAppOpLeaderFootsteps
	case "top-leaders", "leaderboard":
		return MinutesAppOpTopLeaders
	case "leader-share", "share", "leaders-share":
		return MinutesAppOpLeaderShare
	default:
		return MinutesAppOpInvalid
	}
}

func (mao MinutesAppOperation) String() string { return mao.GetInfo().Key }
func (mao MinutesAppOperation) Validate() error {
	return ers.Whenf(!mao.Ok(), "invalid OperationID %s %d", mao, mao)
}

func (mao MinutesAppOperation) Ok() bool { return mao.isValidOp() || mao == MinutesAppOpExit }
func (mao MinutesAppOperation) isValidOp() bool {
	return mao > MinutesAppOpUnknown && mao < MinutesAppOpInvalid
}

func (mao MinutesAppOperation) GetInfo() irt.KV[string, string] {
	// response maps the subcommand (key) to the usage text.
	switch mao {
	case MinutesAppOpLeaders:
		return irt.MakeKV("leaders", "return a list of all of the lessons a leader has given, and their frequence with information about the song (page, title, key).")
	case MinutesAppOpSongs:
		return irt.MakeKV("songs", "return basic information about a song, with a list of the leaders who have led the song the most.")
	case MinutesAppOpSingings:
		return irt.MakeKV("singings", "provide basic information about a specific singing, with a list of the leaders and the songs they led.")
	case MinutesAppOpBuddies:
		return irt.MakeKV("buddies", "return a list of the singers most-frequent co-attenders of of singings for one singer.")
	case MinutesAppOpStrangers:
		return irt.MakeKV("strangers", "return a list of singers that the specified singer has never sung with, (but most of their buddies have!)")
	case MinutesAppOpPopularInOnesExperience:
		return irt.MakeKV("popular-in-ones-experience", "a list of songs ordered by number of leads of all songs sung at singings thatone singer has attended.")
	case MinutesAppOpLocallyPopular:
		return irt.MakeKV("popular-locally", "a list of songs ordered by number of leads at all singings in a particular region.")
	case MinutesAppOpPopularInYears:
		return irt.MakeKV("popular", "a list of songs ordered by the number of leads at all singings in a particular year or years. Negative values remove that year's singings.")
	case MinutesAppOpNeverSung:
		return irt.MakeKV("never-sung", "a list of the songs that the specified singer has never **sung** at a minuted singing.")
	case MinutesAppOpNeverLed:
		return irt.MakeKV("never-led", "a list of songs that the specified singer has never **led** at a minuted singing.")
	case MinutesAppOpRetry:
		return irt.MakeKV("retry", "(interactive) select an operation.")
	case MinutesAppOpUnfamilarHits:
		return irt.MakeKV("unfamilar-hits", "a list of the most popular songs that a singer has sung less often")
	case MinutesAppOpConnectedness:
		return irt.MakeKV("connectedness", "a list of singers, ordered by their connectedness ratio, or the percentge of the community they've sung with.")
	case MinutesAppOpLeaderFootsteps:
		return irt.MakeKV("leader-footsteps", "a list of a leaders most frequently led songs, with that song's most frequently leader.")
	case MinutesAppOpTopLeaders:
		return irt.MakeKV("top-leaders", "a list of all leaders ordered by their total number of minuted leads.")
	case MinutesAppOpLeaderShare:
		return irt.MakeKV("leader-share", "a list of all leaders ordered by their percentage of total leads optionally filtered by year")
	case MinutesAppOpExit:
		return irt.MakeKV("exit", "181")
	case MinutesAppOpUnknown:
		return irt.MakeKV("unknown", "operation is not defined (zero)")
	case MinutesAppOpInvalid:
		return irt.MakeKV("invalid", fmt.Sprintf("invalid operation %d", mao))
	default:
		return irt.MakeKV("undefined", fmt.Sprint(mao))
	}
}

func (mao MinutesAppOperation) FuzzyDispatcher() MinutesAppOperationHandler {
	return func(ctx context.Context, conn *db.Connection, args ...string) error {
		switch mao {
		case MinutesAppOpLeaders:
			return fzfui.LeaderAction(ctx, conn, args)
		case MinutesAppOpSongs:
			return fzfui.SongAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpSingings:
			return fzfui.SingingAction(ctx, conn)
		case MinutesAppOpBuddies:
			return fzfui.SingingBuddiesAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpStrangers:
			return fzfui.SingingStrangersAction(ctx, conn, "")
		case MinutesAppOpPopularInOnesExperience:
			return fzfui.PopularInOnesExperienceAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpNeverSung:
			return fzfui.NeverSungAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpNeverLed:
			return fzfui.NeverLedAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpLocallyPopular:
			return fzfui.LocallyPopularAction(ctx, conn, irt.Collect(irt.Convert(irt.Slice(args), models.NewSingingLocality))...)
		case MinutesAppOpPopularInYears:
			return fzfui.PopularInYearsAction(ctx, conn, strings.Join(args, ","))
		case MinutesAppOpUnfamilarHits:
			return fzfui.UnfamilarHitsAction(ctx, conn, strings.Join(args, ","))
		case MinutesAppOpConnectedness:
			return fzfui.SingersByConnectednessAction(ctx, conn)
		case MinutesAppOpLeaderFootsteps:
			return fzfui.LeaderFootstepsAction(ctx, conn, strings.Join(args, " "))
		case MinutesAppOpTopLeaders:
			return fzfui.TopLeadersByLeadsAction(ctx, conn, strings.Join(args, ","))
		case MinutesAppOpLeaderShare:
			return fzfui.LeadersShareOfLeadsAction(ctx, conn, strings.Join(args, ","))
		case MinutesAppOpRetry:
			return fuzzySelectOperation(strings.Join(args, "-")).FuzzyDispatcher().Handle(ctx, conn, args...)
		case MinutesAppOpExit:
			grip.Info("goodbye!")
			return nil
		case MinutesAppOpInvalid:
			return ers.New("explicitly invalid operation")
		case MinutesAppOpUnknown:
			return ers.New("unknown operation")
		default:
			return fmt.Errorf("undefined operation at %d (%s)", mao, mao)
		}
	}
}

func (mao MinutesAppOperation) ReportDispatcher() Reporter {
	return func(ctx context.Context, conn *db.Connection, params reportui.Params) error {
		switch mao {
		case MinutesAppOpLeaders:
			return reportui.Leader(ctx, conn, params)
		case MinutesAppOpSongs:
			return reportui.Songs(ctx, conn, params)
		case MinutesAppOpSingings:
			return reportui.Singings(ctx, conn, params)
		case MinutesAppOpBuddies:
			return reportui.Buddies(ctx, conn, params)
		case MinutesAppOpStrangers:
			return reportui.Strangers(ctx, conn, params)
		case MinutesAppOpPopularInOnesExperience:
			return reportui.PopularityAsExperienced(ctx, conn, params)
		case MinutesAppOpPopularInYears:
			return reportui.PopularityInYears(ctx, conn, params)
		case MinutesAppOpLocallyPopular:
			return reportui.LocallyPopular(ctx, conn, params)
		case MinutesAppOpNeverSung:
			return reportui.NeverSung(ctx, conn, params)
		case MinutesAppOpNeverLed:
			return reportui.NeverLed(ctx, conn, params)
		case MinutesAppOpUnfamilarHits:
			return reportui.UnfamilarHits(ctx, conn, params)
		case MinutesAppOpConnectedness:
			return reportui.Connectedness(ctx, conn, params)
		case MinutesAppOpTopLeaders:
			return reportui.TopLeader(ctx, conn, params)
		case MinutesAppOpLeaderShare:
			return reportui.LeadershipShare(ctx, conn, params)
		case MinutesAppOpLeaderFootsteps:
			return reportui.LeaderFootsteps(ctx, conn, params)
		case MinutesAppOpRetry:
			return fuzzySelectOperation(params.Name).ReportDispatcher().Report(ctx, conn, params)
		case MinutesAppOpExit:
			grip.Info("goodbye!")
			return nil
		case MinutesAppOpUnknown:
			return ers.New("unknown operation")
		case MinutesAppOpInvalid:
			return ers.New("explicitly invalid operation")
		default:
			return fmt.Errorf("undefinedoperation at %d (%s)", mao, mao)
		}
	}
}

func fuzzySelectOperation(arg string) MinutesAppOperation {
	// this needs to be in the dispatcher package to avoid a circular dependency, even though it
	// feels like it wants to be in the fzfui package.
	grip.Debugln("selecting operation to dispatch", arg)

	operation := NewMinutesAppOperation(arg)
	if !operation.Ok() {
		var err error
		operation, err = infra.NewFuzzySearch[MinutesAppOperation](AllMinutesAppOperations()).Prompt("odem operation").FindOne()
		if operation.Ok() {
			return operation
		}
		if newop := NewMinutesAppOperation(operation.String()); newop.Ok() {
			grip.Debugln("succeeded to identify %s on fallback", newop)
			return newop
		}

		if err != nil {
			grip.Warningf("operation %q is not valid, %v, retrying", operation.String(), err)
			return MinutesAppOpRetry
		}
	}

	grip.Debugln("selected", operation)
	return operation
}
