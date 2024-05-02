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
	for _, item := range e.module {
		switch s := item.(type) {
		case parse.ActorStmt:
			state, err := strconv.Atoi(s.State.Value)
			if err != nil {
				return err
			}
			a := newActor(s.Ident.Value, int(state))
			for _, action := range s.Actions {
				id := messageId(action.Ident.Value)
				locals := []string{}
				for _, t := range action.Params {
					locals = append(locals, t.Value)
				}
				a.With(id, func(a *actor, id messageId, params []int) {
					if a == nil {
						return
					}
					if len(action.Params) != len(params) {
						fmt.Printf("ERROR: message `%s` in actor `%v` requires `%v` args\n",
							id, s.Ident.Value, len(action.Params))
						return
					}
					locals := map[string]int{}
					for pos, p := range action.Params {
						locals[p.Value] = params[pos]
					}
					ctx := newEvalCtx(action.Scope, (*a).state, locals)
					if err := ctx.eval(); err != nil {
						fmt.Printf("EVAL ERROR: %v \n", err)
						return
					}
					(*a).state = ctx.state
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
			a.Show()
		case parse.SendStmt:
			id := s.ActorIdent.Value
			a, defined := e.actors[id]
			if !defined {
				return fmt.Errorf("actor with name `%s` not defined yet", id)
			}
			params := []int{}
			for _, t := range s.Args {
				v, err := strconv.Atoi(t.Value)
				if err != nil {
					return err
				}
				params = append(params, v)
			}
			mId := messageId(s.Message.Value)
			if _, ok := a.actions[mId]; !ok {
				return fmt.Errorf("message `%v` for actor `%v` is not defined", mId, a.addr)
			}
			if err := a.Recv(messageId(s.Message.Value), params...); err != nil {
				return err
			}
		case parse.SpawnStmt:
			id := s.Scope[0].Value
			a, defined := e.actors[id]
			if !defined {
				return fmt.Errorf("actor with name `%s` not defined yet", id)
			}
			p := e.sched.startProc(s.PidIdent.Value, a)
			for i := 0; i < 10; i++ {
				p.inbox <- Message{messageId(fmt.Sprintf("foobar-%d", i)), []string{"Hello, World!"}}
			}
			// p.errs <- errors.New("you should be dead by now")
		default:
			panic(fmt.Sprintf("item `%v` is not supported yet", item))
		}

		e.sched.wg.Wait()
	}
	return nil
}
