package subpub

import "context"

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
	SubjectPuffer int
}

func NewSubPub(cfg Config) SubPub {
	cfg.validate()

	return &subPub{
		subjects:  make(map[string]*subject, 8),
		closeChan: make(chan struct{}),
		cfg:       cfg,
	}
}

const defaultSubjectPuffer = 32

func DefaultConfig() Config {
	return Config{
		SubjectPuffer: defaultSubjectPuffer,
	}
}

func (cfg *Config) validate() {
	if cfg.SubjectPuffer <= 0 {
		cfg.SubjectPuffer = defaultSubjectPuffer
	}
}
