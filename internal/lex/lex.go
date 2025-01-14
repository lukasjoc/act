package lex

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode"
)

//go:generate stringer -type=TokenType
type TokenType int

const (
	TokenTypeKeywordActor TokenType = iota
	TokenTypeKeywordShow
	TokenTypeIdent
	TokenTypeLit
	TokenTypeOp
	TokenTypeSymbol
	TokenTypeInvalid
)

type Token struct {
	Typ   TokenType
	Value *string
}

func (t Token) String() string { return fmt.Sprintf("%v(`%s`)", t.Typ, *t.Value) }

func dropWhile(r *bufio.Reader, pred func(b byte) bool) (int, error) {
	n := 0
	for {
		b, err := r.ReadByte()
		if err != nil {
			return n, err
		}
		n += 1
		if pred(b) {
			break
		}
	}
	return n, nil
}

func eatWhile(r *bufio.Reader, pred func(b byte) bool) ([]byte, int, error) {
	buf := []byte{}
	for {
		b, err := r.Peek(1)
		if err != nil {
			return buf, len(buf), err
		}
		if pred(b[0]) {
			rb, err := r.ReadByte()
			if err != nil {
				return buf, len(buf), err
			}
			buf = append(buf, rb)
		} else {
			break
		}
	}
	return buf, len(buf), nil
}

func New(r *bufio.Reader) ([]*Token, error) {
	tokens := []*Token{}
	offset := 0
	for {
		buf, err := r.Peek(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return tokens, err
		}
		if buf[0] == '#' {
			n, err := dropWhile(r, func(b byte) bool { return b == '\n' || b == '\r' })
			if err != nil {
				return tokens, err
			}
			offset += n
		} else if unicode.IsSpace(rune(buf[0])) {
			n, err := dropWhile(r, func(b byte) bool { return unicode.IsSpace(rune(b)) })
			if err != nil {
				return tokens, err
			}
			offset += n
		} else if unicode.IsLetter(rune(buf[0])) {
			buf, n, err := eatWhile(r, func(b byte) bool { return unicode.IsLetter(rune(b)) || unicode.IsNumber(rune(b)) })
			if err != nil {
				return tokens, err
			}
			v := string(buf)
			tokens = append(tokens, &Token{TokenTypeIdent, &v})
			offset += n
		} else if unicode.IsNumber(rune(buf[0])) {
			buf, n, err := eatWhile(r, func(b byte) bool { return unicode.IsNumber(rune(b)) })
			if err != nil {
				return tokens, err
			}
			v := string(buf)
			tokens = append(tokens, &Token{TokenTypeLit, &v})
			offset += n
		} else if buf[0] == '=' || buf[0] == '}' || buf[0] == '{' || buf[0] == ',' || buf[0] == ';' || buf[0] == '@' {
			b, err := r.ReadByte()
			if err != nil {
				return tokens, err
			}
			v := string(b)
			tokens = append(tokens, &Token{TokenTypeSymbol, &v})
			offset += 1
		} else if buf[0] == '+' || buf[0] == '-' || buf[0] == '*' || buf[0] == '%' || buf[0] == '<' {
			buf, n, err := eatWhile(r, func(b byte) bool {
				return b == '+' || b == '-' || b == '*' || b == '%' || b == '<' || b == '>' || b == '='
			})
			if err != nil {
				return tokens, err
			}
			v := string(buf)
			tokens = append(tokens, &Token{TokenTypeOp, &v})
			offset += n
		} else {
			panic(fmt.Sprintf("LEX: next token `%s` at offset: %d not allowed", string(buf[0]), offset))
		}
	}
	for _, token := range tokens {
		if token.Typ != TokenTypeIdent {
			continue
		}
		if *token.Value == "actor" {
			token.Typ = TokenTypeKeywordActor
		}
		if *token.Value == "show" {
			token.Typ = TokenTypeKeywordShow
		}
	}
	return tokens, nil
}
