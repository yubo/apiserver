package session

import (
	"sync"

	"github.com/yubo/golib/api/errors"
	"github.com/yubo/golib/util"
)

func newMemStorage(cf *Config, opts *Options) (storage, error) {
	st := &memStorage{
		data:   make(map[string]*sessionConn),
		opts:   opts,
		config: cf,
	}

	util.UntilWithTick(st.gc,
		opts.clock.NewTicker(cf.GcInterval.Duration).C(),
		opts.ctx.Done())

	return st, nil
}

type memStorage struct {
	sync.RWMutex
	data map[string]*sessionConn

	opts   *Options
	config *Config
}

func (p *memStorage) all() int {
	p.RLock()
	defer p.RUnlock()
	return len(p.data)
}

func (p *memStorage) get(sid string) (*sessionConn, error) {
	p.RLock()
	defer p.RUnlock()
	s, ok := p.data[sid]
	if !ok {
		return nil, errors.NewNotFound(sid)
	}
	return s, nil
}

func (p *memStorage) insert(s *sessionConn) error {
	p.Lock()
	defer p.Unlock()

	p.data[s.Sid] = s
	return nil
}

func (p *memStorage) del(sid string) error {
	p.Lock()
	defer p.Unlock()

	delete(p.data, sid)
	return nil
}

func (p *memStorage) update(s *sessionConn) error {
	p.Lock()
	defer p.Unlock()

	p.data[s.Sid] = s
	return nil
}

func (p *memStorage) gc() {
	p.Lock()
	defer p.Unlock()

	expiresAt := p.opts.clock.Now().Add(-p.config.MaxIdleTime.Duration)
	keys := []string{}
	for k, v := range p.data {
		if v.UpdatedAt.Before(expiresAt) {
			keys = append(keys, k)
		}
	}

	for _, k := range keys {
		delete(p.data, k)
	}
}
