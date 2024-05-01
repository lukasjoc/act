package parse

import (
	"bufio"
	"fmt"

	"github.com/lukasjoc/act/internal/lex"
)

//go:generate stringer -type=ModuleItemType
type ModuleItemType int

const (
	ModuleItemActor ModuleItemType = iota
	ModuleItemShow
	ModuleItemSend
)

type ModuleItem interface {
	// FIXME: this can be used for type casting (i know its bad will cleanup later)
	Type() ModuleItemType
}

// TODO: fix this later (maybe)
type Module struct{ Items []ModuleItem }

type actStatementActionBody struct {
	Left  *lex.Token
	Right *lex.Token
}
type actStatementAction struct {
	Ident *lex.Token
	Arg   *lex.Token
	Body  actStatementActionBody
}
type ActorStmt struct {
	Ident   *lex.Token
	State   *lex.Token
	Actions []*actStatementAction
}

func (a ActorStmt) Type() ModuleItemType { return ModuleItemActor }

type SendStmt struct {
	ActorIdent *lex.Token
	Message    *lex.Token
	Op         *lex.Token
}

func (a SendStmt) Type() ModuleItemType { return ModuleItemSend }

type ShowStmt struct{ ActorIdent *lex.Token }

func (a ShowStmt) Type() ModuleItemType { return ModuleItemShow }

// TODO: cleanup to use tokenstream
// TODO: this must be the most horrible code 've ever written (this month)
func New(r *bufio.Reader) (*Module, error) {
	toks, err := lex.New(r)
	if err != nil {
		return nil, err
	}
	eatTok := func(tokens []*lex.Token, index *int) *lex.Token {
		prev := tokens[*index]
		*index++
		return prev
	}
	module := Module{Items: []ModuleItem{}}
	for at := 0; at < len(toks); {
		tok := toks[at]
		if tok.Typ == lex.TokenTypeKeywordActor {
			_ = eatTok(toks, &at) // skip 'actor' keyword
			ident := eatTok(toks, &at)
			state := eatTok(toks, &at)
			_ = eatTok(toks, &at) // skip 'assign'
			actions := []*actStatementAction{}
			for toks[at].Value != ";" {
				ident := eatTok(toks, &at)
				arg := eatTok(toks, &at)
				_ = eatTok(toks, &at) // skip 'paren'
				lhs := eatTok(toks, &at)
				rhs := eatTok(toks, &at)
				_ = eatTok(toks, &at) // skip 'paren'
				if toks[at].Value != ";" {
					_ = eatTok(toks, &at) // skip 'comma' if not 'semi'
				}
				action := &actStatementAction{ident, arg, actStatementActionBody{lhs, rhs}}
				actions = append(actions, action)
			}
			_ = eatTok(toks, &at) // skip 'semi'
			item := ActorStmt{ident, state, actions}
			module.Items = append(module.Items, item)
		} else if tok.Typ == lex.TokenTypeKeywordShow {
			_ = eatTok(toks, &at) // skip 'show' keyword
			actorIdent := eatTok(toks, &at)
			_ = eatTok(toks, &at) // skip 'semi'
			item := ShowStmt{actorIdent}
			module.Items = append(module.Items, item)
		} else if tok.Typ == lex.TokenTypeIdent {
			actorIdent := eatTok(toks, &at)
			_ = eatTok(toks, &at) // skip 'send symbol'
			message := eatTok(toks, &at)
			op := eatTok(toks, &at)
			_ = eatTok(toks, &at) // skip 'semi'
			item := SendStmt{actorIdent, message, op}
			module.Items = append(module.Items, item)
		} else {
      panic(fmt.Sprintf("PARSE: next token `%s` at index %d/%d` not allowed", tok, at, len(toks)))
		}
	}
	return &module, nil
}
