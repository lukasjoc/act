package runtime

import (
	"fmt"
	"strconv"

	"github.com/lukasjoc/act/internal/parse"
)

type Env struct {
	Module *parse.Module
	actors map[string]*Actor
}

func New(module *parse.Module) *Env {
	return &Env{Module: module, actors: map[string]*Actor{}}
}

func (e *Env) Exec() error {
	for _, item := range (e.Module).Items {
		switch item.Type() {
		case parse.ModuleItemActor:
			s := item.(parse.ActorStmt)
			state, err := strconv.Atoi(s.State.Value)
			if err != nil {
				return err
			}
			actor := NewActor(int(state))
			for _, action := range s.Actions {
				id := MessageId(action.Ident.Value)
				actor.With(id, func(a *Actor, id MessageId, op *int) {
					if a == nil || op == nil {
						return
					}
					switch action.Body.Left.Value {
					case "+":
						(*a).state += *op
					case "-":
						(*a).state -= *op
					case "*":
						(*a).state *= *op
					}
				})
			}
			// fmt.Println(s.Ident.Value, e.actors)
			if _, defined := e.actors[s.Ident.Value]; defined {
				return fmt.Errorf("actor with name `%s` is already defined", s.Ident.Value)
			}
			e.actors[s.Ident.Value] = actor
			fmt.Printf("NEW ACTOR: %v [%v %v]\n", s.Ident.Value, actor.state, actor.actions)
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
			op, err := strconv.Atoi(s.Op.Value)
			if err != nil {
				return err
			}
			if err := a.Recv(MessageId(s.Message.Value), &op); err != nil {
				return err
			}
		default:
			panic(fmt.Sprintf("item `%v` is not supported yet", item.Type()))
		}
	}
	return nil
}
