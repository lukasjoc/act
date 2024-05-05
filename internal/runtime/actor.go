package runtime

import (
	"fmt"

	"github.com/lukasjoc/act/internal/parse"
)

type actor struct {
	name    string
	state   int
	actions map[messageId]actionFunc
}

type actionFunc func(*message, *proc) uint16

func newActionFunc(action *parse.ActorActionStmt) actionFunc {
	return func(m *message, p *proc) uint16 {
		if len(action.Params) != len(m.args) {
			fmt.Printf("ERROR: message `%s` in actor `%s` requires `%v` args\n",
				m.id, p.a.name, len(action.Params))
			return 0
		}
		locals := map[string]int{}
		for pos, param := range action.Params {
			locals[*param.Value] = m.args[pos]
		}
		ctx := newEvalCtx(action.Scope, p.a.state, locals)
		if err := ctx.eval(); err != nil {
			fmt.Printf("EVAL ERROR: %v \n", err)
			return 0
		}
		p.a.state = ctx.state
		if action.ReturnPid != nil {
			returnPidVal, ok := locals[*action.ReturnPid.Value]
			if !ok {
				fmt.Printf("ERROR: message `%s` in actor `%s` requires a from arg\n", m.id, p.a.name)
				return 0
			}
			return uint16(returnPidVal)
		}
		return 0
	}
}

func (a *actor) setAction(id messageId, f actionFunc) {
	// TODO: figure out a way to move more stuff into this default impl
	a.actions[id] = f
}
