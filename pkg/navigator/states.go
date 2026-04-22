package navigator

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/reportui"
	"github.com/tychoish/odem/pkg/selector"
)

// isInputQueryType reports whether qt is a type that requires interactive
// input from the user.
func isInputQueryType(qt dispatch.MinutesAppQueryType) bool {
	switch qt {
	case dispatch.MinutesAppQueryTypeLeader,
		dispatch.MinutesAppQueryTypeSong,
		dispatch.MinutesAppQueryTypeSinging,
		dispatch.MinutesAppQueryTypeYear,
		dispatch.MinutesAppQueryTypeKey,
		dispatch.MinutesAppQueryTypeLocality,
		dispatch.MinutesAppQueryTypeWord:
		return true
	default:
		return false
	}
}

// ── operation selection ────────────────────────────────────────────────────

type opEntry struct {
	op        dispatch.MinutesOperation
	display   string
	available bool
}

// Navigation entry display strings. Unset entries use prefixes so they can be
// detected without a separate field; the suffix carries the entity description.
// Set entries are exact matches (no entity value appended).
const (
	navBack         = "← back to menu"
	navExit         = "✗ exit"
	navSetLeader    = "→ set leader"
	navSetSong      = "→ set song"
	navSetSinging   = "→ set singing"
	navSetYears     = "→ set year(s)"
	navUnsetLeader  = "unset leader: "
	navUnsetSong    = "unset song: "
	navUnsetSinging = "unset singing: "
	navUnsetYears   = "unset year(s): "
)

// opIsGlobal reports whether reg has no isInputQueryType requirements, meaning
// it runs without any entity selection (e.g. connectedness, all-leaders).
func opIsGlobal(reg dispatch.MinutesOpRegistration) bool {
	if reg.Requires == nil {
		return true
	}
	for req := range reg.Requires.Iterator() {
		if isInputQueryType(req) {
			return false
		}
	}
	return true
}

// opMarkerLabel returns the short category label for an operation: "browse",
// "global", "ready", or "" (needs entity input). The label is unpadded.
func (n *Navigator) opMarkerLabel(reg dispatch.MinutesOpRegistration) string {
	switch {
	case reg.IsBrowse():
		return "browse"
	case opIsGlobal(reg):
		return "global"
	case n.opAvailable(reg):
		return "ready"
	default:
		return ""
	}
}

// opMarker pads label to a fixed width for column-aligned display.
const opMarkerWidth = 9

func opMarker(label string) string {
	if len(label) >= opMarkerWidth {
		return label[:opMarkerWidth]
	}
	return label + strings.Repeat(" ", opMarkerWidth-len(label))
}

