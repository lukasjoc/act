package runtime

import (
	"fmt"
	"math/rand/v2"
	"sync"
)

const envCtxMessageLimit = 1024

type Message struct {
	id   messageId
	args []string
}

type proc struct {
	*actor
	pid   int
	inbox chan Message
	errs  chan error
}

func newProc(a *actor) *proc {
	return &proc{
		actor: a,
		pid:   rand.N(100),
		inbox: make(chan Message, envCtxMessageLimit),
		errs:  make(chan error, 0),
	}
}

type scheduler struct {
	wg    sync.WaitGroup
	procs map[string]*proc
}

func newScheduler() *scheduler {
	return &scheduler{sync.WaitGroup{}, map[string]*proc{}}
}
func (s *scheduler) killProc(p *proc) {}
func (s *scheduler) startProc(name string, a *actor) *proc {
	p := newProc(a)
	s.procs[name] = p
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case message := <-p.inbox:
				fmt.Printf("Message: %v\n", message)
			case err := <-p.errs:
				fmt.Printf("ERROR in proc: %v\n", err)
				return
			default:
				fmt.Printf("PID %v is waiting.. Doodle.. Di dia doo\n", p.pid)
			}
		}
	}()

	return p
}

// func (s *scheduler) start() {
// 	fmt.Printf("START actor %v\n", ctx.addr)
// 	s.wg.Add(1)
// 	go func() {
// 		for {
// 			select {
// 			case message := <-ctx.inbox:
// 			case <-ctx.errs:
// 				return
// 			}
// 		}
// 	}()
// 	fmt.Println("??")
// }

// func (ctx *execCtx) kill() {
// 	go func() { ctx.errs <- errors.New("kill"); return }()
// }
