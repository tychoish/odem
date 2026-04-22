// Package navigator implements a dynamic fuzzy terminal interface for exploring
// the minutes data via a state machine of chained fzf menus.
//
// The design mirrors tgbot's state machine: stateFn is a function that
// performs one step and returns the next step to execute. discoverNext
// iterates over the current operation's Requires set minus what has already
// been gathered, calling selectFor on the first unsatisfied requirement.
// When all requirements are satisfied it calls renderResults.
//
// An entityContext persists resolved leaders, songs, and singings across
// successive operations so the user does not have to re-select them.
// selectFor checks the entityContext before launching an fzf prompt.
package navigator

import (
	"context"
	"fmt"
	"strings"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/grip"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/dispatch"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/models"
)

// stateFn is one step in the navigator state machine. It performs some
// interaction (fzf prompt, run reporter) and returns the next step, or nil
// to exit.
type stateFn func(context.Context) (stateFn, error)

// queryState mirrors the tgbot's queryState. It tracks the selected operation,
// which of its requirements have been gathered, and the params built so far.
type queryState struct {
	op     dispatch.MinutesOperation
	entry  *dispatch.MinutesOpRegistration
	has    *dt.Set[dispatch.MinutesAppQueryType]
	params models.Params
}

// entityContext stores resolved entities across operations. selectFor checks
// here before prompting interactively, letting the user pick a leader once
// and run many operations against it without re-selecting.
type entityContext struct {
	leader  *models.LeaderProfile
	song    *models.SongDetail
	singing *models.SingingInfo
	years   []int
}

// ErrNavigatorExt is returned when the user explicitly exits the navigator.
const ErrNavigatorExt = ers.Error("navigator: user exit")

// Navigator drives the state machine loop.
type Navigator struct {
	conn   *db.Connection
	qstate queryState
	entity entityContext
}

// New creates a Navigator ready to run.
func New(conn *db.Connection) *Navigator {
	n := &Navigator{conn: conn}
	n.reset()
	return n
}

// reset clears operation state between runs. Entity context is preserved.
func (n *Navigator) reset() {
	n.qstate.entry = nil
	n.qstate.has = &dt.Set[dispatch.MinutesAppQueryType]{}
	n.qstate.params = models.Params{}
}

// Run executes the state machine until exit or a fatal error. Per-step errors
// are logged as warnings and the machine resets to the main menu.
func (n *Navigator) Run(ctx context.Context) error {
	for current := n.mainMenu; current != nil; {
		next, err := current(ctx)
		if err != nil {
			if ers.Is(err, ErrNavigatorExt) {
				return nil
			}
			grip.Warning(grip.KV("navigator", "error").KV("err", err))
			n.reset()
			current = n.mainMenu
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		current = next
	}
	return nil
}

// ── main menu ──────────────────────────────────────────────────────────────

const (
	menuSelectOp = "select operation"
	menuExit     = "exit"
)

func (n *Navigator) mainMenu(ctx context.Context) (stateFn, error) {
	options := []string{menuSelectOp, menuExit}
	if desc := n.currentEntityDescription(); desc != "" {
		options = append([]string{fmt.Sprintf("continue with: %s", desc)}, options...)
	}

	chosen, err := infra.NewFuzzySearch[string](options).
		Prompt("odem navigator").
		FindOne()

	switch {
	case err != nil:
		return nil, ers.Wrap(ErrNavigatorExt, "unexpected")
	case chosen == menuExit:
		return nil, ErrNavigatorExt
	default:
		// both menuSelectOp and "continue with: ..." land here
		return n.selectOp, nil
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

// currentEntityDescription returns a short description of the currently-held entities.
func (n *Navigator) currentEntityDescription() string {
	var parts []string
	if n.entity.leader != nil {
		parts = append(parts, n.entity.leader.Name)
	}
	if n.entity.song != nil {
		parts = append(parts, n.entity.song.MenuFormat())
	}
	if n.entity.singing != nil {
		parts = append(parts, n.entity.singing.MenuFormat())
	}
	if len(n.entity.years) > 0 {
		parts = append(parts, fmt.Sprintf("years:%v", n.entity.years))
	}
	return strings.Join(parts, ", ")
}
