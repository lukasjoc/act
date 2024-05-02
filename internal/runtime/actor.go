package runtime

import "fmt"

type messageId string

type actionFunc func(*actor, messageId, []int)

type actor struct {
	addr    string
	state   int
	actions map[messageId]actionFunc
}

func newActor(addr string, state int) *actor {
	return &actor{addr, state, map[messageId]actionFunc{}}
}

func (a *actor) With(id messageId, f actionFunc) { a.actions[id] = f }
func (a *actor) Recv(id messageId, params ...int) error {
	f, exists := a.actions[id]
	if !exists {
		return nil
	}
	f(a, id, params)
	return nil
}
func (a *actor) Show() { fmt.Printf("ACTOR STATE %v(%v)\n", a.addr, a.state) }