func (n *Navigator) selectOp(_ context.Context) (stateFn, error) {
	var avail, notAvail []opEntry

	for op := range dispatch.AllMinutesAppNavigatorOps() {
		reg := op.Registry()
		label := n.opMarkerLabel(reg)
		e := opEntry{
			op:        op,
			available: label != "",
			display:   fmt.Sprintf("[%s] %-24s  %s", opMarker(label), op.String(), reg.Description),
		}
		if label != "" {
			avail = append(avail, e)
		} else {
			notAvail = append(notAvail, e)
		}
	}

	entries := append(avail, notAvail...)

	// Context management: set entries for unset entities, unset entries for set ones.
	if n.entity.leader == nil {
		entries = append(entries, opEntry{display: navSetLeader})
	} else {
		entries = append(entries, opEntry{display: navUnsetLeader + n.entity.leader.Name})
	}
	if n.entity.song == nil {
		entries = append(entries, opEntry{display: navSetSong})
	} else {
		entries = append(entries, opEntry{display: navUnsetSong + n.entity.song.MenuFormat()})
	}
	if n.entity.singing == nil {
		entries = append(entries, opEntry{display: navSetSinging})
	} else {
		entries = append(entries, opEntry{display: navUnsetSinging + n.entity.singing.MenuFormat()})
	}
	if len(n.entity.years) == 0 {
		entries = append(entries, opEntry{display: navSetYears})
	} else {
		entries = append(entries, opEntry{display: fmt.Sprintf("%s%v", navUnsetYears, n.entity.years)})
	}

	entries = append(entries,
		opEntry{display: navBack},
		opEntry{display: navExit},
	)

	prompt := "select operation"
	if desc := n.currentEntityDescription(); desc != "" {
		prompt = fmt.Sprintf("operation [%s]", desc)
	}

	chosen, err := infra.NewFuzzySearch[opEntry](entries).
		WithToString(func(e opEntry) string { return e.display }).
		Prompt(prompt).
		FindOne()
	if err != nil {
		return n.mainMenu, nil
	}

	switch {
	case chosen.display == navBack:
		n.reset()
		return n.mainMenu, nil
	case chosen.display == navExit:
		return nil, ErrNavigatorExt
	case chosen.display == navSetLeader:
		return n.selectFor(dispatch.MinutesAppQueryTypeLeader), nil
	case chosen.display == navSetSong:
		return n.selectFor(dispatch.MinutesAppQueryTypeSong), nil
	case chosen.display == navSetSinging:
		return n.selectFor(dispatch.MinutesAppQueryTypeSinging), nil
	case chosen.display == navSetYears:
		return n.selectFor(dispatch.MinutesAppQueryTypeYear), nil
	case strings.HasPrefix(chosen.display, navUnsetLeader):
		n.entity.leader = nil
		return n.selectOp, nil
	case strings.HasPrefix(chosen.display, navUnsetSong):
		n.entity.song = nil
		return n.selectOp, nil
	case strings.HasPrefix(chosen.display, navUnsetSinging):
		n.entity.singing = nil
		return n.selectOp, nil
	case strings.HasPrefix(chosen.display, navUnsetYears):
		n.entity.years = nil
		return n.selectOp, nil
	}

	if !n.setupQuery(chosen.op) {
		return n.selectOp, nil
	}
	return n.discoverNext, nil
}

// setupQuery initialises queryState for op, mirroring tgbot's setupQuery.
func (n *Navigator) setupQuery(op dispatch.MinutesOperation) bool {
	if !op.Ok() {
		return false
	}
	reg := op.Registry()
	n.qstate.op = op
	n.qstate.entry = &reg
	n.qstate.has = &dt.Set[dispatch.MinutesAppQueryType]{}
	n.qstate.params = models.Params{}
	return true
}

// opAvailable reports whether every requirement of reg is already covered by
// the entity context, meaning discoverNext will reach renderResults without
// any interactive prompts. Operations with nil or empty Requires are always
// available.
func (n *Navigator) opAvailable(reg dispatch.MinutesOpRegistration) bool {
	if reg.Requires == nil {
		return true
	}
	for req := range reg.Requires.Iterator() {
		if !isInputQueryType(req) {
			continue
		}
		switch req {
		case dispatch.MinutesAppQueryTypeLeader:
			if n.entity.leader == nil {
				return false
			}
		case dispatch.MinutesAppQueryTypeSong:
			if n.entity.song == nil {
				return false
			}
		case dispatch.MinutesAppQueryTypeSinging:
			if n.entity.singing == nil {
				return false
			}
		case dispatch.MinutesAppQueryTypeYear:
			if len(n.entity.years) == 0 {
				return false
			}
		default:
			// Key, Locality, Word always require interactive input.
			return false
		}
	}
	return true
}

// ── discoverNext (mirrors tgbot's discoverNext) ───────────────────────────

// discoverNext iterates over the current operation's unsatisfied requirements
// and dispatches to selectFor for the first one found. When all requirements
// are satisfied it proceeds to renderResults.
func (n *Navigator) discoverNext(ctx context.Context) (stateFn, error) {
	if n.qstate.entry == nil {
		return n.selectOp, nil
	}
	if n.qstate.entry.Requires == nil {
		grip.Debug(grip.KV("navigator", "discoverNext").KV("status", "nil requires; rendering"))
		return n.renderResults, nil
	}
	if n.qstate.has == nil {
		grip.Debug(grip.KV("navigator", "discoverNext").KV("status", "nil has; rendering"))
		return n.renderResults, nil
	}

	for req := range irt.Remove(
		irt.RemoveValue(n.qstate.entry.Requires.Iterator(), dispatch.MinutesAppQueryTypeDocumentOutput),
		n.qstate.has.Check,
	) {
		grip.Debug(grip.KV("navigator", "discoverNext").KV("requirement", req).KV("op", n.qstate.entry.Command))
		return n.selectFor(req), nil
	}

	grip.Debug(grip.KV("navigator", "discoverNext").KV("status", "all satisfied; rendering").KV("op", n.qstate.entry.Command))
	return n.renderResults, nil
}

