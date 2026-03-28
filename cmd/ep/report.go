package ep

import (
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tychoish/cmdr"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/fzfui"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/mdwn"
	"github.com/tychoish/odem/pkg/models"
	"github.com/urfave/cli/v3"
)

type reportInput struct {
	db     *db.Connection
	singer string
	stdout bool
}

func reportOperationSpec() *cmdr.OperationSpec[*reportInput] {
	// TODO should be able to use the infra.DBoperation spec and the `WithInput[T]` type wihtout (potentially having a very thin wrapper around that to read the name+stduout flag)
	return cmdr.SpecBuilder(
		func(ctx context.Context, cc *cli.Command) (*reportInput, error) {
			conn, err := db.Connect(ctx)
			if err != nil {
				return nil, err
			}
			return &reportInput{
				db:     conn,
				singer: cmdr.GetFlagOrFirstArg[string](cc, "name"),
				stdout: cmdr.GetFlag[bool](cc, "stdout"), 
			}, nil
		},
	).SetAction(func(ctx context.Context, in *reportInput) error {
		return reportAction(ctx, in.db, in.singer, in.stdout)
	})
}

// THe subcommand and the fullcommand should be factorable into a single implementation once the arguents are sorted out. 
func reportSubcmd(op fzfui.MinutesAppOperation) *cmdr.Commander {
	info := op.GetInfo()
	return cmdr.MakeCommander().
		SetName(info.Key).
		SetUsage(info.Value).
		With(infra.DBOperationSpec(func(ctx context.Context, conn *db.Connection, arg string) error {
			reporter := dispatcher(op)
			if reporter == nil {
				return nil
			}
			return reporter(ctx, conn, os.Stdout, arg)
		}).Add)
}

func reportSubcmdFull() *cmdr.Commander {
	info := fzfui.MinutesAppOpLeaders.GetInfo()
	return cmdr.MakeCommander().
		SetName(info.Key).
		SetUsage(info.Value).
		With(reportOperationSpec().Add)
}

func Report() *cmdr.Commander {
	return cmdr.MakeCommander().
		SetName("report").
		Aliases("rpt").
		SetUsage("generate a markdown report for a singer").
		Flags(cmdr.FlagBuilder[bool](false).
			SetName("stdout", "o").
			SetUsage("write report to stdout instead of a file").
			Flag()).
		With(reportOperationSpec().Add).
		Subcommanders(
			// TODO derive this by being able to iterate through the names/ids of the
			// MinutesAppOperation enum values
			reportSubcmdFull(),
			reportSubcmd(fzfui.MinutesAppOpSongs),
			reportSubcmd(fzfui.MinutesAppOpSingings),
			reportSubcmd(fzfui.MinutesAppOpBuddies),
			reportSubcmd(fzfui.MinutesAppOpStrangers),
			reportSubcmd(fzfui.MinutesAppOpPopularInOnesExperience),
			reportSubcmd(fzfui.MinutesAppOpPopularInYears),
			reportSubcmd(fzfui.MinutesAppOpLocallyPopular),
			reportSubcmd(fzfui.MinutesAppOpNeverSung),
			reportSubcmd(fzfui.MinutesAppOpNeverLed),
			reportSubcmd(fzfui.MinutesAppOpUnfamilarHits),
			reportSubcmd(fzfui.MinutesAppOpConnectedness),
			reportSubcmd(fzfui.MinutesAppOpTopLeaders),
			reportSubcmd(fzfui.MinutesAppOpLeaderShare),
			reportSubcmd(fzfui.MinutesAppOpLeaderFootsteps),
		)
}

func reportAction(ctx context.Context, conn *db.Connection, singer string, stdout bool) (err error) {
	if singer == "" {
		var ec erc.Collector
		names := irt.Collect(erc.HandleAll(conn.AllLeaderNames(ctx), ec.Push))
		if err := ec.Resolve(); err != nil {
			return err
		}
		singer, err = infra.NewFuzzySearch[string](names).FindOne("singer")
		if err != nil {
			return err
		}
	}

	// TODO use the gitWriter function here to get a correct fileName and produce the correct writer.
	var w io.Writer
	if stdout {
		w = os.Stdout
	} else {
		f, err := getFile(singer)
		if err != nil {
			return err
		}

		defer func() { err = erc.Join(f.Close()) }()
		defer grip.Infof("report written to %s", f.Name())
		grip.Infof("writing report for %q to %s", singer, f.Name())
		w = f
	}

	if err := fullReport(ctx, conn, w, singer); err != nil {
		return err
	}

	return nil
}

