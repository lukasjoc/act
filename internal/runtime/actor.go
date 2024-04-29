package runtime

import "fmt"

type MessageId int

type ActionFunc func(*Actor, MessageId, *int)

type Actor struct {
	state   int
	actions map[MessageId]ActionFunc
}

func (a *Actor) With(id MessageId, f ActionFunc) { a.actions[id] = f }
func (a *Actor) Recv(id MessageId, op *int) error {
	f, exists := a.actions[id]
	if !exists {
		return nil
	}
	f(a, id, op)
	return nil
}

func (a *Actor) Show() { fmt.Printf("ACTOR STATE %v", a.state) }
