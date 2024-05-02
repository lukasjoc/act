package parse

import (
	"bufio"
	"fmt"

	"github.com/lukasjoc/act/internal/lex"
)

type Module []any

type actStatementAction struct {
	Ident  *lex.Token
	Params []*lex.Token
	Body   []*lex.Token
}
type ActorStmt struct {
	Ident   *lex.Token
	State   *lex.Token
	Actions []*actStatementAction
}

type SendStmt struct {
	ActorIdent *lex.Token
	Message    *lex.Token
	Args       []*lex.Token
}

type ShowStmt struct {
	ActorIdent *lex.Token
}

// FIXME: cleanup to use tokenstream
// FIXME: this must be the most horrible code 've ever written (this month)

func eatToken(tokens *[]*lex.Token, index *int) *lex.Token {
	prev := (*tokens)[*index]
	*index++
	return prev
}

func eatTokenAs(s string, tokens *[]*lex.Token, index *int) *lex.Token {
	prev := (*tokens)[*index]
	v := (*tokens)[*index].Value
	if v != s {
		panic(fmt.Errorf("expected `%v` but instead got `%v`", s, v))
	}
	*index++
	return prev
}

func parseAction(tokens *[]*lex.Token, index *int) *actStatementAction {
	ident := eatToken(tokens, index)
	params := []*lex.Token{}
	for (*tokens)[*index].Value != "{" {
		param := eatToken(tokens, index)
		params = append(params, param)
	}
	eatTokenAs("{", tokens, index)
	body := []*lex.Token{}
	for (*tokens)[*index].Value != "}" {
		tok := eatToken(tokens, index)
		body = append(body, tok)
	}
	eatTokenAs("}", tokens, index)
	return &actStatementAction{ident, params, body}
}
func New(r *bufio.Reader) (module Module, err error) {
	tokens, err := lex.New(r)
	if err != nil {
		return nil, err
	}
    for index := 0; index < len(tokens); {
		token := tokens[index]
		if token.Typ == lex.TokenTypeKeywordActor {
			eatTokenAs("actor", &tokens, &index)
			ident := eatToken(&tokens, &index)
			state := eatToken(&tokens, &index)
			eatTokenAs("=", &tokens, &index)
			actions := []*actStatementAction{}
			for {
				a := parseAction(&tokens, &index)
				actions = append(actions, a)
				if tokens[index].Value != "," {
					break
				}
				eatTokenAs(",", &tokens, &index)
			}
			eatTokenAs(";", &tokens, &index)
			item := ActorStmt{ident, state, actions}
			module = append(module, item)
		} else if token.Typ == lex.TokenTypeKeywordShow {
			eatTokenAs("show", &tokens, &index)
			actorIdent := eatToken(&tokens, &index)
			eatTokenAs(";", &tokens, &index)
			item := ShowStmt{actorIdent}
			module = append(module, item)
		} else if token.Typ == lex.TokenTypeIdent {
			actorIdent := eatToken(&tokens, &index)
			eatTokenAs("<-", &tokens, &index)
			message := eatToken(&tokens, &index)
			params := []*lex.Token{}
			for tokens[index].Value != ";" {
				params = append(params, eatToken(&tokens, &index))
			}
			eatTokenAs(";", &tokens, &index)
			item := SendStmt{actorIdent, message, params}
			module = append(module, item)
		} else {
			panic(fmt.Sprintf("PARSE: next token `%s` at index %d/%d` not allowed", token, index, len(tokens)))
		}
	}
	return module, nil
}
