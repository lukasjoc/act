package runtime

import (
	"errors"
	"fmt"
	"sync"
)

const envCtxMessageLimit = 1024

type messageId string
type message struct {
	id   string
	args []int
}

type actionFunc func(*message)
type actor struct {
	name    string
	state   int
	actions map[messageId]actionFunc
}

func newActor(name string, state int) *actor {
	return &actor{name, state, map[messageId]actionFunc{}}
}

func (a *actor) setAction(id messageId, f actionFunc) {
	// TODO: figure out a way to move more stuff into this default impl
	a.actions[id] = func(m *message) { f(m) }
}

func (a *actor) show() { fmt.Printf("ACTOR STATE %v(%v)\n", a.name, a.state) }

type pid int

type proc struct {
	a     *actor
	pid   pid
	inbox chan *message
	errs  chan error
}

func newProc(a *actor, pid pid) *proc {
	return &proc{
		a:     a,
		pid:   pid,
		inbox: make(chan *message, envCtxMessageLimit),
		errs:  make(chan error, 1),
	}
}

func (p *proc) recv(m *message) { p.inbox <- m }

type scheduler struct {
	wg        sync.WaitGroup
	procs     map[string]*proc
	pidsource uint16
}

func newScheduler() *scheduler {
	return &scheduler{sync.WaitGroup{}, map[string]*proc{}, 1}
}

func (s *scheduler) killProc(p *proc) { p.errs <- errors.New("kill") }
func (s *scheduler) proc(name string) (*proc, error) {
	p, ok := s.procs[name]
	if !ok {
		return nil, errors.New("noproc")
	}
	return p, nil
}
func (s *scheduler) nextPid() pid {
	s.pidsource += 1
	return pid(s.pidsource)
}
func (s *scheduler) destroy(name string) {
	delete(s.procs, name)
}
func (s *scheduler) startProc(name string, a *actor) {
	// TODO: this is blocking for some reason the creation of other processies
	// need to investigate (it blocks until the timer returns)
	p := newProc(a, s.nextPid())
	s.procs[name] = p
	s.wg.Add(1)
	go func() {
		fmt.Printf("NEW PROC: %#v\n", p)
		for {
			defer s.destroy(name)
			defer s.wg.Done()
			select {
			case message := <-p.inbox:
				fmt.Printf("Message: %v\n", message)
				f, ok := p.a.actions[messageId(message.id)]
				if !ok {
					return
				}
				f(message)
                return
			case err := <-p.errs:
				fmt.Printf("ERROR in proc: %v\n", err)
				return
			default:
			}
		}
	}()
}
