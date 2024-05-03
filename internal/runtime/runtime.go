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
	e.sched.startAtProc()
	p, _ := e.sched.proc("@")
	<-p.ok
	for _, item := range e.module {
		switch s := item.(type) {
		case parse.ActorStmt:
			state, err := strconv.Atoi(s.State.Value)
			if err != nil {
				return err
			}
			a := newActor(s.Ident.Value, int(state))
			for _, action := range s.Actions {
				action := action
				id := messageId(action.Ident.Value)
				a.setAction(id, func(m *message) *pid {
					if len(action.Params) != len(m.args) {
						fmt.Printf("ERROR: message `%s` in actor `%v` requires `%v` args\n",
							id, s.Ident.Value, len(action.Params))
						return nil
					}
					locals := map[string]int{}
					for pos, param := range action.Params {
						locals[param.Value] = m.args[pos]
					}
					// TODO: if pid not in locals then returnPid wrong etc.. --> fail
					var returnPid = pid(locals[action.ReturnPid.Value])
					ctx := newEvalCtx(action.Scope, (*a).state, locals)
					if err := ctx.eval(); err != nil {
						fmt.Printf("EVAL ERROR: %v \n", err)
						return nil
					}
					(*a).state = ctx.state
					return &returnPid
				})
			}
			if _, defined := e.actors[s.Ident.Value]; defined {
				return fmt.Errorf("actor with name `%s` is already defined", s.Ident.Value)
			}
			e.actors[s.Ident.Value] = a
			fmt.Printf("NEW ACTOR: %v [%v %v]\n", s.Ident.Value, a.state, a.actions)
		case parse.ShowStmt:
			id := s.ActorIdent.Value
			a, defined := e.actors[id]
			if !defined {
				return fmt.Errorf("actor with name `%s` not defined yet", id)
			}
			a.show()
		case parse.SendStmt:
			p, err := e.sched.proc(s.ActorIdent.Value)
			if err != nil {
				return err
			}
			args := []int{}
			for _, t := range s.Args {
				// FIXME: refer to self properly (currently @ refers to the `@` process)
				if t.Value == "@" {
					args = append(args, pid1)
					continue
				}
				v, err := strconv.Atoi(t.Value)
				if err != nil {
					return err
				}
				args = append(args, v)
			}
			p.recv(&message{messageId(s.Message.Value), args})
		case parse.SpawnStmt:
			id := s.Scope[0].Value
			a, defined := e.actors[id]
			if !defined {
				return fmt.Errorf("actor with name `%s` not defined yet", id)
			}
			e.sched.startProc(s.PidIdent.Value, a)
		default:
			panic(fmt.Sprintf("item `%v` is not supported yet", item))
		}
	}

	e.sched.wg.Wait()
	return nil
}
