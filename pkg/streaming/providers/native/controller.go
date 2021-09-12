package native

//
//import (
//	"fmt"
//	"sync"
//
//	"github.com/yubo/golib/api/errors"
//)
//
//func newController() *Controller {
//	return &Controller{
//		sessions: make(map[string]*Session),
//	}
//}
//
//type Controller struct {
//	sync.RWMutex
//	sessions map[string]*Session
//}
//
//func (p *Controller) start() error {
//	// TODO: add sessions gc
//	return nil
//}
//
//func (p *Controller) getSession(id string) (*Session, error) {
//	p.RLock()
//	defer p.RUnlock()
//
//	s, ok := p.sessions[id]
//	if !ok {
//		return nil, errors.NewNotFound("session id: " + id)
//	}
//	return s, nil
//}
//
//// TODO
//func (p *Controller) newSession(cf *execConfig) (*Session, error) {
//	p.Lock()
//	defer p.Unlock()
//
//	id, err := p.uniqueID()
//	if err != nil {
//		return nil, err
//	}
//
//	s, err := NewSession(cf)
//	if err != nil {
//		return nil, err
//	}
//
//	p.sessions[id] = s
//
//	return s, nil
//}
//
//func (p *Controller) checkSessionStatus(id string) (*Session, error) {
//	session, err := p.getSession(id)
//	if err != nil {
//		return nil, err
//	}
//	if !session.Status().Running {
//		return nil, fmt.Errorf("session not running (%s)", id)
//	}
//	return session, nil
//
//}
