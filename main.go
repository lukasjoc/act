package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"unicode"
)

type messageId int

const (
	messageIdAdd messageId = iota
	messageIdSub
	messageIdMult
)

type actionFunc func(*actor, messageId, *int)

type actor struct {
	state   int
	actions map[messageId]actionFunc
}

func (a *actor) with(id messageId, f actionFunc) { a.actions[id] = f }
func (a *actor) recv(id messageId, op *int) error {
	f, exists := a.actions[id]
	if !exists {
		return nil
	}
	f(a, id, op)
	return nil
}
func (a *actor) show() { fmt.Printf("ACTOR STATE %v", a.state) }

// func main() {
// 	foo := actor{0, map[messageId]actionFunc{}}
// 	foo.with(messageIdAdd, func(a *actor, id messageId, op *int) {
// 		if a == nil || op == nil {
// 			return
// 		}
// 		(*a).state += *op
// 	})
// 	foo.with(messageIdSub, func(a *actor, id messageId, op *int) {
// 		if a == nil || op == nil {
// 			return
// 		}
// 		(*a).state -= *op
// 	})
// 	foo.with(messageIdMult, func(a *actor, id messageId, op *int) {
// 		if a == nil || op == nil {
// 			return
// 		}
// 		(*a).state *= *op
// 	})
//
// 	foo.show()
//
// 	op := 1
// 	foo.recv(messageIdAdd, &op)
// 	foo.show()
//
// 	foo.recv(messageIdSub, &op)
// 	foo.show()
// }

func dropWhile(r *bufio.Reader, pred func(b byte) bool) error {
	for {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		if pred(b) {
			break
		}
	}
	return nil
}

func eatWhile(r *bufio.Reader, pred func(b byte) bool) ([]byte, error) {
	buf := []byte{}
	for {
		b, err := r.Peek(1)
		if err != nil {
			return buf, err
		}
		if pred(b[0]) {
			rb, err := r.ReadByte()
			if err != nil {
				return buf, err
			}
			buf = append(buf, rb)
		} else {
			break
		}
	}
	return buf, nil
}

//go:generate stringer -type=tokenType
type tokenType int

const (
	tokenTypeKeywordActor tokenType = iota
	tokenTypeKeywordShow
	tokenTypeIdent
	tokenTypeLit
	tokenTypeOp
	tokenTypeSymbol
	tokenTypeInvalid
)

var supportedOp = map[string]bool{"+": true, "-": true, "*": true, "<-": true}
var supportedSymbol = map[string]bool{".": true, ";": true}

type token struct {
	typ tokenType
	// TODO: use span isntead
	value string
}

func (t token) String() string { return fmt.Sprintf("%s(`%s`)", t.typ, t.value) }

// TODO: cleanup
func tokenize(r *bufio.Reader) ([]*token, error) {
	tokens := []*token{}
	for {
		buf, err := r.Peek(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return tokens, err
		}
		if buf[0] == '#' {
			err := dropWhile(r, func(b byte) bool { return b == '\n' || b == '\r' })
			if err != nil {
				return tokens, err
			}
		} else if unicode.IsSpace(rune(buf[0])) {
			err := dropWhile(r, func(b byte) bool { return unicode.IsSpace(rune(b)) })
			if err != nil {
				return tokens, err
			}
		} else if unicode.IsLetter(rune(buf[0])) {
			buf, err := eatWhile(r, func(b byte) bool { return unicode.IsLetter(rune(b)) })
			if err != nil {
				return tokens, err
			}
			tokens = append(tokens, &token{tokenTypeIdent, string(buf)})
		} else if unicode.IsNumber(rune(buf[0])) {
			buf, err := eatWhile(r, func(b byte) bool { return unicode.IsNumber(rune(b)) })
			if err != nil {
				return tokens, err
			}
			tokens = append(tokens, &token{tokenTypeLit, string(buf)})
		} else if buf[0] == '=' || buf[0] == '}' || buf[0] == '{' || buf[0] == ',' || buf[0] == ';' {
			b, err := r.ReadByte()
			if err != nil {
				return tokens, err
			}
			tokens = append(tokens, &token{tokenTypeSymbol, string(b)})
		} else if buf[0] == '+' || buf[0] == '-' || buf[0] == '*' {
			b, err := r.ReadByte()
			if err != nil {
				return tokens, err
			}
			tokens = append(tokens, &token{tokenTypeOp, string(b)})
		} else if buf[0] == '<' {
			buf2, err := r.Peek(2)
			if err != nil {
				return tokens, err
			}
			if buf2[1] == '-' {
				b1, err := r.ReadByte()
				if err != nil {
					return tokens, err
				}
				b2, err := r.ReadByte()
				if err != nil {
					return tokens, err
				}
				tokens = append(tokens, &token{tokenTypeOp, string([]byte{b1, b2})})
			}
		} else {
			panic(fmt.Sprintf("token `%s` not implemented yet", string(buf[0])))
		}
	}
	//clean up
	for _, token := range tokens {
		if token.typ == tokenTypeIdent && token.value == "actor" {
			token.typ = tokenTypeKeywordActor
		}
		if token.typ == tokenTypeIdent && token.value == "show" {
			token.typ = tokenTypeKeywordShow
		}
	}
	return tokens, nil
}

