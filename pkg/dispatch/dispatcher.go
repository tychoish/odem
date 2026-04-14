package dispatch

import (
	"context"
	"fmt"
	"iter"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/fzfui"
	"github.com/tychoish/odem/pkg/mcpsrv"
	"github.com/tychoish/odem/pkg/msgui"
	"github.com/tychoish/odem/pkg/reportui"
)

type MinutesAppOperation int

const (
	MinutesAppOpUnknown MinutesAppOperation = iota
	MinutesAppOpLeaderMostLed
	MinutesAppOpLeaderFavoriteKey
	MinutesAppOpLeaderRoleModels
	MinutesAppOpLeaderUnfamilarHits
	MinutesAppOpLeaderBuddies
	MinutesAppOpLeaderStrangers
	MinutesAppOpLeaderNeverSung
	MinutesAppOpLeaderNeverLed
	MinutesAppOpLeaderLeadHistory
	MinutesAppOpLeaderSingings
	MinutesAppOpLeaderConnectedness
	MinutesAppOpLeaderShare
	MinutesAppOpSingings
	MinutesAppOpSongs
	MinutesAppOpSongsByKey
	MinutesAppOpPopularAsObserved
	MinutesAppOpPopularInYears
	MinutesAppOpPopularSongsByKey
	MinutesAppOpPopularLocally
	MinutesAppOpTop20Leaders
	MinutesAppOpTopLeaders
	MinutesAppOpTopLeadersByKey
	MinutesAppOpLeaderDebutes
	MinutesAppOpLeaderSingingsPerYear
	MinutesAppOpSongLyrics
	MinutesAppOpSongsByWord
	MinutesAppOpTop20LeadersActiveInLastYear
	MinutesAppOpInvalid
	MinutesAppOpRetry
	MinutesAppOpExit = 181
)

func AllMinutesAppOps() iter.Seq[MinutesAppOperation] {
	return irt.Keep(irt.Convert(irt.Range(0, 181), toOp), isOk)
}

func AllMinutesAppAliases() iter.Seq2[MinutesAppOperation, []string] {
	return irt.With(AllMinutesAppOps(), getAliases)
}

func MinutesAppAliasMapping() iter.Seq2[string, MinutesAppOperation] {
	return irt.ReverseMapping(AllMinutesAppAliases())
}

func AllMinutesAppCommands() iter.Seq2[string, MinutesAppOperation] {
	return irt.Flip(irt.With(AllMinutesAppOps(), toString))
}

func NewMinutesAppOperation(arg string) MinutesAppOperation     { return aliases.Get(arg) }
func (mao MinutesAppOperation) GetInfo() irt.KV[string, string] { return mao.Registry().Info() }
func (mao MinutesAppOperation) ReportDispatcher() Reporter      { return mao.Registry().GetReporter() }
func (mao MinutesAppOperation) FuzzyDispatcher() FuzzHandler    { return mao.Registry().GetFuzzHandler() }
func (mao MinutesAppOperation) Aliases() []string               { return mao.Registry().Aliases }
func (mao MinutesAppOperation) String() string                  { return mao.GetInfo().Key }
func (mao MinutesAppOperation) Validate() error                 { return mao.Registry().err }
func (mao MinutesAppOperation) Ok() bool                        { return mao.isvalid() || mao == MinutesAppOpExit }
func (mao MinutesAppOperation) isvalid() bool                   { return mao > 0 && mao < MinutesAppOpInvalid }

