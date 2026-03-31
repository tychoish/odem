package dispatch

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/fzfui"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/mcpsrv"
	"github.com/tychoish/odem/pkg/reportui"
)

type MinutesAppOperation int

type aliasMap struct {
	adt.SyncMap[string, MinutesAppOperation]
}

const (
	MinutesAppOpUnknown MinutesAppOperation = iota
	MinutesAppOpLeaderMostLed
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
	MinutesAppOpLeaderLeadHistory
	MinutesAppOpLeaderSingings
	MinutesAppOpInvalid
	MinutesAppOpExit = 181
)

var aliases aliasMap

func AllMinutesAppOps() iter.Seq[MinutesAppOperation] {
	return irt.Keep(irt.Convert(irt.Range(0, 181), toOp), isOk)
}

func AllMinutesAppAliases() iter.Seq2[MinutesAppOperation, []string] {
	return irt.With(AllMinutesAppOps(), getAliases)
}

func init() { aliases.populate(); aliases.addFallback() }

func NewMinutesAppOperation(arg string) MinutesAppOperation     { return aliases.Get(arg) }
func (mao MinutesAppOperation) GetInfo() irt.KV[string, string] { return mao.Registry().Info() }
func (mao MinutesAppOperation) ReportDispatcher() Reporter      { return mao.Registry().GetReporter() }
func (mao MinutesAppOperation) FuzzyDispatcher() FuzzHandler    { return mao.Registry().GetFuzzHandler() }
func (mao MinutesAppOperation) Aliases() []string               { return mao.Registry().Aliases }
func (mao MinutesAppOperation) String() string                  { return mao.GetInfo().Key }
func (mao MinutesAppOperation) Validate() error                 { return mao.Registry().err }
func (mao MinutesAppOperation) Ok() bool                        { return mao.isvalid() || mao == MinutesAppOpExit }
func (mao MinutesAppOperation) isvalid() bool                   { return mao > 0 && mao < MinutesAppOpInvalid }
func getAliases(mao MinutesAppOperation) []string               { return mao.Aliases() }
func (am *aliasMap) addFallback()                               { am.Store("", MinutesAppOpInvalid) }
func (am *aliasMap) populate()                                  { aliases.Extend(infra.ReverseMapping(AllMinutesAppAliases())) }

type MinutesAppRegistration struct {
	ID          MinutesAppOperation
	Command     string
	Description string
	Aliases     []string
	Reporter    Reporter
	Fuzz        FuzzHandler
	MCP         RegisterMCP
	err         error
}

func (reg MinutesAppRegistration) Ok() bool        { return reg.ID.Ok() }
func (reg MinutesAppRegistration) Validate() error { return reg.err }

func (reg MinutesAppRegistration) Info() irt.KV[string, string] {
	return irt.MakeKV(reg.Command, reg.Description)
}

func (reg MinutesAppRegistration) GetFuzzHandler() FuzzHandler {
	return func(c context.Context, d *db.Connection, a string) error {
		if reg.Fuzz == nil {
			return reg.err
		}
		return reg.Fuzz(c, d, a)
	}
}

func (reg MinutesAppRegistration) GetReporter() Reporter {
	return func(c context.Context, d *db.Connection, p reportui.Params) error {
		if reg.Reporter == nil {
			return reg.err
		}
		return reg.Reporter(c, d, p)
	}
}

