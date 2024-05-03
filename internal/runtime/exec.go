package runtime

import (
	"errors"
	"fmt"
	"sync"
)

const envCtxMessageLimit = 1024

type messageId string
type message struct {
	id   messageId
	args []int
}

type actor struct {
	name    string
	state   int
	actions map[messageId]func(*message)
}

func newActor(name string, state int) *actor {
	return &actor{name, state, map[messageId]func(*message){}}
}

func (a *actor) setAction(id messageId, f func(*message)) {
	// TODO: figure out a way to move more stuff into this default impl
	a.actions[id] = func(m *message) { f(m) }
}

func (a *actor) show() { fmt.Printf("ACTOR STATE %v(%v)\n", a.name, a.state) }

type pid int

type proc struct {
	a     *actor
	pid   pid
	dirty bool
	inbox chan *message
}

func newProc(a *actor, pid pid) *proc {
	return &proc{
		a:     a,
		pid:   pid,
		inbox: make(chan *message, envCtxMessageLimit),
		// TODO: add error sync channel to sync exiting process go routines with
		// the outer world and have a ticker that cleans up stuck ones etc..
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
	p := s.procs[name]
	close(p.inbox)
	delete(s.procs, name)
}
func (s *scheduler) startProc(name string, a *actor) {
	s.procs[name] = newProc(a, s.nextPid())
	s.wg.Add(1)
	go func(name string) {
		defer s.destroy(name)
		defer s.wg.Done()
		p, err := s.proc(name)
		if err != nil {
			return
		}
		fmt.Printf("NEW PROC: PID:%v, actor:%v\n", p.pid, p.a.name)
		for {
			select {
			case message := <-p.inbox:
				p.dirty = true
				fmt.Printf("PROC[%v]: %v\n", p.pid, message)
				f, ok := p.a.actions[messageId(message.id)]
				if !ok {
					return
				}
				f(message)
			default:
				if p.dirty {
					return
				}
			}
		}
	}(name)
}
