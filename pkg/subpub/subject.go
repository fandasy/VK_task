package subpub

import (
	"fmt"
	"runtime/debug"
	"sync"
)

type subject struct {
	subscribers map[*subscription]MessageHandler
	queue       chan interface{}
	mu          sync.RWMutex

	wg *sync.WaitGroup // pointer to wg from subPub

	closed bool // true when chan queue is closed
}

func newSubject(queueBuf int, wg *sync.WaitGroup) *subject {
	return &subject{
		subscribers: make(map[*subscription]MessageHandler, 8),
		queue:       make(chan interface{}, queueBuf),
		wg:          wg,
	}
}

func (s *subject) registerSubscriber(sub *subscription, cb MessageHandler) {
	s.mu.Lock()
	s.subscribers[sub] = cb
	s.mu.Unlock()
}

func (s *subject) dispatchMessage(closeChan <-chan struct{}) {
	for {
		select {
		case msg, ok := <-s.queue:
			if !ok {
				return
			}
			s.deliverMessage(msg)
		case <-closeChan:
			return
		}
	}
}

func (s *subject) deliverMessage(msg interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, callback := range s.subscribers {
		s.wg.Add(1)
		go startCallback(msg, callback, s.wg)
	}
}

func startCallback(msg interface{}, cb MessageHandler, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
		if r := recover(); r != nil {
			fmt.Printf("Panic recovered: %v\nstack: %s", r, debug.Stack())
		}
	}()
	cb(msg)
}
