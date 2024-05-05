package runtime

import (
	"errors"
	"fmt"
	"sync"
)

type messageId string
type message struct {
	id   messageId
	args []int
}

type actionFunc func(*message) uint16
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

type proc struct {
	name string
	pid  uint16

	// TODO: move this to proper state handling
	dirty bool
	ok    chan bool

	inbox chan *message
	a     *actor
}

func newProc(a *actor, name string, pid uint16) *proc {
	return &proc{
		name,
		pid,
		false,
		make(chan bool, 1),
		make(chan *message, 1),
		a,
	}
}

func (p *proc) recv(m *message) { p.inbox <- m }

type scheduler struct {
	wg        sync.WaitGroup
	procs     map[uint16]*proc
	pidsource uint16
	// TODO: add error sync channel to sync exiting process go routines with
	// the outer world and have a ticker that cleans up stuck ones etc..
}

const atPid uint16 = 1

func newScheduler() *scheduler {
	return &scheduler{sync.WaitGroup{}, map[uint16]*proc{}, atPid}
}

func (s *scheduler) procByName(name string) (*proc, error) {
	for _, p := range s.procs {
		if p.name == name {
			return p, nil
		}
	}
	return nil, errors.New("proc with name not found")
}
func (s *scheduler) proc(pid uint16) (*proc, error) {
	for _, p := range s.procs {
		if p.pid == pid {
			return p, nil
		}
	}
	return nil, errors.New("proc with pid not found")
}
func (s *scheduler) nextPid() uint16 {
	s.pidsource += 1
	return uint16(s.pidsource)
}
func (s *scheduler) startAtProc() *proc {
	p := newProc(&actor{}, "@", atPid)
	s.procs[atPid] = p
	s.wg.Add(1)
	go func(p *proc) {
		defer s.destroy(atPid)
		defer s.wg.Done()
		fmt.Printf("NEW PROC: PID:%v, actor:%v\n", p.pid, p.a.name)
		p.ok <- true
		for {
			select {
			case message := <-p.inbox:
				p.dirty = true
				fmt.Printf("Received: %v\n", message)
			default:
				// TODO: find a good way to stop the initial process
				// i was thinking to kill this at the end (manually)
			}
		}
	}(p)
	return p
}
func (s *scheduler) destroy(pid uint16) {
	p := s.procs[pid]
	fmt.Printf("PROC[%v] DESTROY \n", p.pid)
	close(p.inbox)
	delete(s.procs, pid)
}
func (s *scheduler) startProc(name string, a *actor) {
	pid := s.nextPid()
	p := newProc(a, name, pid)
	s.procs[pid] = p
	s.wg.Add(1)
	go func(p *proc) {
		defer s.destroy(pid)
		defer s.wg.Done()
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
				if returnPid := f(m); returnPid > 0 {
					fromProc, _ := s.proc(returnPid)
					resPackage := []int{int(p.pid), int(returnPid), p.a.state}
					fromProc.recv(&message{m.id, resPackage})
				}
				return
			default:
				if p.dirty {
					return
				}
			}
		}
	}(p)
}
