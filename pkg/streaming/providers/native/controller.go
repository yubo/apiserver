package native

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"sync"

	"github.com/yubo/golib/api/errors"
)

var (
	idLen = 10
)

func newController() *Controller {
	return &Controller{
		sessions: make(map[string]*Session),
	}
}

type Controller struct {
	sync.RWMutex
	sessions map[string]*Session
}

func (p *Controller) start() error {
	// TODO: add sessions gc
	return nil
}

func (p *Controller) getSession(id string) (*Session, error) {
	p.RLock()
	defer p.RUnlock()

	s, ok := p.sessions[id]
	if !ok {
		return nil, errors.NewNotFound("session id: " + id)
	}
	return s, nil
}

// unsafe
func (p *Controller) uniqueID() (string, error) {
	const maxTries = 10
	// Number of bytes to be tokenLen when base64 encoded.
	idSize := math.Ceil(float64(idLen) * 6 / 8)
	rawId := make([]byte, int(idSize))
	for i := 0; i < maxTries; i++ {
		if _, err := rand.Read(rawId); err != nil {
			return "", err
		}
		encoded := base64.RawURLEncoding.EncodeToString(rawId)
		id := encoded[:idLen]
		// If it's unique, return it. Otherwise retry.
		if _, exists := p.sessions[encoded]; !exists {
			return id, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique id")
}

// TODO
func (p *Controller) newSession(cf *execConfig) (*Session, error) {
	p.Lock()
	defer p.Unlock()

	id, err := p.uniqueID()
	if err != nil {
		return nil, err
	}

	s, err := NewSession(cf)
	if err != nil {
		return nil, err
	}

	p.sessions[id] = s

	return s, nil
}

func (p *Controller) checkSessionStatus(id string) (*Session, error) {
	session, err := p.getSession(id)
	if err != nil {
		return nil, err
	}
	if !session.Status().Running {
		return nil, fmt.Errorf("session not running (%s)", id)
	}
	return session, nil

}