// ── selectFor (mirrors tgbot's selectFor / captureXxx) ────────────────────

// ensureEntity fetches *ptr if nil via fetch, then returns the entity's name.
// This eliminates the repeated nil-check/fetch/assign pattern for the three
// entity selector cases in selectFor.
func ensureEntity[T any](ptr **T, fetch func() (*T, error), name func(*T) string) (string, error) {
	if *ptr == nil {
		v, err := fetch()
		if err != nil {
			return "", err
		}
		*ptr = v
	}
	return name(*ptr), nil
}

// selectFor returns a stateFn that gathers one requirement. For entity types
// (Leader, Song, Singing) it checks the entityContext before prompting, so a
// pre-selected entity is reused without an extra fzf dialog. After storing the
// value it returns discoverNext.
func (n *Navigator) selectFor(req dispatch.MinutesAppQueryType) stateFn {
	return func(ctx context.Context) (stateFn, error) {
		sp := new(infra.SearchParams).Interaction(true)
		var err error

		switch req {
		case dispatch.MinutesAppQueryTypeLeader:
			n.qstate.params.Name, err = ensureEntity(&n.entity.leader,
				func() (*models.LeaderProfile, error) { return selector.Leader(ctx, n.conn, sp) },
				func(v *models.LeaderProfile) string { return v.Name })

		case dispatch.MinutesAppQueryTypeSong:
			// MenuFormat ("pg 123 -- Title") gives an unambiguous fuzzy match.
			n.qstate.params.Name, err = ensureEntity(&n.entity.song,
				func() (*models.SongDetail, error) { return selector.Song(ctx, n.conn, sp) },
				func(v *models.SongDetail) string { return v.MenuFormat() })

		case dispatch.MinutesAppQueryTypeSinging:
			n.qstate.params.Name, err = ensureEntity(&n.entity.singing,
				func() (*models.SingingInfo, error) { return selector.Singing(ctx, n.conn, sp) },
				func(v *models.SingingInfo) string { return v.MenuFormat() })

		case dispatch.MinutesAppQueryTypeYear:
			if len(n.entity.years) == 0 {
				n.entity.years, err = selector.Years(sp)
			}
			n.qstate.params.Years = n.entity.years

		case dispatch.MinutesAppQueryTypeKey:
			n.qstate.params.Name, err = selector.Key(ctx, n.conn, sp)

		case dispatch.MinutesAppQueryTypeLocality:
			var loc models.SingingLocality
			loc, err = selector.Locality(sp)
			n.qstate.params.Name = string(loc)

		case dispatch.MinutesAppQueryTypeWord:
			n.qstate.params.Name, err = selector.Concordance(ctx, n.conn, sp)
		}

		if err != nil {
			return n.selectOp, err
		}
		n.qstate.has.Add(req)
		return n.discoverNext, nil
	}
}

// ── renderResults ──────────────────────────────────────────────────────────

func (n *Navigator) renderResults(ctx context.Context) (stateFn, error) {
	var buf bytes.Buffer
	params := reportui.Params{
		Params:   n.qstate.params,
		ToWriter: &buf,
	}

	grip.Info(grip.KV("navigator", "render").KV("op", n.qstate.entry.Command).KV("params", n.qstate.params))

	if err := n.qstate.entry.GetReporter()(ctx, n.conn, params); err != nil {
		grip.Warning(grip.KV("navigator", "render error").KV("op", n.qstate.entry.Command).KV("err", err))
	}

	var lines []string
	for _, line := range strings.Split(buf.String(), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}

	if len(lines) > 0 {
		prompt := fmt.Sprintf("%s results", n.qstate.entry.Command)
		// FindOne is used only for browsing; the selection result is discarded.
		_, _ = infra.NewFuzzySearch[string](lines).Prompt(prompt).FindOne()
	}

	// Browse operations are entity selectors — they return to selectOp so the
	// user can immediately apply an operation to whatever they were browsing.
	// All other operations offer follow mode to re-run or swap an entity.
	if n.qstate.entry.IsBrowse() {
		n.reset()
		return n.selectOp, nil
	}
	return n.followMenu, nil
}

