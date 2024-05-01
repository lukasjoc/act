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
}

// FIXME: later lex, parse these directly into the tokenstream
// for now we'll have to string match them by hand
type opType int

const (
	opTypeAddEq opType = iota
	opTypeSubEq
	opTypeModEq
	opTypeAssign
	opTypeMod
	opTypeInvalid
)

var ops = map[opType]uint8{
	opTypeAddEq:  1,
	opTypeSubEq:  1,
	opTypeModEq:  1,
	opTypeAssign: 1,
	opTypeMod:    2,
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
	case "=":
		return opTypeAssign
	case "%":
		return opTypeMod
	}
	return opTypeInvalid
}

func newEvalCtx(tokens []*lex.Token, state int) *evalCtx {
	return &evalCtx{[]int{}, state, tokens}
}
func (ctx *evalCtx) push(value int) { ctx.stack = append(ctx.stack, value) }
func (ctx *evalCtx) pop(typ opType) ([]int, error) {
	c := ops[typ]
	if len(ctx.stack) < int(c) {
		return nil, fmt.Errorf("Need stack height of %d but stack height is %d", c, len(ctx.stack))
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
	case opTypeMod:
		ctx.state = popped[0] % popped[1]
	}
	return nil
}

func (ctx *evalCtx) eval() error {
	for _, t := range ctx.tokens {
		switch t.Typ {
		case lex.TokenTypeLit:
			value, err := strconv.Atoi(t.Value)
			if err != nil {
				return err
			}
			ctx.push(value)
		case lex.TokenTypeOp, lex.TokenTypeIdent, lex.TokenTypeSymbol:
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
