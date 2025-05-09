package subpub

import (
	"context"
	"errors"
	"log/slog"
	"sync"
)

type subPub struct {
	subjects map[string]*subject
	mu       sync.RWMutex

	closed    bool // true when subPub is closed
	closeChan chan struct{}

	wg sync.WaitGroup // MessageHandler WaitGroup

	log *slog.Logger
	cfg *Config
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
	if subject == "" || cb == nil {
		return nil, ErrInvalidArgument
	}

	sp.mu.RLock()
	if sp.closed {
		sp.mu.RUnlock()
		return nil, ErrSubPubClosed
	}

	subj, exists := sp.subjects[subject]
	sp.mu.RUnlock()

	if !exists {
		subj = newSubject(sp.cfg.SubjectBuffer)

		sp.addSubject(subject, subj)

		go subj.dispatchMessages(sp.closeChan)
	}

	sub := newSubscription(subject, cb, sp)

	subj.registerSubscriber(sub)

	go sub.dispatchMessages()

	return sub, nil
}

/*
Publish

Если subject не существует, возвращает ошибку ErrNoSuchSubject.
*/
func (sp *subPub) Publish(subject string, msg interface{}) error {
	if subject == "" || msg == nil {
		return ErrInvalidArgument
	}

	sp.mu.RLock()
	if sp.closed {
		sp.mu.RUnlock()
		return ErrSubPubClosed
	}

	subj, exists := sp.subjects[subject]
	sp.mu.RUnlock()
	if !exists {
		return ErrNoSuchSubject
	}

	return subj.publish(msg, sp.closeChan)
}

func (sp *subPub) Close(ctx context.Context) error {
	sp.mu.Lock()

	if sp.closed {
		sp.mu.Unlock()
		return ErrSubPubClosed
	}

	sp.closed = true
	close(sp.closeChan)

	for _, subj := range sp.subjects {
		subj.close()
	}

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

func (sp *subPub) addSubject(subject string, subj *subject) {
	sp.mu.Lock()
	sp.subjects[subject] = subj
	sp.mu.Unlock()
}

func (sp *subPub) removeSubject(subject string) {
	sp.mu.Lock()
	delete(sp.subjects, subject)
	sp.mu.Unlock()
}
