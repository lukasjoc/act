package runtime

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/lukasjoc/act/internal/lex"
)

type evalCtx struct {
	stack  []int
	state  int
	tokens []*lex.Token
	locals map[string]int
}

// FIXME: later lex, parse these directly into the tokenstream
// for now we'll have to string match them by hand

//go:generate stringer -type=opType
type opType int

const (
	opTypeAddEq opType = iota
	opTypeSubEq
	opTypeModEq
	opTypeAssign
	opTypeMultEq
	opTypeMod
	opTypeAdd
	opTypeMult
	opTypeInvalid
)

var ops = map[opType]uint8{
	opTypeAddEq:  1,
	opTypeSubEq:  1,
	opTypeModEq:  1,
	opTypeMultEq: 1,
	opTypeAssign: 1,
	opTypeMod:    2,
	opTypeAdd:    2,
	opTypeMult:   2,
}

// TODO: it would be really cool if stringer could do this
func opTypeFromStr(opStr string) opType {
	switch opStr {
	case "+=":
		return opTypeAddEq
	case "-=":
		return opTypeSubEq
	case "%=":
		return opTypeModEq
	case "*=":
		return opTypeMultEq
	case "=":
		return opTypeAssign
	case "%":
		return opTypeMod
	case "+":
		return opTypeAdd
	case "*":
		return opTypeMult
	}
	return opTypeInvalid
}

func newEvalCtx(tokens []*lex.Token, state int, locals map[string]int) *evalCtx {
	return &evalCtx{[]int{}, state, tokens, locals}
}
func (ctx *evalCtx) push(value int) { ctx.stack = append(ctx.stack, value) }
func (ctx *evalCtx) pop(typ opType) ([]int, error) {
	c := ops[typ]
	if len(ctx.stack) < int(c) {
		return nil, fmt.Errorf("need stack height of %d but stack height is %d", c, len(ctx.stack))
	}
	popped := ctx.stack[:c]
	ctx.stack = ctx.stack[c:]
	return popped, nil
}

func (ctx *evalCtx) applyOp(typ opType, popped []int) error {
	if typ == opTypeInvalid {
		return errors.New("invalid op type for application")
	}
	if int(ops[typ]) != len(popped) {
		return errors.New("not enough pop values for type")
	}
	switch typ {
	case opTypeAddEq:
		ctx.state += popped[0]
	case opTypeSubEq:
		ctx.state -= popped[0]
	case opTypeModEq:
		ctx.state %= popped[0]
	case opTypeAssign:
		ctx.state = popped[0]
	case opTypeMultEq:
		ctx.state *= popped[0]
	case opTypeMod:
		ctx.state = popped[0] % popped[1]
	case opTypeAdd:
		ctx.state = popped[0] + popped[1]
	case opTypeMult:
		ctx.state = popped[0] * popped[1]
	}
	return nil
}

func (ctx *evalCtx) eval() error {
	for _, t := range ctx.tokens {
		switch t.Typ {
		case lex.TokenTypeIdent:
			lv, ok := ctx.locals[t.Value]
			if !ok {
				return fmt.Errorf("undefined local `%v` for ctx", t.Value)
			}
			ctx.push(lv)
		case lex.TokenTypeLit:
			value, err := strconv.Atoi(t.Value)
			if err != nil {
				return err
			}
			ctx.push(value)
		case lex.TokenTypeOp, lex.TokenTypeSymbol:
			typ := opTypeFromStr(t.Value)
			if typ == opTypeInvalid {
				return fmt.Errorf("eval of symbol `%v` not allowed", t.Value)
			}
			popped, err := ctx.pop(typ)
			if err != nil {
				return err
			}
			if err := ctx.applyOp(typ, popped); err != nil {
				return err
			}
		default:
			return fmt.Errorf("eval of token with type `%v` not allowed", t)
		}
	}
	return nil
}