// actor keyword, ident, literal start, = symbol,
// list of (message name, ?operands args, { operator  operand name } )
// actor definition

// show keyword, ident, semi
// show statement

// ident, <- symbol, message, operands?, semi
// send statement

type actStatementActionBody struct {
	lhs *token
	rhs *token
}
type actStatementAction struct {
	ident *token
	arg   *token
	body  actStatementActionBody
}

type actStatementActor struct {
	ident   *token
	state   *token
	actions []*actStatementAction
}

type actStatementSend struct {
	actorIdent *token
	message    *token
	op         *token
}

type actStatementShow struct{ actorIdent *token }

// TODO: fix this later (maybe)
type act struct{ items []any }

func parse(r *bufio.Reader) (*act, error) {
	a := act{items: []any{}}
	tokens, err := tokenize(r)
	if err != nil {
		return nil, err
	}
	eatToken := func(tokens []*token, index *int) *token {
		prev := tokens[*index]
		*index++
		return prev
	}

	// TODO: cleanup to use tokenstream
	// TODO: this must be the most horrible code 've ever written (this month)
	index := 0
	for index < len(tokens) {
		token := tokens[index]
		if token.typ == tokenTypeKeywordActor {
			_ = eatToken(tokens, &index) // skip 'actor' keyword
			ident := eatToken(tokens, &index)
			state := eatToken(tokens, &index)
			_ = eatToken(tokens, &index) // skip 'assign'
			actions := []*actStatementAction{}
			for tokens[index].value != ";" {
				ident := eatToken(tokens, &index)
				arg := eatToken(tokens, &index)
				_ = eatToken(tokens, &index) // skip 'paren'
				lhs := eatToken(tokens, &index)
				rhs := eatToken(tokens, &index)
				_ = eatToken(tokens, &index) // skip 'paren'
				if tokens[index].value != ";" {
					_ = eatToken(tokens, &index) // skip 'comma' if not 'semi'
				}
				action := &actStatementAction{ident, arg, actStatementActionBody{lhs, rhs}}
				actions = append(actions, action)
			}
			_ = eatToken(tokens, &index) // skip 'semi'
			item := actStatementActor{ident, state, actions}
			a.items = append(a.items, item)
		} else if token.typ == tokenTypeKeywordShow {
			_ = eatToken(tokens, &index) // skip 'show' keyword
			actorIdent := eatToken(tokens, &index)
			_ = eatToken(tokens, &index) // skip 'semi'
			item := actStatementShow{actorIdent}
			a.items = append(a.items, item)
		} else if token.typ == tokenTypeIdent {
			actorIdent := eatToken(tokens, &index)
			_ = eatToken(tokens, &index) // skip 'send symbol'
			message := eatToken(tokens, &index)
			op := eatToken(tokens, &index)
			_ = eatToken(tokens, &index) // skip 'semi'
			item := actStatementSend{actorIdent, message, op}
			a.items = append(a.items, item)
		} else {
			panic(fmt.Sprintf("token `%s` at index `%d/%d` not supported yet", token, index, len(tokens)))
		}
	}
	return &a, nil
}

func main() {
	// !TODO: better filepath validation of input
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %v ./path/to/file.act\n", os.Args[0])
		os.Exit(1)
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	act, err := parse(bufio.NewReader(f))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Could not parse file: %v\n", err)
	}
	for _, item := range (*act).items {
		fmt.Println(item)
	}
}
