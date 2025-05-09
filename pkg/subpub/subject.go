package subpub

import (
	"sync"
)

type subject struct {
	subscribers map[string]*subscription
	queue       chan interface{}
	mu          sync.RWMutex

	closed bool // true when chan queue is closed
}

func newSubject(bufferSize int) *subject {
	return &subject{
		subscribers: make(map[string]*subscription, 8),
		queue:       make(chan interface{}, bufferSize),
	}
}

func (s *subject) registerSubscriber(sub *subscription) {
	s.mu.Lock()
	s.subscribers[sub.id] = sub
	s.mu.Unlock()
}

func (s *subject) unregisterSubscriber(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.subscribers, id)
	return len(s.subscribers) == 0
}

func (s *subject) publish(msg interface{}, closeChan <-chan struct{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Защита от паники при записи в закрытый queue канал
	if s.closed {
		return nil
	}

	select {
	case s.queue <- msg:
		return nil
	case <-closeChan:
		return ErrSubPubClosed
	}
}

func (s *subject) dispatchMessages(closeChan <-chan struct{}) {
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

	for _, sub := range s.subscribers {
		sub.deliver(msg)
	}
}

func (s *subject) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.closed = true
	close(s.queue)

	for _, sub := range s.subscribers {
		sub.clear()
	}

	s.subscribers = nil
}
