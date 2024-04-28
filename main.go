package main

import "fmt"

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
func (a *actor) show() { fmt.Printf("ACTOR STATE %v\n", a.state) }

func main() {
	foo := actor{0, map[messageId]actionFunc{}}
	foo.with(messageIdAdd, func(a *actor, id messageId, op *int) {
		if a == nil || op == nil {
			return
		}
		(*a).state += *op
	})
	foo.with(messageIdSub, func(a *actor, id messageId, op *int) {
		if a == nil || op == nil {
			return
		}
		(*a).state -= *op
	})
	foo.with(messageIdMult, func(a *actor, id messageId, op *int) {
		if a == nil || op == nil {
			return
		}
		(*a).state *= *op
	})

	foo.show()

	op := 1
	foo.recv(messageIdAdd, &op)
	foo.show()

	foo.recv(messageIdSub, &op)
	foo.show()
}
