package runtime

import (
	"fmt"
	"strconv"

	"github.com/lukasjoc/act/internal/lex"
	"github.com/lukasjoc/act/internal/parse"
)

type Env struct {
	module *parse.Module
	actors map[string]*actor
}

func New(module *parse.Module) *Env { return &Env{module, map[string]*actor{}} }
func (e *Env) Exec() error {
	for _, item := range (e.module).Items {
		switch item.Type() {
		case parse.ModuleItemActor:
			s := item.(parse.ActorStmt)
			state, err := strconv.Atoi(s.State.Value)
			if err != nil {
				return err
			}
			a := newActor(int(state))
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
					for pos, ident := range action.Params {
						// locals[ident.Value] = params[pos]
						action.Body[pos].Typ = lex.TokenTypeLit
						action.Body[pos].Value = strconv.Itoa(params[pos])
						fmt.Printf("LOCAL: %v=%v\n", ident.Value, params[pos])
					}
					ctx := newEvalCtx(action.Body, (*a).state)
					if err := ctx.eval(); err != nil {
						fmt.Printf("EVAL ERROR: %v \n", err)
						return
					}
					(*a).state = ctx.state
				})
			}
			// fmt.Println(s.Ident.Value, e.actors)
			if _, defined := e.actors[s.Ident.Value]; defined {
				return fmt.Errorf("actor with name `%s` is already defined", s.Ident.Value)
			}
			e.actors[s.Ident.Value] = a
			fmt.Printf("NEW ACTOR: %v [%v %v]\n", s.Ident.Value, a.state, a.actions)
			// for n, a := range e.actors {
			// 	fmt.Printf("RUNTIME DUMP: ACTION %v [%v %v]\n", n, a.state, a.actions)
			// }
		case parse.ModuleItemShow:
			s := item.(parse.ShowStmt)
			id := s.ActorIdent.Value
			a, defined := e.actors[id]
			if !defined {
				return fmt.Errorf("actor with name `%s` not defined yet", id)
			}
			a.Show()
		case parse.ModuleItemSend:
			s := item.(parse.SendStmt)
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
			if err := a.Recv(messageId(s.Message.Value), params...); err != nil {
				return err
			}
		default:
			panic(fmt.Sprintf("item `%v` is not supported yet", item.Type()))
		}
	}
	return nil
}
