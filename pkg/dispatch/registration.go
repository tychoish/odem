package dispatch

import (
	"cmp"
	"context"
	"io"
	"iter"
	"strings"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/mdwn"
	"github.com/tychoish/odem/pkg/db"
	"github.com/tychoish/odem/pkg/mcpsrv"
	"github.com/tychoish/odem/pkg/models"
	"github.com/tychoish/odem/pkg/msgui"
	"github.com/tychoish/odem/pkg/reportui"
)

type aliasMap struct {
	adt.SyncMap[string, MinutesAppOperation]
}

var aliases aliasMap

func init() {
	aliases.addCommands()
	aliases.addAlaises()
	aliases.addFallback()
	aliases.withJoinedWords()
	aliases.withSpacedWords()
}

func getAliases(mao MinutesAppOperation) []string { return mao.Aliases() }
func (am *aliasMap) addFallback()                 { am.Store("", MinutesAppOpInvalid) }
func (am *aliasMap) addAlaises()                  { am.Extend(MinutesAppAliasMapping()) }
func (am *aliasMap) addCommands()                 { am.Extend(AllMinutesAppCommands()) }
func (am *aliasMap) with(f aliasFilter)           { am.Extend(irt.Convert2(MinutesAppAliasMapping(), f)) }
func (am *aliasMap) withJoinedWords()             { am.with(joinKebabs) }
func (am *aliasMap) withSpacedWords()             { am.with(replaceKebabsWithSpace) }

type MinutesAppRegistration struct {
	ID          MinutesAppOperation
	Command     string
	Description string
	Aliases     []string
	Reporter  Reporter
	Fuzz      FuzzHandler
	Messenger msgui.Messenger
	MCP       mcpsrv.RegistrationFunc
	Requires    *dt.Set[MinutesAppQueryType]
	err         error
}

func (reg MinutesAppRegistration) Ok() bool                     { return reg.ID.Ok() }
func (reg MinutesAppRegistration) Validate() error              { return reg.err }
func (reg MinutesAppRegistration) infoKV() (string, string)     { return reg.Command, reg.Description }
func (reg MinutesAppRegistration) Info() irt.KV[string, string] { return irt.MakeKV(reg.infoKV()) }
func (reg MinutesAppRegistration) HasMessenger() bool { return reg.Messenger != nil }
func (reg MinutesAppRegistration) HasReporter() bool  { return reg.Reporter != nil }
func (reg MinutesAppRegistration) HasFuzz() bool      { return reg.Fuzz != nil }
func (reg MinutesAppRegistration) unavailable() error { return unavailableOp(reg.Command) }

// IsDocumentOp reports whether this operation renders its output as a file
// attachment (signalled by MinutesAppQueryTypeDocumentOutput in Requires).
func (reg MinutesAppRegistration) IsDocumentOp() bool {
	return reg.Requires != nil && reg.Requires.Check(MinutesAppQueryTypeDocumentOutput)
}

// DocumentFilename returns the suggested attachment filename for this operation.
func (reg MinutesAppRegistration) DocumentFilename(params models.Params) string {
	name := strings.ToLower(strings.ReplaceAll(params.Name, " ", "-"))
	return name + "-" + reg.Command + ".md"
}

// CallReporterToWriter invokes the Reporter writing output to w rather than a file.
func (reg MinutesAppRegistration) CallReporterToWriter(ctx context.Context, conn *db.Connection, params models.Params, w io.Writer) error {
	return reg.GetReporter()(ctx, conn, reportui.Params{
		Params:                params,
		ToWriter:              w,
		SuppressInteractivity: true,
	})
}

func (reg MinutesAppRegistration) GetFuzzHandler() FuzzHandler { return resolver(reg, reg.Fuzz) }
func (reg MinutesAppRegistration) GetReporter() Reporter       { return resolver(reg, reg.Reporter) }

func (reg MinutesAppRegistration) GetMessenger() msgui.Messenger {
	if reg.Messenger != nil {
		return reg.Messenger
	}
	err := cmp.Or(reg.err, unavailableOp(reg.Command))
	return func(_ context.Context, _ *db.Connection, _ models.Params) iter.Seq2[*mdwn.Builder, error] {
		return func(yield func(*mdwn.Builder, error) bool) { yield(nil, err) }
	}
}

func AllMinutesAppFuzzOps() iter.Seq[MinutesAppOperation] {
	return func(yield func(MinutesAppOperation) bool) {
		for op := range AllMinutesAppOps() {
			if op.Registry().HasFuzz() {
				if !yield(op) {
					return
				}
			}
		}
	}
}

// AllMinutesAppMessengerOps returns operations available to the Telegram bot:
// streaming-message ops (HasMessenger) and file-document ops (IsDocumentOp).
func AllMinutesAppMessengerOps() iter.Seq[MinutesAppOperation] {
	return func(yield func(MinutesAppOperation) bool) {
		for op := range AllMinutesAppOps() {
			r := op.Registry()
			if r.HasMessenger() || r.IsDocumentOp() {
				if !yield(op) {
					return
				}
			}
		}
	}
}

func AllMinutesAppMCPHandlers() iter.Seq2[irt.KV[string, string], mcpsrv.RegistrationFunc] {
	return func(yield func(irt.KV[string, string], mcpsrv.RegistrationFunc) bool) {
		for op := range AllMinutesAppOps() {
			r := op.Registry()
			if r.MCP == nil {
				continue
			}
			if !yield(r.Info(), r.MCP) {
				return
			}
		}
	}
}