// getWriter returns an io.Writer (stdout or a new file) plus a cleanup func.
// The caller must call cleanup() when done. For stdout, cleanup is a no-op.
func getWriter(stdout bool, singer string, tags ...string) (io.Writer, func() error, error) {
	if stdout {
		return os.Stdout, func() error { return nil }, nil
	}
	f, err := getFile(singer, tags...)
	if err != nil {
		return nil, nil, err
	}
	return f, f.Close, nil
}

func getFile(singer string, tags ...string) (*os.File, error) {
	mut := strut.MakeMutable(len(singer))
	defer mut.Release()
	mut.PushString(singer)
	mut.ReplaceAllString(" ", "-")
	mut.ReplaceAllString("’", "-")
	mut.ReplaceAllString(".", "")
	mut.ToLower()
	mut.JoinStrings(tags, "-")
	mut.PushString(".md")

	f, err := os.Create(mut.String())
	if err != nil {
		return nil, err
	}

	return f, nil
}

func fullReport(ctx context.Context, conn *db.Connection, w io.Writer, singer string) error {
	var ec erc.Collector
	var mb mdwn.Builder

	mb.H1(singer)
	mb.KV("Generated", time.Now().Format(time.DateOnly))
	mb.Line()

	share, err := conn.LeaderShareOfLeads(ctx, singer)
	ec.Push(err)
	v, err := conn.SingersConnectedness(ctx, singer)
	ec.Push(err)

	mb.KV("Share of All Leads", fmt.Sprintf("%.4f%%", stw.DerefZ(share)*100))
	mb.KV("Connectedness", fmt.Sprintf("%.2f%%", stw.DerefZ(v)*100))
	mb.Line()

	mb.H2("Most Led Songs")
	writeSongTable(&mb, erc.HandleAll(conn.MostLeadSongs(ctx, singer, 24), ec.Push))

	mb.H2("Songs in Your Experience")
	mb.Paragraph("Most frequently led songs at singings ", singer, " attended.")
	writeSongTable(&mb, erc.HandleAll(conn.PopularSongsInOnesExperience(ctx, singer, 12), ec.Push))

	mb.H2("Singing Buddies")
	mb.Paragraph("The people that have been the most singings that ", singer, " was at.")
	mb.KVTable(irt.MakeKV("Name", "Shared Singings"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingBuddies(ctx, singer, 24), ec.Push)), intValToStr),
	)
	mb.Line()

	mb.H2("Singing Strangers")
	mb.Paragraph("People that ", singer, " has never sung with who share many connections.")
	mb.KVTable(irt.MakeKV("Name", "Mutual Connections"),
		irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingStrangers(ctx, singer, 24), ec.Push)), intValToStr),
	)
	mb.Line()

	mb.H2("Singing Idols")
	mb.Paragraph("The top leaders of all of ", singer, "'s top songs!")
	writeLeaderFootstepTable(&mb, erc.HandleAll(conn.LeaderFootsteps(ctx, singer, 20), ec.Push))

	mb.H2("Unfamiliar Hits")
	mb.Paragraph("Othewise popular songs that are under represented at singings ", singer, " has been at.")
	writeSongTable(&mb, erc.HandleAll(conn.TheUnfamilarHits(ctx, singer, 20), ec.Push))

	mb.H2("Never Led")
	mb.Paragraph("Songs from the 2025 book that ", singer, " has never led, by global popularity.")
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverLed(ctx, singer), 12), ec.Push))

	mb.H2("Never Sung")
	mb.Paragraph("Songs that have not been called at a singing ", singer, " attended, by global popularity.")
	writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, singer), 12), ec.Push))

	// todo
	_, err = mb.WriteTo(w)
	ec.Push(err)
	return ec.Resolve()
}

func flush(wr io.Writer, payload io.WriterTo) (err error) { _, err = payload.WriteTo(wr); return }
func intValToStr(key string, value int) (string, string)  { return key, strconv.Itoa(value) }
func asRows(lsr models.LeaderSongRank) []string           { return (&lsr).StringFields() }
func writeSongTable(mb *mdwn.Builder, seq iter.Seq[models.LeaderSongRank]) {
	mb.NewTable(
		mdwn.Column{Name: "Count", RightAlign: true},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Title"},
		mdwn.Column{Name: "Key"},
	).Extend(irt.Convert(seq, asRows)).Build()

	mb.Line()
}

