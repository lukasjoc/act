package parse

import (
	"bufio"
	"errors"
	"fmt"
	"io"

	"github.com/lukasjoc/act/internal/lex"
)

type Module []any

type ActorActionStmt struct {
	Ident     *lex.Token
	Params    []*lex.Token
	Scope     []*lex.Token
	ReturnPid *lex.Token
}
type ActorStmt struct {
	Ident   *lex.Token
	State   *lex.Token
	Actions []*ActorActionStmt
}

type SendStmt struct {
	SendPid *lex.Token
	Message *lex.Token
	Args    []*lex.Token
}

type SpawnStmt struct {
	PidIdent *lex.Token
	Scope    []*lex.Token
}

// FIXME: cleanup to use tokenstream
// FIXME: this must be the most horrible code 've ever written (this month)

func eatToken(tokens *[]*lex.Token, index *int) *lex.Token {
	prev := (*tokens)[*index]
	*index++
	return prev
}

func peekToken(tokens *[]*lex.Token, index *int) (*lex.Token, error) {
	p := (*index) + 1
	if p >= len(*tokens) {
		return nil, io.EOF
	}
	return (*tokens)[p], nil
}

func eatTokenAs(s string, tokens *[]*lex.Token, index *int) *lex.Token {
	prev := (*tokens)[*index]
	v := *(*tokens)[*index].Value
	if v != s {
		panic(fmt.Errorf("expected `%v` but instead got `%v`", s, v))
	}
	*index++
	return prev
}

func parseAction(tokens *[]*lex.Token, index *int) *ActorActionStmt {
	ident := eatToken(tokens, index)
	params := []*lex.Token{}
	for *(*tokens)[*index].Value != "{" {
		param := eatToken(tokens, index)
		params = append(params, param)
	}
	eatTokenAs("{", tokens, index)
	scope := []*lex.Token{}
	for *(*tokens)[*index].Value != "}" {
		tok := eatToken(tokens, index)
		scope = append(scope, tok)
	}
	eatTokenAs("}", tokens, index)
	var returnPid *lex.Token
	if *(*tokens)[*index].Value == "->" {
		eatTokenAs("->", tokens, index)
		returnPid = eatToken(tokens, index)
	}
	return &ActorActionStmt{ident, params, scope, returnPid}
}

func parseActor(tokens *[]*lex.Token, index *int) *ActorStmt {
	eatTokenAs("actor", tokens, index)
	ident := eatToken(tokens, index)
	state := eatToken(tokens, index)
	eatTokenAs("=", tokens, index)
	actions := []*ActorActionStmt{}
	for {
		a := parseAction(tokens, index)
		actions = append(actions, a)
		if *(*tokens)[*index].Value != "," {
			break
		}
		eatTokenAs(",", tokens, index)
	}
	eatTokenAs(";", tokens, index)
	return &ActorStmt{ident, state, actions}
}

func parseSend(tokens *[]*lex.Token, index *int) *SendStmt {
	actorIdent := eatToken(tokens, index)
	eatTokenAs("<-", tokens, index)
	message := eatToken(tokens, index)
	params := []*lex.Token{}
	for *(*tokens)[*index].Value != ";" {
		params = append(params, eatToken(tokens, index))
	}
	eatTokenAs(";", tokens, index)
	return &SendStmt{actorIdent, message, params}
}

func parseSpawn(tokens *[]*lex.Token, index *int) *SpawnStmt {
	pidIdent := eatToken(tokens, index)
	eatTokenAs("=", tokens, index)
	eatTokenAs("spawn", tokens, index)
	eatTokenAs("{", tokens, index)
	scope := []*lex.Token{}
	for *(*tokens)[*index].Value != "}" {
		tok := eatToken(tokens, index)
		scope = append(scope, tok)
	}
	eatTokenAs("}", tokens, index)
	eatTokenAs(";", tokens, index)
	return &SpawnStmt{pidIdent, scope}
}
func New(r *bufio.Reader) (module Module, err error) {
	tokens, err := lex.New(r)
	if err != nil {
		return nil, err
	}
	for index := 0; index < len(tokens); {
		switch tokens[index].Typ {
		case lex.TokenTypeKeywordActor:
			actor := parseActor(&tokens, &index)
			module = append(module, *actor)
		case lex.TokenTypeIdent:
			p, err := peekToken(&tokens, &index)
			if err != nil && errors.Is(err, io.EOF) {
				return module, nil
			}
			if *p.Value == "<-" {
				send := parseSend(&tokens, &index)
				module = append(module, *send)
			} else if *p.Value == "=" {
				spawn := parseSpawn(&tokens, &index)
				module = append(module, *spawn)
			}
		default:
			panic(fmt.Sprintf("PARSE: next token `%s` at index %d/%d` not allowed",
				tokens[index], index, len(tokens)))
		}
	}
	return module, nil
}
