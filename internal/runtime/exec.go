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

type actionFunc func(*message) int
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
	a.actions[id] = f
}

func (a *actor) show() { fmt.Printf("ACTOR STATE %v(%v)\n", a.name, a.state) }

type pid uint16

type proc struct {
	a     *actor
	pid   pid
	dirty bool
	ok    chan bool
	inbox chan *message
}

func newProc(a *actor, pid pid) *proc {
	return &proc{
		a:     a,
		pid:   pid,
		inbox: make(chan *message, envCtxMessageLimit),
		ok:    make(chan bool, 1),
	}
}

func (p *proc) recv(m *message) { p.inbox <- m }

type scheduler struct {
	wg        sync.WaitGroup
	procs     map[string]*proc
	pidsource uint16
	// TODO: add error sync channel to sync exiting process go routines with
	// the outer world and have a ticker that cleans up stuck ones etc..
}

const pid1 = 1

func newScheduler() *scheduler {
	return &scheduler{sync.WaitGroup{}, map[string]*proc{}, pid1}
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
func (s *scheduler) startAtProc() {
	s.procs["@"] = newProc(&actor{}, pid1)
	s.wg.Add(1)
	go func() {
		defer s.destroy("@")
		defer s.wg.Done()
		p, err := s.proc("@")
		if err != nil {
			return
		}
		fmt.Printf("NEW PROC: PID:%v, actor:%v\n", p.pid, p.a.name)
		p.ok <- true
		for {
			select {
			case message := <-p.inbox:
				p.dirty = true
				fmt.Printf("Received: %v\n", message)
			default:
			}
		}
	}()
}
func (s *scheduler) destroy(name string) {
	p := s.procs[name]
	fmt.Printf("PROC[%v] DESTROY \n", p.pid)
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
		p.ok <- true
		for {
			select {
			case m := <-p.inbox:
				p.dirty = true
				fmt.Printf("PROC[%v]: %v\n", p.pid, m)
				f, ok := p.a.actions[messageId(m.id)]
				if !ok {
					return
				}
				if pid := f(m); pid > 0 {
					fromProc, _ := s.proc("@")
					fromProc.recv(&message{m.id, []int{int(p.pid), int(pid), p.a.state}})
				}
				return
			default:
				if p.dirty {
					return
				}
			}
		}
	}(name)
}