func writeLeaderFootstepTable(mb *mdwn.Builder, seq iter.Seq[models.LeaderFootstep]) {
	mb.NewTable(
		mdwn.Column{Name: "Song"},
		mdwn.Column{Name: "Page"},
		mdwn.Column{Name: "Key"},
		mdwn.Column{Name: "Top Leader"},
		mdwn.Column{Name: "Their Leads", RightAlign: true},
		mdwn.Column{Name: "Last Year", RightAlign: true},
		mdwn.Column{Name: "Self Leads", RightAlign: true},
	).Extend(irt.Convert(seq, func(row models.LeaderFootstep) []string {
		return []string{
			row.SongTitle,
			row.SongPage,
			row.SongKeys,
			row.LeaderName,
			strconv.Itoa(row.TheirLeadCount),
			strconv.Itoa(row.TheirLastLeadYear),
			strconv.Itoa(row.SelfLeadCount),
		}
	})).Build()

	mb.Line()
}

type reporterFunc func(context.Context, *db.Connection, io.Writer, string) error

func (r Reporter) Report(ctx context.Context, conn*db.Connection, wr io.Writer, argstring) error {
	return r(ctx, conn, wr, arg)
}

type reporter interface {
	Report((context.Context, *db.Connection, io.Writer, string) error)
}

