package subpub

import "sync"

type subscription struct {
	subject string
	subPub  *subPub
	active  bool
	mu      sync.Mutex
}

func newSubscription(subject string, pub *subPub) *subscription {
	return &subscription{
		subject: subject,
		subPub:  pub,
		active:  true,
	}
}

func (sub *subscription) Unsubscribe() {
	sub.mu.Lock()
	defer sub.mu.Unlock()

	if !sub.active {
		return
	}
	sub.active = false

	sp := sub.subPub

	subj, exists := sp.getSubject(sub.subject)
	if !exists {
		return
	}

	subj.mu.Lock()
	delete(subj.subscribers, sub)

	// Удаление subject если нет подписчиков
	if len(subj.subscribers) == 0 {
		sp.removeSubject(sub.subject)

		close(subj.queue)
		subj.closed = true
	}

	subj.mu.Unlock()
}
