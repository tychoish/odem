package dispatch

import (
	"iter"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/odem/pkg/infra"
	"github.com/tychoish/odem/pkg/mcpsrv"
	"github.com/tychoish/odem/pkg/msgui"
)

type aliasMap struct {
	adt.SyncMap[string, MinutesAppOperation]
}

var aliases aliasMap

func init()                                       { aliases.addCommands(); aliases.addAlaises(); aliases.addFallback() }
func getAliases(mao MinutesAppOperation) []string { return mao.Aliases() }
func (am *aliasMap) addFallback()                 { am.Store("", MinutesAppOpInvalid) }
func (am *aliasMap) addAlaises()                  { am.Extend(infra.ReverseMapping(AllMinutesAppAliases())) }
func (am *aliasMap) addCommands()                 { am.Extend(AllMinutesAppCommands()) }

type MinutesAppRegistration struct {
	ID          MinutesAppOperation
	Command     string
	Description string
	Aliases     []string
	Reporter    Reporter
	Fuzz        FuzzHandler
	Messenger   msgui.Messenger
	MCP         mcpsrv.RegistrationFunc
	Requires    *dt.Set[MinutesAppQueryType]
	err         error
}

func (reg MinutesAppRegistration) Ok() bool                     { return reg.ID.Ok() }
func (reg MinutesAppRegistration) Validate() error              { return reg.err }
func (reg MinutesAppRegistration) infoKV() (string, string)     { return reg.Command, reg.Description }
func (reg MinutesAppRegistration) Info() irt.KV[string, string] { return irt.MakeKV(reg.infoKV()) }
func (reg MinutesAppRegistration) GetFuzzHandler() FuzzHandler  { return resolver(reg.Fuzz, reg.err) }
func (reg MinutesAppRegistration) GetReporter() Reporter        { return resolver(reg.Reporter, reg.err) }

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