func (mao MinutesAppOperation) Registry() MinutesAppRegistration {
	switch mao {
	case MinutesAppOpLeaderMostLed:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "most-led",
			Description: "return a list of all of the lessons a leader has given, and their frequence with information about the song (page, title, key).",
			Aliases:     []string{"leader-most-led", "leader-most-frequent", "most-led", "often-led"},
			Reporter:    reportui.Leader,
			Fuzz:        fzfui.LeaderAction,
			MCP:         mcpsrv.NewTool(mcpsrv.MostLeadSongs).Register,
		}
	case MinutesAppOpLeaderLeadHistory:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "leader-history",
			Description: "a list of all leads for a leader, with details about the song and the singing",
			Aliases:     []string{"leaders", "leader", "lead-history", "leader-history", "all-leads"},
			Reporter:    reportui.LeaderLeadHistory,
			Fuzz:        fzfui.LeaderLeadHistoryAction,
			MCP:         mcpsrv.NewTool(mcpsrv.LeaderLeadHistory).Register,
		}
	case MinutesAppOpLeaderSingings:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "leader-singings",
			Description: "a list of singings a leader attended, with their lead count, total leaders, and locality",
			Aliases:     []string{"leader-singings", "singings-attended", "attended"},
			Reporter:    reportui.LeaderSingings,
			Fuzz:        fzfui.LeaderSingingsAttendedAction,
			MCP:         mcpsrv.NewTool(mcpsrv.LeaderSingings).Register,
		}
	case MinutesAppOpSongs:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "songs",
			Description: "return basic information about a song, with a list of the leaders who have led the song the most.",
			Aliases:     []string{"song", "tune", "hymn", "songs"},
			Reporter:    reportui.Songs,
			Fuzz:        fzfui.SongAction,
			MCP:         mcpsrv.NewTool(mcpsrv.Songs).Register,
		}
	case MinutesAppOpSingings:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "singings",
			Description: "provide basic information about a specific singing, with a list of the leaders and the songs they led.",
			Aliases:     []string{"singing", "singings", "allday", "convention"},
			Reporter:    reportui.Singings,
			Fuzz:        SimpleFuzzyHandler(fzfui.SingingAction),
			MCP:         mcpsrv.NewTool(mcpsrv.Singings).Register,
		}
	case MinutesAppOpBuddies:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "leader-buddies",
			Description: "return a list of the singers most-frequent co-attenders of of singings for one singer.",
			Aliases:     []string{"buddies", "buddy", "connections", "neighbors", "leader-buddies"},
			Reporter:    reportui.Buddies,
			Fuzz:        fzfui.SingingBuddiesAction,
			MCP:         mcpsrv.NewTool(mcpsrv.Buddies).Register,
		}
	case MinutesAppOpStrangers:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "strangers",
			Description: "return a list of singers that the specified singer has never sung with, (but most of their buddies have!)",
			Aliases:     []string{"strangers", "enemies", "never-neighbors", "leader-strangers"},
			Reporter:    reportui.Strangers,
			Fuzz:        fzfui.SingingStrangersAction,
			MCP:         mcpsrv.NewTool(mcpsrv.Strangers).Register,
		}
	case MinutesAppOpPopularInOnesExperience:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "popular-in-ones-experience",
			Description: "a list of songs ordered by number of leads of all songs sung at singings thatone singer has attended.",
			Aliases:     []string{"prevalent", "popular-in-ones-experience"},
			Reporter:    reportui.PopularityAsExperienced,
			Fuzz:        fzfui.PopularInOnesExperienceAction,
			MCP:         mcpsrv.NewTool(mcpsrv.PopularInOnesExperience).Register,
		}
	case MinutesAppOpLocallyPopular:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "popular-locally",
			Description: "a list of songs ordered by number of leads at all singings in a particular region.",
			Aliases:     []string{"locally-popular", "localpop", "locally"},
			Reporter:    reportui.LocallyPopular,
			Fuzz:        fzfui.LocallyPopularAction,
			MCP:         mcpsrv.NewTool(mcpsrv.LocallyPopular).Register,
		}
	case MinutesAppOpPopularInYears:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "popular",
			Description: "a list of songs ordered by the number of leads at all singings in a particular year or years. Negative values remove that year's singings.",
			Aliases:     []string{"popular-for-years", "popular-in-years"},
			Reporter:    reportui.PopularityInYears,
			Fuzz:        fzfui.PopularInYearsAction,
			MCP:         mcpsrv.NewTool(mcpsrv.PopularInYears).Register,
		}
	case MinutesAppOpNeverSung:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "never-sung",
			Description: "a list of the songs that the specified singer has never **sung** at a minuted singing.",
			Aliases:     []string{"never-sung", "unknown"},
			Reporter:    reportui.NeverSung,
			Fuzz:        fzfui.NeverSungAction,
			MCP:         mcpsrv.NewTool(mcpsrv.NeverSung).Register,
		}
	case MinutesAppOpNeverLed:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "never-led",
			Description: "a list of songs that the specified singer has never **led** at a minuted singing.",
			Aliases:     []string{"never-led", "neverled", "unled"},
			Reporter:    reportui.NeverLed,
			Fuzz:        fzfui.NeverLedAction,
			MCP:         mcpsrv.NewTool(mcpsrv.NeverLed).Register,
		}
	case MinutesAppOpRetry:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "retry",
			Description: "(interactive) select an operation.",
			Aliases:     []string{"retry", "again", "restart", "repeat"},
			Reporter: func(ctx context.Context, conn *db.Connection, params reportui.Params) error {
				return fuzzySelectOperation(params.Name).ReportDispatcher().Report(ctx, conn, params)
			},
			Fuzz: func(ctx context.Context, conn *db.Connection, args string) error {
				return fuzzySelectOperation(args).FuzzyDispatcher().Handle(ctx, conn)
			},
		}
	case MinutesAppOpUnfamilarHits:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "unfamilar-hits",
			Description: "a list of the most popular songs that a singer has sung less often",
			Aliases:     []string{"unfamilar-hits", "unsung-hits", "unexpectedly-rare"},
			Reporter:    reportui.UnfamilarHits,
			Fuzz:        fzfui.UnfamilarHitsAction,
			MCP:         mcpsrv.NewTool(mcpsrv.UnfamilarHits).Register,
		}
	case MinutesAppOpConnectedness:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "connectedness",
			Description: "a list of singers, ordered by their connectedness ratio, or the percentge of the community they've sung with.",
			Aliases:     []string{"connectedness", "connected", "network"},
			Reporter:    reportui.Connectedness,
			Fuzz:        SimpleFuzzyHandler(fzfui.SingersByConnectednessAction),
			MCP:         mcpsrv.NewTool(mcpsrv.Connectedness).Register,
		}
	case MinutesAppOpLeaderFootsteps:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "leader-footsteps",
			Description: "a list of a leaders most frequently led songs, with that song's most frequently leader.",
			Aliases:     []string{"leader-footsteps", "footsteps", "giants", "singing-idols"},
			Reporter:    reportui.LeaderFootsteps,
			Fuzz:        fzfui.LeaderFootstepsAction,
			MCP:         mcpsrv.NewTool(mcpsrv.LeaderFootsteps).Register,
		}
	case MinutesAppOpTopLeaders:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "top-leaders",
			Description: "a list of all leaders ordered by their total number of minuted leads.",
			Aliases:     []string{"top-leaders", "leaderboard"},
			Reporter:    reportui.TopLeader,
			Fuzz:        fzfui.TopLeadersByLeadsAction,
			MCP:         mcpsrv.NewTool(mcpsrv.TopLeaders).Register,
		}
	case MinutesAppOpLeaderShare:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "leader-share",
			Description: "a list of all leaders ordered by their percentage of total leads optionally filtered by year",
			Aliases:     []string{"leader-share", "share", "leaders-share"},
			Reporter:    reportui.LeadershipShare,
			Fuzz:        fzfui.LeadersShareOfLeadsAction,
			MCP:         mcpsrv.NewTool(mcpsrv.LeaderShare).Register,
		}
	case MinutesAppOpExit:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "exit",
			Description: "exit <181>",
			Aliases:     []string{"exit", "return", "abort"},
			Reporter: func(ctx context.Context, conn *db.Connection, params reportui.Params) error {
				grip.Debugf("input-params", params)
				grip.Info("goodbye!")
				return nil
			},
			Fuzz: func(ctx context.Context, conn *db.Connection, args string) error {
				grip.Debugln("input-args", args)
				grip.Info("goodbye!")
				return nil
			},
		}
	case MinutesAppOpUnknown:
		return MinutesAppRegistration{ID: mao, err: ers.Error("unknown/undefined operation")}
	case MinutesAppOpInvalid:
		return MinutesAppRegistration{ID: mao, Aliases: []string{""}, err: ers.Error("invalid operation")}
	default:
		return MinutesAppRegistration{ID: mao, err: fmt.Errorf("undefined/invalid operation %s", mao)}
	}
}

func fuzzySelectOperation(arg string) MinutesAppOperation {
	// this needs to be in the dispatcher package to avoid a circular dependency, even though it
	// feels like it wants to be in the fzfui package.
	arg = strings.ReplaceAll(arg, " ", "-")
	grip.Debugln("selecting operation to dispatch", arg)

	operation := NewMinutesAppOperation(arg)

	if !operation.Ok() {
		var err error
		operation, err = infra.NewFuzzySearch[MinutesAppOperation](AllMinutesAppOps()).Prompt("odem operation").FindOne()
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
