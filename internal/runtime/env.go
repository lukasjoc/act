package runtime

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/lukasjoc/act/internal/lex"
	"github.com/lukasjoc/act/internal/parse"
)

type Env struct {
	module     parse.Module
	sched      *scheduler
	actorDefs  map[string]int
	actorCache map[string]actor
}

func New(module parse.Module) *Env {
	sched := newScheduler()
	return &Env{module, sched, map[string]int{}, map[string]actor{}}
}

func (e *Env) newActor(s *parse.ActorStmt) (*actor, error) {
	name := *s.Ident.Value
	if a, ok := e.actorCache[name]; ok {
		return &a, nil
	}
	state := 0
	if s.State.Typ == lex.TokenTypeLit {
		s, err := strconv.Atoi(*s.State.Value)
		if err != nil {
			return nil, err
		}
		state = s
	}
	a := actor{name, state, map[messageId]actionFunc{}}
	for _, action := range s.Actions {
		action := action
		id := messageId(*action.Ident.Value)
		a.setAction(id, newActionFunc(action))
	}
	e.actorCache[name] = a
	return &a, nil
}

func (e *Env) newScopedActor(scope []string) (*actor, error) {
	name := scope[0]
	defIndex, foundDef := e.actorDefs[name]
	if !foundDef {
		return nil, errors.New("actor not defined")
	}
	s := e.module[defIndex].(parse.ActorStmt)
	a, err := e.newActor(&s)
	if err != nil {
		return nil, err
	}
	if len(scope) > 1 {
		state, err := strconv.Atoi(scope[1])
		if err != nil {
			return nil, err
		}
		a.state = state
	}
	return a, nil
}

func (e *Env) Exec() error {
	atProc := e.sched.startAtProc()
	<-atProc.ok
	for i, item := range e.module {
		switch s := item.(type) {
		case parse.ActorStmt:
			e.actorDefs[*s.Ident.Value] = i
		case parse.SendStmt:
			if s.SendPid == nil {
				return errors.New("need sendTo pid ident in send")
			}
			p, err := e.sched.procByName(*s.SendPid.Value)
			if err != nil {
				return err
			}
			args := []int{}
			for _, t := range s.Args {
				// TODO: dont refer to `@` here but instead evaluate from current execEnv
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
			spawnScope := []string{}
			for _, scopeVal := range s.Scope {
				spawnScope = append(spawnScope, *scopeVal.Value)
			}
			if len(spawnScope) == 0 {
				return errors.New("need actor name to spawn")
			}
			a, err := e.newScopedActor(spawnScope)
			if err != nil {
				return err
			}
			e.sched.startProc(*s.PidIdent.Value, a)
		default:
			panic(fmt.Sprintf("item `%v` is not supported yet", item))
		}
	}

	e.sched.wg.Wait()
	return nil
}
