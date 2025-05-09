package subpub

import (
	fast_id "VK_task/pkg/fast-id"
	"github.com/google/uuid"
	"log/slog"
	"runtime/debug"
	"sync"
)

type subscription struct {
	id      string
	subject string
	cb      MessageHandler
	queue   chan interface{}

	sp *subPub

	once sync.Once // For single Unsubscribe
}

func newSubscription(subject string, cb MessageHandler, sp *subPub) *subscription {
	var id string
	UUID, err := uuid.NewRandom()
	if err != nil {
		id = fast_id.New()
	} else {
		id = UUID.String()
	}

	return &subscription{
		id:      id,
		subject: subject,
		cb:      cb,
		queue:   make(chan interface{}, sp.cfg.SubscriptionBuffer),
		sp:      sp,
	}
}

func (sub *subscription) Unsubscribe() {
	sub.once.Do(func() {
		sp := sub.sp
		sp.mu.RLock()

		if sp.closed {
			sp.mu.RUnlock()
			return
		}

		subj, exists := sp.subjects[sub.subject]
		sp.mu.RUnlock()

		if !exists {
			return
		}

		if subj.unregisterSubscriber(sub.id) {

			// Удаление subject если нет подписчиков
			sp.removeSubject(sub.subject)
			subj.close()
		}

		close(sub.queue)
	})
}

func (sub *subscription) deliver(msg interface{}) {
	select {
	case sub.queue <- msg:
	default:
		sub.sp.log.Warn("Subscription queue is full",
			slog.String("id", sub.id),
			slog.String("subject", sub.subject),
		)
	}
}

func (sub *subscription) dispatchMessages() {
	for {
		select {
		case msg, ok := <-sub.queue:
			if !ok {
				return
			}
			sub.handleMessage(msg)

		case <-sub.sp.closeChan:
			return
		}
	}
}

func (sub *subscription) handleMessage(msg interface{}) {
	sub.sp.wg.Add(1)
	defer sub.sp.wg.Done()

	defer func() {
		if r := recover(); r != nil {
			sub.sp.log.Error("Panic in message handler",
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())),
				slog.String("id", sub.id),
				slog.String("subject", sub.subject),
			)
		}
	}()

	sub.cb(msg)
}

func (sub *subscription) clear() {
	sub.once.Do(func() {
		close(sub.queue)
	})
}