func (mao MinutesAppOperation) Registry() MinutesAppRegistration {
	switch mao {
	case MinutesAppOpLeaderMostLed:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "most-led",
			Description: "return a list of all of the lessons a leader has given, and their frequence with information about the song (page, title, key).",
			Aliases: []string{
				"leader-most-led", "leader-most-frequent", "most-led", "often-led",
			},
			Reporter:  reportui.MostLed,
			Fuzz:      fzfui.LeaderAction,
			MCP:       mcpsrv.NewTool(mcpsrv.MostLeadSongs).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
			Messenger: msgui.MostLed,
		}
	case MinutesAppOpLeaderLeadHistory:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "lead-history",
			Description: "a list of all leads for a leader, with details about the song and the singing",
			Aliases:     []string{"leaders", "leader", "lead-history", "leader-history", "all-leads", "leader history"},
			Reporter:    reportui.LeaderLeadHistory,
			Fuzz:        fzfui.LeaderLeadHistoryAction,
			MCP:         mcpsrv.NewTool(mcpsrv.LeaderLeadHistory).Register,
			Messenger:   msgui.LeaderLeadHistory,
			Requires:    dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
		}
	case MinutesAppOpLeaderSingings:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "leader-singings",
			Description: "a list of singings a leader attended, with their lead count, total leaders, and locality",
			Aliases: []string{
				"leader-singings", "singings-attended", "attended", "attended-singings",
			},
			Reporter:  reportui.LeaderSingings,
			Fuzz:      fzfui.LeaderSingingsAttendedAction,
			MCP:       mcpsrv.NewTool(mcpsrv.LeaderSingings).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
			Messenger: msgui.LeaderSingings,
		}
	case MinutesAppOpSongs:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "songs",
			Description: "return basic information about a song, with a list of the leaders who have led the song the most.",
			Aliases: []string{
				"song", "tune", "hymn", "songs", "page",
			},
			Reporter:  reportui.Songs,
			Fuzz:      fzfui.SongAction,
			MCP:       mcpsrv.NewTool(mcpsrv.Songs).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeSong)),
			Messenger: msgui.Songs,
		}
	case MinutesAppOpSingings:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "singings",
			Description: "provide basic information about a specific singing, with a list of the leaders and the songs they led.",
			Aliases: []string{
				"singing", "singings", "allday", "convention", "all-days", "conventions", "all-day", "alldays",
			},
			Reporter:  reportui.Singings,
			Fuzz:      SimpleFuzzyHandler(fzfui.SingingAction),
			MCP:       mcpsrv.NewTool(mcpsrv.Singings).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeSinging)),
			Messenger: msgui.Singings,
		}
	case MinutesAppOpLeaderBuddies:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "buddies",
			Description: "return a list of the singers most-frequent co-attenders of of singings for one singer.",
			Aliases: []string{
				"buddies", "buddy", "connections", "neighbors", "leader-buddies", "singing-buddies",
			},
			Reporter:  reportui.Buddies,
			Fuzz:      fzfui.SingingBuddiesAction,
			MCP:       mcpsrv.NewTool(mcpsrv.Buddies).Register,
			Messenger: msgui.Buddies,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
		}
	case MinutesAppOpLeaderStrangers:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "strangers",
			Description: "return a list of singers that the specified singer has never sung with, (but most of their buddies have!)",
			Aliases: []string{
				"strangers", "enemies", "enemy", "stranger",
				"anti-matter-twin", "alter-twin", "never-neighbors", "leader-strangers", "singing-strangers", "singing-stranger",
			},
			Reporter:  reportui.Strangers,
			Fuzz:      fzfui.SingingStrangersAction,
			MCP:       mcpsrv.NewTool(mcpsrv.Strangers).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
			Messenger: msgui.Strangers,
		}
	case MinutesAppOpPopularAsObserved:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "popular-as-observed",
			Description: "a list of songs ordered by number of leads of all songs sung at singings thatone singer has attended.",
			Aliases: []string{
				"prevalent",
				"popular-for-me", "popular-for-them", "poplar-for-who",
				"popular-in-ones-experience", "as-observed",
			},
			Reporter:  reportui.PopularityAsExperienced,
			Fuzz:      fzfui.PopularInOnesExperienceAction,
			MCP:       mcpsrv.NewTool(mcpsrv.PopularInOnesExperience).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
			Messenger: msgui.PopularAsObserved,
		}
	case MinutesAppOpPopularLocally:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "popular-locally",
			Description: "a list of songs ordered by number of leads at all singings in a particular region.",
			Aliases: []string{
				"locally-popular", "localpop", "locally", "popular-where",
				"popular-where", "popular-locally", "local-fave", "local-favorite",
			},
			Reporter:  reportui.LocallyPopular,
			Fuzz:      fzfui.LocallyPopularAction,
			MCP:       mcpsrv.NewTool(mcpsrv.LocallyPopular).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLocality)),
			Messenger: msgui.PopularLocally,
		}
	case MinutesAppOpPopularInYears:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "popular",
			Description: "a list of songs ordered by the number of leads at all singings in a particular year or years. Negative values remove that year's singings.",
			Aliases: []string{
				"popular-for-years", "popular-in-years", "popular-when", "popular-in-years", "popular-for-years", "when",
			},
			Reporter:  reportui.PopularityInYears,
			Fuzz:      fzfui.PopularInYearsAction,
			MCP:       mcpsrv.NewTool(mcpsrv.PopularInYears).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeYear)),
			Messenger: msgui.PopularInYears,
		}
	case MinutesAppOpLeaderNeverSung:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "never-sung",
			Description: "a list of the songs that the specified singer has never **sung** at a minuted singing.",
			Aliases: []string{
				"unknown", "missed", "miss", "unsung",
				"never-sung", "never-sung",
			},
			Reporter:  reportui.NeverSung,
			Fuzz:      fzfui.NeverSungAction,
			MCP:       mcpsrv.NewTool(mcpsrv.NeverSung).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
			Messenger: msgui.NeverSung,
		}
	case MinutesAppOpLeaderNeverLed:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "never-led",
			Description: "a list of songs that the specified singer has never **led** at a minuted singing.",
			Aliases: []string{
				"unled",
				"never-led", "neverled", "not-led",
			},
			Reporter:  reportui.NeverLed,
			Fuzz:      fzfui.NeverLedAction,
			MCP:       mcpsrv.NewTool(mcpsrv.NeverLed).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
			Messenger: msgui.NeverLed,
		}
	case MinutesAppOpRetry:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "retry",
			Description: "(interactive) select an operation.",
			Aliases:     []string{"retry", "again", "restart", "repeat", "once more", "start over", "start-over"},
			Reporter: func(ctx context.Context, conn *db.Connection, params reportui.Params) error {
				return fuzzySelectOperation(params.Name).ReportDispatcher().Report(ctx, conn, params)
			},
			Fuzz: func(ctx context.Context, conn *db.Connection, args string) error {
				return fuzzySelectOperation(args).FuzzyDispatcher().Handle(ctx, conn)
			},
			Requires: dt.MakeSet(irt.Args(MinutesAppQueryTypeOperation)),
		}
	case MinutesAppOpLeaderUnfamilarHits:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "unfamilar-hits",
			Description: "a list of the most popular songs that a singer has sung less often",
			Aliases: []string{
				"rare",
				"unfamiliar-hits", "unfamiliar-hit", "unsung-hits",
				"unexpectedly-rare", "popular elsewhere", "popular-elsewhere",
			},
			Reporter:  reportui.UnfamilarHits,
			Fuzz:      fzfui.UnfamilarHitsAction,
			MCP:       mcpsrv.NewTool(mcpsrv.UnfamilarHits).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
			Messenger: msgui.UnfamilarHits,
		}
	case MinutesAppOpLeaderConnectedness:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "connectedness",
			Description: "a list of singers, ordered by their connectedness ratio, or the percentge of the community they've sung with.",
			Aliases: []string{
				"connectedness", "connected", "network", "networked",
				"leader-connectedness", "connected-leader",
			},
			Reporter:  reportui.Connectedness,
			Fuzz:      SimpleFuzzyHandler(fzfui.SingersByConnectednessAction),
			MCP:       mcpsrv.NewTool(mcpsrv.Connectedness).Register,
			Requires:  &dt.Set[MinutesAppQueryType]{},
			Messenger: msgui.Connectedness,
		}
	case MinutesAppOpLeaderRoleModels:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "leader-role-models",
			Description: "a list of a leaders most frequently led songs, with that song's most frequently leader.",
			Aliases: []string{
				"leader-footsteps", "footsteps", "giants", "singing-idols", "idols",
				"role model", "role models", "role-models", "rolemodels", "idol",
			},
			Reporter:  reportui.LeaderFootsteps,
			Fuzz:      fzfui.LeaderFootstepsAction,
			MCP:       mcpsrv.NewTool(mcpsrv.LeaderFootsteps).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
			Messenger: msgui.LeaderRoleModels,
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
			Requires:    dt.MakeSet(irt.Args(MinutesAppQueryTypeYear)),
			Messenger:   msgui.TopLeaders,
		}
	case MinutesAppOpLeaderFavoriteKey:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "leader-favorite-key",
			Description: "a list of musical keys ordered by the number of leads a given leader has given in each key",
			Aliases: []string{
				"favorite-key", "leader-key", "keys-led", "favorite-keys",
				"keys-for-songs-led", "keys-for-leads",
			},
			Reporter:  reportui.LeaderFavoriteKey,
			Fuzz:      fzfui.LeaderFavoriteKeyAction,
			MCP:       mcpsrv.NewTool(mcpsrv.LeaderFavoriteKey).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader)),
			Messenger: msgui.LeaderFavoriteKey,
		}
	case MinutesAppOpLeaderDebutes:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "debuts",
			Description: "leaders making their debut in a given year, by lead count",
			Aliases: []string{
				"debut",
				"new-leader", "debut-year",
				"new-leaders", "first-timers", "leader-debuts",
			},
			Reporter:  reportui.NewLeadersByYear,
			Fuzz:      fzfui.NewLeadersByYearAction,
			MCP:       mcpsrv.NewTool(mcpsrv.NewLeadersByYear).Register,
			Messenger: msgui.LeaderDebutsByYear,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeYear)),
		}
	case MinutesAppOpSongsByKey:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "songs-by-key",
			Description: "frequency of song keys in the minutes, with percentage of total",
			Aliases:     []string{"songs-by-key", "keys", "key-stats", "key stats", "songs by key"},
			Reporter:    reportui.SongsByKey,
			Fuzz:        fzfui.SongsByKeyAction,
			MCP:         mcpsrv.NewTool(mcpsrv.SongsByKey).Register,
			Requires:    dt.MakeSet(irt.Args(MinutesAppQueryTypeYear)),
			Messenger:   msgui.SongsByKey,
		}
	case MinutesAppOpLeaderShare:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "leader-share",
			Description: "a list of all leaders ordered by their percentage of total leads optionally filtered by year",
			Aliases: []string{
				"share",
				"leader-share", "leaders-share", "share-by-leader",
			},
			Reporter:  reportui.LeadershipShare,
			Fuzz:      fzfui.LeadersShareOfLeadsAction,
			MCP:       mcpsrv.NewTool(mcpsrv.LeaderShare).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeLeader, MinutesAppQueryTypeYear)),
			Messenger: msgui.LeaderShare,
		}
	case MinutesAppOpTop20Leaders:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "top20-leaders",
			Description: "leaders ordered by number of songs for which they are the top-20 leads",
			Aliases: []string{
				"top-twenty",
				"top20-leaders", "top20", "top-20-leaders", "top-20",
			},
			Reporter:  reportui.LeadersByTop20Leads,
			Fuzz:      fzfui.LeadersByTop20LeadsAction,
			MCP:       mcpsrv.NewTool(mcpsrv.LeadersByTop20Leads).Register,
			Messenger: msgui.Top20Leaders,
		}
	case MinutesAppOpLeaderSingingsPerYear:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "singings-per-year",
			Description: "number of singings a leader attended per year",
			Aliases: []string{
				"singings-per-year", "yearly-singings", "annual-singings", "yearly-singings", "annual-singings",
			},
			Reporter:  reportui.LeaderSingingsPerYear,
			Fuzz:      fzfui.LeaderSingingsPerYearAction,
			MCP:       mcpsrv.NewTool(mcpsrv.LeaderSingingsPerYear).Register,
			Messenger: msgui.LeaderSingingsPerYear,
		}
	case MinutesAppOpTopLeadersByKey:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "leader-by-key",
			Description: "leaders ordered by number of leads in a given key",
			Aliases: []string{
				"leaders-in-key", "key-leaders", "leads-in-key", "key-leads",
			},
			Reporter:  reportui.LeadersByKey,
			Fuzz:      fzfui.LeadersByKeyAction,
			MCP:       mcpsrv.NewTool(mcpsrv.LeadersByKey).Register,
			Requires:  dt.MakeSet(irt.Args(MinutesAppQueryTypeKey)),
			Messenger: msgui.LeadersByKey,
		}
	case MinutesAppOpSongsByWord:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "songs-by-word",
			Description: "search song lyrics for a word or phrase; returns page number, title, and the matching line.",
			Aliases:     []string{"word", "find-word", "lyrics-search", "search-lyrics", "by-word", "contains"},
			Reporter:    reportui.SongsByWord,
			Fuzz:        fzfui.SongsByWordAction,
			MCP:         mcpsrv.NewTool(mcpsrv.SongsByWord).Register,
			Requires:    dt.MakeSet(irt.Args(MinutesAppQueryTypeWord)),
			Messenger:   msgui.SongsByWord,
		}
	case MinutesAppOpSongLyrics:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "lyrics",
			Description: "render the full text/lyrics of a song with its page number, title, music author, words author, meter, and key.",
			Aliases:     []string{"lyrics", "song-lyrics", "tune-lyrics", "words", "text"},
			Reporter:    reportui.SongLyrics,
			Fuzz:        fzfui.SongLyricsAction,
			MCP:         mcpsrv.NewTool(mcpsrv.SongLyrics).Register,
			Requires:    dt.MakeSet(irt.Args(MinutesAppQueryTypeSong)),
			Messenger:   msgui.SongLyrics,
		}
	case MinutesAppOpPopularSongsByKey:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "songs-in-key",
			Description: "most frequently led songs in a given key",
			Aliases: []string{
				"popular-in-key", "key-songs", "songs-in-key", "songs-for-key", "songs-in-the-key-of",
			},
			Reporter:  reportui.PopularSongsByKey,
			Fuzz:      fzfui.PopularSongsByKeyAction,
			MCP:       mcpsrv.NewTool(mcpsrv.PopularSongsByKey).Register,
			Messenger: msgui.PopularSongsByKey,
		}
	case MinutesAppOpTop20LeadersActiveInLastYear:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "top20-active",
			Description: "leaders ordered by number of top-20 songs who have led at least once in the last year",
			Aliases: []string{
				"top20-active", "top-twenty-active", "top20-last-year", "active-top20",
				"recent-top20", "top-20-active",
			},
			Reporter:  reportui.Top20LeadersActiveInLastYear,
			Fuzz:      fzfui.Top20LeadersActiveInLastYearAction,
			MCP:       mcpsrv.NewTool(mcpsrv.Top20LeadersActiveInLastYear).Register,
			Messenger: msgui.Top20LeadersActiveInLastYear,
			Requires:  &dt.Set[MinutesAppQueryType]{},
		}
	case MinutesAppOpExit:
		return MinutesAppRegistration{
			ID:          mao,
			Command:     "exit",
			Description: "exit <181>",
			Aliases:     []string{"exit", "return", "abort"},
			Reporter: func(ctx context.Context, conn *db.Connection, params reportui.Params) error {
				grip.Debug(grip.MPrintf("input-params", params))
				grip.Info("goodbye!")
				return nil
			},
			Fuzz: func(ctx context.Context, conn *db.Connection, args string) error {
				grip.Debug(grip.MPrintln("input-args", args))
				grip.Info("goodbye!")
				return nil
			},
			Requires: &dt.Set[MinutesAppQueryType]{},
		}
	case MinutesAppOpUnknown:
		return MinutesAppRegistration{ID: mao, err: ers.Error("unknown/undefined operation")}
	case MinutesAppOpInvalid:
		return MinutesAppRegistration{ID: mao, Aliases: []string{""}, err: ers.Error("invalid operation")}
	default:
		return MinutesAppRegistration{ID: mao, err: fmt.Errorf("undefined/invalid operation %s", mao)}
	}
}