func dispatcher(op fzfui.MinutesAppOperation) reporterFunc {
	// TODO reexamine the implementation in pkg/fzfui/dispatch. and modify this accordingly. Move these anonomus functions into named <name>Report(<...>) functions, 

	// TODO we should remove the io.Writer from the interface: there should be a params{} struct with: {arg string, usestdout bool, limit int } passed to each function instead. 


	switch op {
	case fzfui.MinutesAppOpLeaders:
		return fullReport
	case fzfui.MinutesAppOpSongs:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, arg string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			sg, err := conn.GetSong(ctx, arg)
			ec.Push(err)

			mb.H2(fmt.Sprintf("Song: %s — %s", sg.PageNum, sg.SongTitle))
			mb.KV("Page", sg.PageNum)
			mb.KV("Keys", sg.Keys)
			mb.KV("Meter", sg.SongMeter)
			mb.KV("Music", sg.MusicAttribution)
			mb.KV("Words", sg.WordsAttribution)
			mb.Line()

			mb.H3("Top Leaders")
			mb.NewTable(
				mdwn.Column{Name: "Name"},
				mdwn.Column{Name: "Count", RightAlign: true},
				mdwn.Column{Name: "Led Last Year"},
				mdwn.Column{Name: "Years Active", RightAlign: true},
			).Extend(irt.Convert(erc.HandleAll(conn.TopLeadersOfSong(ctx, sg.PageNum, 20), ec.Push), func(l models.LeaderOfSongInfo) []string {
				return []string{l.Name, strconv.Itoa(l.Count), strconv.FormatBool(l.LedInLastYear), strconv.Itoa(l.NumYears)}
			})).Build()
			mb.Line()

			ec.Push(flush(w, &mb))

			return ec.Resolve()
		}
	case fzfui.MinutesAppOpSingings:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, arg string) error {
			if arg == "" {
				return ers.New("singing name required")
			}
			var ec erc.Collector
			var mb mdwn.Builder

			// Find the singing by name
			var found *models.SingingInfo
			// TODO use erc.HandleAll to avoid this loop
			for s, err := range conn.AllSingings(ctx) {
				if !ec.PushOk(err) {
					break
				}

				if s.SingingName == arg {
					s := s
					found = &s
					break
				}
			}
			if !ec.Ok() {
				return ec.Resolve()
			}

			if found != nil {
				mb.H2(fmt.Sprintf("Singing: %s", found.SingingName))
				mb.KV("Date", found.SingingDate.Time().Format(time.DateOnly))
				mb.KV("Location", found.SingingLocation)
				mb.KV("State", found.SingingState)
				mb.KV("Lessons", strconv.FormatInt(found.NumberOfLessons, 10))
				mb.KV("Leaders", strconv.FormatInt(found.NumberOfLeaders, 10))
				mb.Line()
			} else {
				// TODO is there a case where we don't find a singing, but there's not an error? I don't believe so.
				//   regardless, iterator gets the first element 
				mb.H2(fmt.Sprintf("Singing: %s", arg))
			}

			mb.H3("Lessons")
			mb.NewTable(
				mdwn.Column{Name: "Lesson", RightAlign: true},
				mdwn.Column{Name: "Leader"},
				mdwn.Column{Name: "Song"},
				mdwn.Column{Name: "Key"},
				mdwn.Column{Name: "Title"},
			).Extend(irt.Convert(erc.HandleAll(conn.SingingLessons(ctx, arg), ec.Push), func(s models.SingingLessionInfo) []string {
				return []string{strconv.Itoa(s.LessonID), s.SingerName, s.SongPageNumber, s.SongKey, s.SongName}
			})).Build()
			mb.Line()

			ec.Push(flush(w, &mb))

			return ec.Resolve()
		}
	case fzfui.MinutesAppOpBuddies:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, singer string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			mb.H2(fmt.Sprintf("Singing Buddies: %s", singer))
			mb.KVTable(
				irt.MakeKV("Name", "Shared Singings"),
				irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingBuddies(ctx, singer, 24), ec.Push)), intValToStr),
			)
			mb.Line()

			ec.Push(flush(w, &mb))

			return ec.Resolve()
		}
	case fzfui.MinutesAppOpStrangers:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, singer string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			mb.H2(fmt.Sprintf("Singing Strangers: %s", singer))
			mb.KVTable(
				irt.MakeKV("Name", "Mutual Connections"),
				irt.Convert2(irt.KVsplit(erc.HandleAll(conn.SingingStrangers(ctx, singer, 24), ec.Push)), intValToStr),
			)
			mb.Line()

			ec.Push(flush(w, &mb))

			return ec.Resolve()
		}
	case fzfui.MinutesAppOpPopularInOnesExperience:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, singer string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			mb.H2(fmt.Sprintf("Popular in %s's Experience", singer))
			writeSongTable(&mb, erc.HandleAll(conn.PopularSongsInOnesExperience(ctx, singer, 25), ec.Push))

			ec.Push(flush(w, &mb))

			return ec.Resolve()
		}
	case fzfui.MinutesAppOpPopularInYears:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, arg string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			var years []int
			if arg != "" {
				for _, part := range strings.Split(arg, ",") {
					y, err := strconv.Atoi(strings.TrimSpace(part))
					if err == nil && y != 0 {
						years = append(years, y)
					}
				}
			}

			if len(years) > 0 {
				mb.H2(fmt.Sprintf("Globally Popular (years: %v)", years))
			} else {
				mb.H2("Globally Popular")
			}
			writeSongTable(&mb, erc.HandleAll(conn.GloballyPopularForYears(ctx, years...), ec.Push))

			ec.Push(flush(w, &mb))

			return ec.Resolve()
		}
	case fzfui.MinutesAppOpLocallyPopular:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, arg string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			var localities []models.SingingLocality
			if arg != "" {
				for _, part := range strings.Split(arg, ",") {
					localities = append(localities, models.NewSingingLocality(strings.TrimSpace(part)))
				}
			}

			mb.H2(fmt.Sprintf("Locally Popular: %s", arg))
			writeSongTable(&mb, erc.HandleAll(conn.LocallyPopular(ctx, 32, localities...), ec.Push))

			ec.Push(flush(w, &mb))
			return ec.Resolve()
		}
	case fzfui.MinutesAppOpRetry:
		return func(_ context.Context, _ *db.Connection, _ io.Writer, _ string) error {
			return nil
		}
	case fzfui.MinutesAppOpNeverSung:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, singer string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			mb.H2(fmt.Sprintf("Never Sung: %s", singer))
			writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverSung(ctx, singer), 20), ec.Push))

			ec.Push(flush(w, &mb))
			return ec.Resolve()
		}
	case fzfui.MinutesAppOpNeverLed:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, singer string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			mb.H2(fmt.Sprintf("Never Led: %s", singer))
			writeSongTable(&mb, erc.HandleAll(irt.Limit2(conn.NeverLed(ctx, singer), 20), ec.Push))

			ec.Push(flush(w, &mb))
			return ec.Resolve()
		}
	case fzfui.MinutesAppOpUnfamilarHits:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, singer string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			mb.H2(fmt.Sprintf("Unfamiliar Hits: %s", singer))
			writeSongTable(&mb, erc.HandleAll(conn.TheUnfamilarHits(ctx, singer, 20), ec.Push))

			ec.Push(flush(w, &mb))
			return ec.Resolve()
		}
	case fzfui.MinutesAppOpConnectedness:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, _ string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			mb.H2("Leaders by Connectedness")
			mb.KVTable(
				irt.MakeKV("Name", "Connectedness"),
				irt.Convert2(irt.KVsplit(erc.HandleAll(conn.AllLeaderConnectedness(ctx, 40), ec.Push)), func(k string, v float64) (string, string) {
					return k, fmt.Sprintf("%.4f%%", v*100)
				}),
			)
			mb.Line()

			ec.Push(flush(w, &mb))
			_, err := mb.WriteTo(w)
			ec.Push(err)
			return ec.Resolve()
		}
	case fzfui.MinutesAppOpTopLeaders:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, arg string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			var years []int
			if arg != "" {
				for _, part := range strings.Split(arg, ",") {
					y, err := strconv.Atoi(strings.TrimSpace(part))
					if err == nil && y != 0 {
						years = append(years, y)
					}
				}
			}

			if len(years) > 0 {
				mb.H2(fmt.Sprintf("Top Leaders (years: %v)", years))
			} else {
				mb.H2("Top Leaders")
			}

			pos := 0
			mb.NewTable(
				mdwn.Column{Name: "#", RightAlign: true},
				mdwn.Column{Name: "Name"},
				mdwn.Column{Name: "Leads", RightAlign: true},
				mdwn.Column{Name: "Last Year", RightAlign: true},
				mdwn.Column{Name: "%", RightAlign: true},
				mdwn.Column{Name: "Running Total %", RightAlign: true},
			).Extend(irt.Convert(erc.HandleAll(conn.TopLeadersByLeads(ctx, 40, years...), ec.Push), func(row models.LeaderLeadCount) []string {
				pos++
				return []string{strconv.Itoa(pos), row.Name, strconv.Itoa(row.Count), strconv.Itoa(row.LastLeadYear), fmt.Sprintf("%.2f%%", row.Percentage*100), fmt.Sprintf("%.2f%%", row.RunningTotal*100)}
			})).Build()
			mb.Line()

			ec.Push(flush(w, &mb))
			return ec.Resolve()
		}
	case fzfui.MinutesAppOpLeaderShare:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, arg string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			parts := strings.SplitN(arg, ",", 2)
			singer := strings.TrimSpace(parts[0])

			var years []int
			if len(parts) > 1 {
				for _, part := range strings.Split(parts[1], ",") {
					y, err := strconv.Atoi(strings.TrimSpace(part))
					if err == nil && y != 0 {
						years = append(years, y)
					}
				}
			}

			v, err := conn.LeaderShareOfLeads(ctx, singer, years...)
			ec.Push(err)

			label := "Share of All Leads"
			if len(years) > 0 {
				label = fmt.Sprintf("Share of Leads (%v)", years)
			}

			mb.H2(fmt.Sprintf("Leader Share: %s", singer))
			mb.KV("Leader", singer)
			mb.KV(label, fmt.Sprintf("%.4f%%", stw.DerefZ(v)*100))
			mb.Line()

			ec.Push(flush(w, &mb))
			return ec.Resolve()
		}
	case fzfui.MinutesAppOpLeaderFootsteps:
		return func(ctx context.Context, conn *db.Connection, w io.Writer, singer string) error {
			var ec erc.Collector
			var mb mdwn.Builder

			mb.H2(fmt.Sprintf("Leader Footsteps: %s", singer))
			writeLeaderFootstepTable(&mb, erc.HandleAll(conn.LeaderFootsteps(ctx, singer, 20), ec.Push))

			return ec.Resolve()
		}
	case fzfui.MinutesAppOpExit:
		grip.Info("goodbye!")
		return nil
	case fzfui.MinutesAppOpInvalid:
		erc.Invariant(ers.New("explicitly invalid"))
		return nil
	case fzfui.MinutesAppOpUnknown:
		erc.Invariant(ers.New("unknown"))
		return nil
	default:
		erc.Invariant(ers.New("implicitly invalid"))
		return nil
	}
}
