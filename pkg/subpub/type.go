package subpub

import (
	"context"
	"log/slog"
)

type MessageHandler func(msg interface{})

type Subscription interface {
	Unsubscribe()
}

type SubPub interface {
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	Publish(subject string, msg interface{}) error
	Close(ctx context.Context) error
}

type Config struct {
	SubjectBuffer      int
	SubscriptionBuffer int
}

func NewSubPub(cfg *Config, log *slog.Logger) SubPub {
	cfg.validate()

	return &subPub{
		subjects:  make(map[string]*subject, 8),
		closeChan: make(chan struct{}),
		log:       log,
		cfg:       cfg,
	}
}

const (
	defaultSubjectPuffer      = 16
	defaultSubscriptionPuffer = 64
)

func DefaultConfig() *Config {
	return &Config{
		SubjectBuffer:      defaultSubjectPuffer,
		SubscriptionBuffer: defaultSubscriptionPuffer,
	}
}

func NewConfig(subjectBuffer, subscriptionBuffer int) *Config {
	return &Config{
		SubjectBuffer:      subjectBuffer,
		SubscriptionBuffer: subscriptionBuffer,
	}
}

func (cfg *Config) validate() {
	if cfg.SubjectBuffer <= 0 {
		cfg.SubjectBuffer = defaultSubjectPuffer
	}
	if cfg.SubscriptionBuffer <= 0 {
		cfg.SubscriptionBuffer = defaultSubscriptionPuffer
	}
}
