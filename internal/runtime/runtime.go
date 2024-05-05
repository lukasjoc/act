package runtime

import (
	"fmt"
	"strconv"

	"github.com/lukasjoc/act/internal/parse"
)

type Env struct {
	module parse.Module
	actors map[string]*actor
	sched  *scheduler
}

func New(module parse.Module) *Env {
	sched := newScheduler()
	return &Env{module, map[string]*actor{}, sched}
}

func (e *Env) Exec() error {
	atProc := e.sched.startAtProc()
	<-atProc.ok
	for _, item := range e.module {
		switch s := item.(type) {
		case parse.ActorStmt:
			state, err := strconv.Atoi(*s.State.Value)
			if err != nil {
				return err
			}
			a := newActor(*s.Ident.Value, int(state))
			for _, action := range s.Actions {
				action := action
				id := messageId(*action.Ident.Value)
				a.setAction(id, func(m *message) (returnPid uint16) {
					if len(action.Params) != len(m.args) {
						fmt.Printf("ERROR: message `%s` in actor `%s` requires `%v` args\n",
							id, *s.Ident.Value, len(action.Params))
						return 0
					}
					locals := map[string]int{}
					for pos, param := range action.Params {
						locals[*param.Value] = m.args[pos]
					}
					if action.ReturnPid != nil {
						returnPidVal, ok := locals[*action.ReturnPid.Value]
						if !ok {
							fmt.Printf("ERROR: message `%s` in actor `%s` requires a from arg\n", id, *s.Ident.Value)
							return 0
						}
						returnPid = uint16(returnPidVal)
					}
					ctx := newEvalCtx(action.Scope, (*a).state, locals)
					if err := ctx.eval(); err != nil {
						fmt.Printf("EVAL ERROR: %v \n", err)
						return 0
					}
					a.state = ctx.state
					return returnPid
				})
			}
			if _, defined := e.actors[*s.Ident.Value]; defined {
				return fmt.Errorf("actor with name `%s` is already defined", *s.Ident.Value)
			}
			e.actors[*s.Ident.Value] = a
			fmt.Printf("NEW ACTOR: %v [%v %v]\n", s.Ident.Value, a.state, a.actions)
		case parse.SendStmt:
			// this should not happend
			if s.SendPid == nil {
				return fmt.Errorf("need sendTo pid ident in send")
			}
			p, err := e.sched.procByName(*s.SendPid.Value)
			if err != nil {
				return err
			}
			args := []int{}
			for _, t := range s.Args {
				if *t.Value == "@" {
					args = append(args, int(atPid))
					continue
				}
				v, err := strconv.Atoi(*t.Value)
				if err != nil {
					return err
				}
				args = append(args, v)
			}
			p.recv(&message{messageId(*s.Message.Value), args})
		case parse.SpawnStmt:
			id := s.Scope[0].Value
			a, defined := e.actors[*id]
			if !defined {
				return fmt.Errorf("actor with name `%s` not defined yet", *id)
			}
			e.sched.startProc(*s.PidIdent.Value, a)
		default:
			panic(fmt.Sprintf("item `%v` is not supported yet", item))
		}
	}

	e.sched.wg.Wait()
	return nil
}
