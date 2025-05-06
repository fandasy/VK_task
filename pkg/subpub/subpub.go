package subpub

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type subPub struct {
	subjects map[string]*subject
	mu       sync.RWMutex

	closed    atomic.Bool // true when subPub is closed
	closeChan chan struct{}

	wg sync.WaitGroup // MessageHandler WaitGroup

	cfg Config
}

var (
	ErrNoSuchSubject   = errors.New("no such subject")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrSubPubClosed    = errors.New("subPub system is closed")
)

/*
Subscribe

Если subject не существует, он будет создан.
*/
func (sp *subPub) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	if sp.closed.Load() {
		return nil, ErrSubPubClosed
	}

	if subject == "" || cb == nil {
		return nil, ErrInvalidArgument
	}

	subj, exists := sp.getSubject(subject)
	if !exists {
		subj = newSubject(sp.cfg.SubjectPuffer, &sp.wg)

		sp.addSubject(subject, subj)

		go subj.dispatchMessage(sp.closeChan)
	}

	sub := newSubscription(subject, sp)

	subj.registerSubscriber(sub, cb)

	return sub, nil
}

/*
Publish

Если subject не существует, возвращает ошибку ErrNoSuchSubject.
*/
func (sp *subPub) Publish(subject string, msg interface{}) error {
	if sp.closed.Load() {
		return ErrSubPubClosed
	}

	if subject == "" || msg == nil {
		return ErrInvalidArgument
	}

	subj, exists := sp.getSubject(subject)
	if !exists {
		return ErrNoSuchSubject
	}

	// Защита от паники при записи в закрытый queue канал
	subj.mu.RLock()
	defer subj.mu.RUnlock()
	if subj.closed {
		return nil
	}

	select {
	case subj.queue <- msg:
		return nil
	case <-sp.closeChan:
		return ErrSubPubClosed
	}
}

func (sp *subPub) Close(ctx context.Context) error {
	sp.mu.Lock()

	if sp.closed.Load() {
		sp.mu.Unlock()
		return ErrSubPubClosed
	}

	sp.closed.Store(true)
	close(sp.closeChan)

	sp.mu.Unlock()

	done := make(chan struct{})
	go func() {
		sp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (sp *subPub) getSubject(subject string) (*subject, bool) {
	sp.mu.RLock()
	subj, exists := sp.subjects[subject]
	sp.mu.RUnlock()

	return subj, exists
}

func (sp *subPub) removeSubject(subject string) {
	sp.mu.Lock()
	delete(sp.subjects, subject)
	sp.mu.Unlock()
}

func (sp *subPub) addSubject(subject string, subj *subject) {
	sp.mu.Lock()
	sp.subjects[subject] = subj
	sp.mu.Unlock()
}