// ── followMenu ─────────────────────────────────────────────────────────────

const (
	followRunAgain   = "↻ run again"
	followNewOp      = "→ new operation"
	followBackToMenu = "← back to menu"
)

// followEntry pairs a display string with the state transition to take.
type followEntry struct {
	display string
	next    stateFn
}

// followMenu is reached after renderResults. It lets the user re-run the same
// operation (optionally re-specifying one of its requirements) or navigate
// elsewhere. Options are built dynamically from the operation's Requires set
// so all input types — leader, song, singing, year, key, locality, word —
// appear when relevant.
func (n *Navigator) followMenu(ctx context.Context) (stateFn, error) {
	op := n.qstate.op

	entries := []followEntry{{
		display: followRunAgain,
		next: func(ctx context.Context) (stateFn, error) {
			n.setupQuery(op)
			return n.discoverNext, nil
		},
	}}

	// One entry per isInputQueryType requirement, derived from the op's Requires set.
	if reqs := n.qstate.entry.Requires; reqs != nil {
		for req := range reqs.Iterator() {
			if !isInputQueryType(req) {
				continue
			}
			req := req // capture loop variable
			entries = append(entries, followEntry{
				display: n.followRespecifyLabel(req),
				next: func(ctx context.Context) (stateFn, error) {
					n.clearEntityForReq(req)
					n.setupQuery(op)
					return n.selectFor(req), nil
				},
			})
		}
	}

	entries = append(entries,
		followEntry{display: followNewOp, next: func(ctx context.Context) (stateFn, error) {
			n.reset()
			return n.selectOp, nil
		}},
		followEntry{display: followBackToMenu, next: func(ctx context.Context) (stateFn, error) {
			n.reset()
			return n.mainMenu, nil
		}},
		followEntry{display: navExit, next: func(ctx context.Context) (stateFn, error) {
			return nil, ErrNavigatorExt
		}},
	)

	chosen, err := infra.NewFuzzySearch[followEntry](entries).
		WithToString(func(e followEntry) string { return e.display }).
		Prompt(fmt.Sprintf("follow: %s", op.String())).
		FindOne()
	if err != nil {
		n.reset()
		return n.mainMenu, nil
	}

	return chosen.next(ctx)
}

// followRespecifyLabel returns the display label for a "re-specify" follow option,
// including the current entity value when one is in context.
func (n *Navigator) followRespecifyLabel(req dispatch.MinutesAppQueryType) string {
	switch req {
	case dispatch.MinutesAppQueryTypeLeader:
		if n.entity.leader != nil {
			return "change leader: " + n.entity.leader.Name
		}
		return "specify leader"
	case dispatch.MinutesAppQueryTypeSong:
		if n.entity.song != nil {
			return "change song: " + n.entity.song.MenuFormat()
		}
		return "specify song"
	case dispatch.MinutesAppQueryTypeSinging:
		if n.entity.singing != nil {
			return "change singing: " + n.entity.singing.MenuFormat()
		}
		return "specify singing"
	case dispatch.MinutesAppQueryTypeYear:
		if len(n.entity.years) > 0 {
			return fmt.Sprintf("change year(s): %v", n.entity.years)
		}
		return "specify year(s)"
	case dispatch.MinutesAppQueryTypeKey:
		return "re-specify key"
	case dispatch.MinutesAppQueryTypeLocality:
		return "re-specify locality"
	case dispatch.MinutesAppQueryTypeWord:
		return "re-specify word"
	default:
		return fmt.Sprintf("re-specify %s", req)
	}
}

// clearEntityForReq removes the entity-context value for the given requirement
// type so that selectFor will prompt for a fresh value.
func (n *Navigator) clearEntityForReq(req dispatch.MinutesAppQueryType) {
	switch req {
	case dispatch.MinutesAppQueryTypeLeader:
		n.entity.leader = nil
	case dispatch.MinutesAppQueryTypeSong:
		n.entity.song = nil
	case dispatch.MinutesAppQueryTypeSinging:
		n.entity.singing = nil
	case dispatch.MinutesAppQueryTypeYear:
		n.entity.years = nil
	}
}
